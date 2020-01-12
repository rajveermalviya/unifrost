package unifrost

import (
	"context"
	"log"
	"sync"
	"time"

	"gocloud.dev/pubsub"
)

// Client manages all the topic subscriptions.
type Client struct {
	ID             string
	messageChannel chan message
	topics         map[string]*pubsub.Subscription
	mu             sync.RWMutex
	ttlTimer       *time.Timer
	connected      bool
	timerStopped   chan bool
}

type message struct {
	event   string
	payload []byte
}

// GetTopics returns a slice of all the topics the client is subscribed to.
func (client *Client) GetTopics(ctx context.Context) []string {
	client.mu.RLock()
	defer client.mu.RUnlock()

	topics := make([]string, 0, len(client.topics))
	for key := range client.topics {
		topics = append(topics, key)
	}

	return topics
}

// TotalTopics returns the number of topics the client is subscribed to.
func (client *Client) TotalTopics(ctx context.Context) int {
	client.mu.RLock()
	defer client.mu.RUnlock()
	return len(client.topics)
}

// Connected reports whether client is connected to the server.
func (client *Client) Connected() bool {
	client.mu.RLock()
	defer client.mu.RUnlock()
	return client.connected
}

// Close closes the client and shutdowns all the subscriptions.
func (client *Client) Close(ctx context.Context) error {
	client.mu.Lock()
	defer client.mu.Unlock()

	log.Printf("Closing client %s", client.ID)

	if len(client.topics) > 0 {
		// Wait for http connection to terminate
		defer time.Sleep(time.Second)
	}

	// loop over all the topics and shut them down
	for key, sub := range client.topics {
		if err := sub.Shutdown(ctx); err != nil {
			log.Println("streamer: ", err)
		}
		delete(client.topics, key)
	}

	close(client.messageChannel)
	return nil
}
