package unifrost

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rajveermalviya/unifrost/drivers"
	"gocloud.dev/gcerrors"
	"gocloud.dev/pubsub"
)

// Streamer is a top-level struct that will handle all the clients and subscriptions.
// It implements the http.Handler interface for easy working with the server.
type Streamer struct {
	subClient       drivers.SubscriberClient
	mu              sync.RWMutex
	clients         map[string]*Client
	clientTTLMillis time.Duration
}

var (
	// ErrNoClient is returned if the client-id is not registered in the streamer.
	ErrNoClient = errors.New("streamer: Client doesn't exists")
)

// Options is a self-refrential function for configuration
type Options func(*Streamer) error

// NewStreamer is the construtor for Streamer struct
func NewStreamer(subClient drivers.SubscriberClient, options ...Options) (*Streamer, error) {

	s := &Streamer{
		subClient:       subClient,
		clients:         make(map[string]*Client),
		clientTTLMillis: time.Duration(time.Minute.Milliseconds()),
	}

	for _, option := range options {
		if err := option(s); err != nil {
			return nil, err
		}
	}

	return s, nil
}

// ClientTTL is a config function that is used to set the client's TTL
// default TTL is 1 minute
func ClientTTL(t time.Duration) Options {
	return func(s *Streamer) error {
		s.clientTTLMillis = time.Duration(t.Milliseconds())
		return nil
	}
}

