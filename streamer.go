package gochan

import (
	"context"
	"encoding/json"
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

type Streamer struct {
	subClient     drivers.SubscriberClient
	mu            sync.RWMutex
	clients       map[string]*Client
	clientTimeout time.Duration
}

type ConfigStreamer struct {
	ClientTimeout time.Duration
	DriverConfig  interface{}
	Driver        drivers.Driver
}

var (
	ErrNoClient = fmt.Errorf("Client doesn't exists")
)

func NewStreamer(ctx context.Context, c *ConfigStreamer) (*Streamer, error) {
	subClient, err := drivers.NewSubscriberClient(ctx, c.Driver, c.DriverConfig)
	if err != nil {
		return nil, err
	}

	if c.ClientTimeout == time.Duration(0) {
		c.ClientTimeout = time.Minute
	}

	log.Println("Starting new streamer\n Config:", c)

	streamer := &Streamer{
		subClient:     subClient,
		clients:       make(map[string]*Client),
		clientTimeout: c.ClientTimeout,
	}

	return streamer, nil
}

func (streamer *Streamer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Make request SSE compatible.
	h := w.Header()
	h.Set("content-type", "text/event-stream")
	h.Set("cache-control", "no-cache, no-store, must-revalidate, pre-check=0, post-check=0")
	h.Set("connection", "keep-alive")
	h.Set("access-control-allow-origin", "*")
	h.Set("server", "gochan")

	clientID := r.URL.Query().Get("id")
	if len(clientID) == 0 {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	streamer.mu.RLock()
	client := streamer.clients[clientID]
	streamer.mu.RLock()

	client.mu.RLock()
	client.timer.Stop()
	client.mu.RUnlock()

	log.Printf("Client %s connected", clientID)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	go func() {
		ctx := r.Context()
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
		streamer.mu.RLock()
		if _, ok := streamer.clients[clientID]; !ok {
			streamer.mu.RUnlock()
			return
		}
		streamer.mu.RUnlock()

		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

func (streamer *Streamer) Subscribe(ctx context.Context, clientID string, topic string) error {
	streamer.mu.RLock()
	client, ok := streamer.clients[clientID]
	streamer.mu.RUnlock()
	if !ok {
		return ErrNoClient
	}

	_, ok = client.topics[topic]
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
		defer log.Printf("Client %s stopped recieving messages for topic %s", clientID, topic)

		for {
			msg, err := sub.Receive(ctx)
			if err != nil {
				if gcerrors.Code(err) == gcerrors.FailedPrecondition {
					break
				}

				log.Println("Error: Cannot recieve message:", err)

				sub.Shutdown(ctx)
				client.mu.Lock()
				delete(client.topics, topic)
				client.mu.Unlock()
				break
			}

			d, err := json.Marshal(map[string]string{"topic": topic, "payload": string(msg.Body)})

			client.writeChannel <- d
			msg.Ack()
		}

	}()

	return nil
}

func (streamer *Streamer) Unsubscribe(ctx context.Context, clientID string, topic string) error {
	streamer.mu.RLock()
	client, ok := streamer.clients[clientID]
	streamer.mu.RUnlock()
	if !ok {
		return ErrNoClient
	}

	client.mu.RLock()
	defer client.mu.RUnlock()

	_, ok = client.topics[topic]
	if !ok {
		return nil
	}

	defer log.Printf("Client %s unsubscribed for topic %s", clientID, topic)

	return client.topics[topic].Shutdown(ctx)
}

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

func (streamer *Streamer) GetClient(ctx context.Context, clientID string) *Client {
	streamer.mu.RLock()
	defer streamer.mu.RUnlock()

	return streamer.clients[clientID]
}

func (streamer *Streamer) TotalClients(ctx context.Context) int {
	streamer.mu.RLock()
	defer streamer.mu.RUnlock()

	return len(streamer.clients)
}

func (streamer *Streamer) RemoveClient(ctx context.Context, clientID string) error {
	streamer.mu.RLock()
	client, ok := streamer.clients[clientID]
	streamer.mu.RUnlock()
	if !ok {
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

func (streamer *Streamer) Close(ctx context.Context) error {
	return streamer.subClient.Close(ctx)
}
