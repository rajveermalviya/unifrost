package gochan

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
	"github.com/rajveermalviya/gochan/drivers"
	"gocloud.dev/gcerrors"
	"gocloud.dev/pubsub"
)

// Streamer is a top-level struct that will handle all the clients and subscriptions.
// It implements the http.Handler interface for easy working with the server.
type Streamer struct {
	subClient     drivers.SubscriberClient
	mu            sync.RWMutex
	clients       map[string]*Client
	clientTimeout time.Duration
}

// ConfigStreamer struct is used to configure the Streamer.
type ConfigStreamer struct {
	// ClientTimeout is the timeout duration to wait after cleaning the resources
	// of a disconnected client.
	// Default is a Minute.
	ClientTimeout time.Duration
	// DriverConfig is config related to the specified driver
	// See drivers package for more information.
	DriverConfig interface{}
	// Driver is the pubsub vendor driver to use.
	Driver drivers.Driver
}

var (
	// ErrNoClient is returned if the client-id is not registered in the streamer.
	ErrNoClient = errors.New("streamer: Client doesn't exists")
)

// NewStreamer is the construtor for Streamer struct
func NewStreamer(ctx context.Context, c *ConfigStreamer) (*Streamer, error) {
	subClient, err := drivers.NewSubscriberClient(ctx, c.Driver, c.DriverConfig)
	if err != nil {
		return nil, err
	}

	if c.ClientTimeout == time.Duration(0) {
		c.ClientTimeout = time.Minute
	}

	return &Streamer{
		subClient:     subClient,
		clients:       make(map[string]*Client),
		clientTimeout: c.ClientTimeout,
	}, nil
}

// ServeHTTP is the http handler for eventsource.
// For connecting query parameter 'id' is required i.e client-id.
func (streamer *Streamer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Make request SSE compatible.
	h := w.Header()
	h.Set("content-type", "text/event-stream")
	h.Set("cache-control", "no-cache, no-store, must-revalidate, pre-check=0, post-check=0")
	h.Set("connection", "keep-alive")
	h.Set("access-control-allow-origin", "*")
	h.Set("server", "gochan")

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
		http.Error(w, "Invalid client ID", http.StatusBadRequest)
		return
	}

	client.mu.RLock()
	client.timer.Stop()
	client.mu.RUnlock()

	log.Printf("Client %s connected", clientID)

	d, _ := json.Marshal(
		map[string]interface{}{
			"topic": "/system/config",
			"payload": map[string]interface{}{
				"config": map[string]interface{}{
					"client_id":             clientID,
					"client_timeout_millis": streamer.clientTimeout.Milliseconds(),
				},
			},
		},
	)
	fmt.Fprintf(w, "data: %s\n\n", d)
	flusher.Flush()

	d, _ = json.Marshal(
		map[string]interface{}{
			"topic": "/system/subscriptions",
			"payload": map[string]interface{}{
				"subscriptions": client.GetTopics(ctx),
			},
		},
	)
	fmt.Fprintf(w, "data: %s\n\n", d)
	flusher.Flush()

	go func() {
		<-ctx.Done()

		log.Printf("Client %s disconnected", clientID)

		timer := time.NewTimer(streamer.clientTimeout)

		client.mu.Lock()
		client.timer = timer
		client.mu.Unlock()

		<-timer.C // FIXME: (goroutine stays running) (#need-help)
		log.Printf("Disconnected client %s timed out, cleaning...", clientID)
		streamer.RemoveClient(context.Background(), clientID)
	}()

	for data := range client.writeChannel {
		client := streamer.GetClient(ctx, clientID)
		// If client is nil then client is closed.
		if client == nil {
			flusher.Flush()
			return
		}

		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

// Subscribe method subscribes the specified client to the specified topic.
// If specified client doesn't exists ErrNoClient error is returned.
func (streamer *Streamer) Subscribe(ctx context.Context, clientID string, topic string) error {
	client := streamer.GetClient(ctx, clientID)
	if client == nil {
		return ErrNoClient
	}

	_, ok := client.topics[topic]
	if ok {
		return nil
	}

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
					break
				}

				log.Println("Error: Cannot receive message:", err)

				sub.Shutdown(ctx)
				client.mu.Lock()
				delete(client.topics, topic)
				client.mu.Unlock()
				break
			}

			d, _ := json.Marshal(map[string]string{"topic": topic, "payload": string(msg.Body)})

			client.writeChannel <- d
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

	client.mu.Lock()
	defer client.mu.Unlock()

	sub, ok := client.topics[topic]
	if !ok {
		return nil
	}

	delete(client.topics, topic)

	log.Printf("Client %s unsubscribed for topic %s", clientID, topic)
	return sub.Shutdown(ctx)
}

// NewClient method creates a new client with an autogenerated client-id.
func (streamer *Streamer) NewClient(ctx context.Context) (*Client, error) {
	streamer.mu.Lock()
	defer streamer.mu.Unlock()

	client := &Client{
		ID:           uuid.New().String(),
		writeChannel: make(chan []byte, 10),
		topics:       make(map[string]*pubsub.Subscription),
		timer:        time.NewTimer(streamer.clientTimeout),
	}

	streamer.clients[client.ID] = client

	return client, nil
}

// NewCustomClient method creates a new client with the specified client-id.
func (streamer *Streamer) NewCustomClient(ctx context.Context, clientID string) (*Client, error) {
	streamer.mu.Lock()
	defer streamer.mu.Unlock()

	client := &Client{
		ID:           clientID,
		writeChannel: make(chan []byte, 10),
		topics:       make(map[string]*pubsub.Subscription),
		timer:        time.NewTimer(streamer.clientTimeout),
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

	for _, client := range streamer.clients {
		go client.Close(ctx)
	}

	return streamer.subClient.Close(ctx)
}