// ServeHTTP is the http handler for eventsource.
// For connecting query parameter 'id' is required i.e client-id.
func (streamer *Streamer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Make request SSE compatible.
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache, no-store, must-revalidate, pre-check=0, post-check=0")
	h.Set("Connection", "keep-alive")
	h.Set("Server", "unifrost")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	clientID := r.URL.Query().Get("id")
	if len(clientID) == 0 {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	client := streamer.GetClient(ctx, clientID)
	if client == nil {
		http.Error(w, "Invalid client ID", http.StatusUnauthorized)
		return
	}

	// Stop the timeout timer if it is running.
	client.mu.RLock()
	client.ttlTimer.Stop()
	client.mu.RUnlock()

	log.Printf("Client %s connected", clientID)

	// First message:
	// client_id & client_ttl_millis
	d, _ := json.Marshal(
		map[string]interface{}{
			"config": map[string]interface{}{
				"client_id":         clientID,
				"client_ttl_millis": streamer.clientTTLMillis,
			},
			"subscriptions": client.GetTopics(ctx),
		},
	)
	fmt.Fprint(w, "event: message\n")
	fmt.Fprintf(w, "data: %s\n\n", d)
	flusher.Flush()

	go func() {
		<-ctx.Done()
		log.Printf("Client %s disconnected", clientID)

		// When client gets disconnected start the timer
		timer := time.NewTimer(streamer.clientTTLMillis * time.Millisecond)

		client.mu.Lock()
		client.ttlTimer = timer
		client.mu.Unlock()

		// cleanup the resources if the timer is expired
		if _, ok := <-timer.C; ok {
			log.Printf("Disconnected client %s timed out, cleaning...", clientID)
			streamer.RemoveClient(context.Background(), clientID)
		}
	}()

	for data := range client.messageChannel {
		client := streamer.GetClient(ctx, clientID)
		if client == nil {
			// If client is nil then client is closed and so complete the request.
			break
		}

		fmt.Fprintf(w, "event: %s\n", data.event)
		fmt.Fprintf(w, "data: %s\n\n", data.payload)
		flusher.Flush()
	}

	fmt.Fprint(w, "event: message\n")
	fmt.Fprintf(w, "data: %s\n\n", "Client closed")
	flusher.Flush()
}

// Subscribe method subscribes the specified client to the specified topic.
// If specified client doesn't exists ErrNoClient error is returned.
func (streamer *Streamer) Subscribe(ctx context.Context, clientID string, topic string) error {
	client := streamer.GetClient(ctx, clientID)
	if client == nil {
		return ErrNoClient
	}

	client.mu.RLock()
	_, ok := client.topics[topic]
	client.mu.RUnlock()
	if ok {
		return nil
	}

	// subscribe to the specified topic
	sub, err := streamer.subClient.Subscribe(ctx, topic)
	if err != nil {
		return err
	}

	client.mu.Lock()
	client.topics[topic] = sub
	client.mu.Unlock()

	log.Printf("Client %s subscribed for topic %s", clientID, topic)

	go func() {
		ctx := context.Background()
		defer log.Printf("Client %s stopped receiving messages for topic %s", clientID, topic)

		for {
			msg, err := sub.Receive(ctx)
			if err != nil {
				if gcerrors.Code(err) == gcerrors.FailedPrecondition {
					// if the error is FailedPrecondition it means that the subscription
					// is shutdown so break the loop and exit the goroutine.
					break
				}

				// if error is unknown then shutdown the subscription and
				// remove the subscription from the client
				log.Println("Error: Cannot receive message:", err)

				d, _ := json.Marshal(map[string]interface{}{
					"error": map[string]string{
						"topic":   topic,
						"code":    "subscription-failure",
						"message": fmt.Sprint("Cannot receive message from subscription, closing subscription"),
					},
				})
				client.messageChannel <- message{event: "message", payload: d}

				sub.Shutdown(ctx)

				client.mu.Lock()
				delete(client.topics, topic)
				client.mu.Unlock()
				break
			}

			client.messageChannel <- message{event: topic, payload: msg.Body}
			msg.Ack()
		}
	}()

	return nil
}

// Unsubscribe method unsubscribes the specified client to the specified topic
// and shutdowns the subscription.
// If specified client doesn't exists ErrNoClient error is returned.
func (streamer *Streamer) Unsubscribe(ctx context.Context, clientID string, topic string) error {
	client := streamer.GetClient(ctx, clientID)
	if client == nil {
		return ErrNoClient
	}

	client.mu.RLock()
	sub, ok := client.topics[topic]
	client.mu.RUnlock()
	if !ok {
		return nil
	}

	client.mu.Lock()
	delete(client.topics, topic)
	client.mu.Unlock()

	log.Printf("Client %s unsubscribed for topic %s", clientID, topic)
	return sub.Shutdown(ctx)
}

// NewClient method creates a new client with an autogenerated client-id.
func (streamer *Streamer) NewClient(ctx context.Context) (*Client, error) {
	streamer.mu.Lock()
	defer streamer.mu.Unlock()

	client := &Client{
		ID:             uuid.New().String(),
		messageChannel: make(chan message, 10),
		topics:         make(map[string]*pubsub.Subscription),
		ttlTimer:       time.NewTimer(streamer.clientTTLMillis * time.Millisecond),
	}

	streamer.clients[client.ID] = client

	return client, nil
}

// NewCustomClient method creates a new client with the specified clientID.
func (streamer *Streamer) NewCustomClient(ctx context.Context, clientID string) (*Client, error) {
	streamer.mu.Lock()
	defer streamer.mu.Unlock()

	client := &Client{
		ID:             clientID,
		messageChannel: make(chan message, 10),
		topics:         make(map[string]*pubsub.Subscription),
		ttlTimer:       time.NewTimer(streamer.clientTTLMillis * time.Millisecond),
	}

	streamer.clients[client.ID] = client

	return client, nil
}

// GetClient method returns the client instance of the specified clientID.
func (streamer *Streamer) GetClient(ctx context.Context, clientID string) *Client {
	streamer.mu.RLock()
	defer streamer.mu.RUnlock()

	return streamer.clients[clientID]
}

// TotalClients method returns the number of client connected to the streamer.
func (streamer *Streamer) TotalClients(ctx context.Context) int {
	streamer.mu.RLock()
	defer streamer.mu.RUnlock()

	return len(streamer.clients)
}

// RemoveClient method removes the client from streamer and closes it.
func (streamer *Streamer) RemoveClient(ctx context.Context, clientID string) error {
	client := streamer.GetClient(ctx, clientID)
	if client == nil {
		return ErrNoClient
	}

	if err := client.Close(ctx); err != nil {
		return err
	}

	streamer.mu.Lock()
	delete(streamer.clients, clientID)
	streamer.mu.Unlock()

	return nil
}

// Close method closes the streamer and also closes all the connected clients.
func (streamer *Streamer) Close(ctx context.Context) error {
	log.Println("Closing streamer")
	streamer.mu.RLock()
	defer streamer.mu.RUnlock()

	if len(streamer.clients) > 0 {
		defer time.Sleep(time.Second)
	}

	wg := sync.WaitGroup{}
	// loop over all the client and close all the clients
	for _, client := range streamer.clients {
		wg.Add(1)
		go func(c *Client) {
			c.Close(ctx)
			wg.Done()
		}(client)
	}

	wg.Wait()

	return streamer.subClient.Close(ctx)
}
