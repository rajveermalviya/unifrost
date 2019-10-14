package gochan

import (
	"context"
	"log"
	"sync"
	"time"

	"gocloud.dev/pubsub"
)

// Client is a top-level struct that manages all the topics.
type Client struct {
	ID             string
	messageChannel chan message
	topics         map[string]*pubsub.Subscription
	mu             sync.RWMutex
	ttlTimer       *time.Timer
}

type message struct {
	topic   string
	payload []byte
}

// GetTopics method returns an array of all the topics client is subscribed to.
func (client *Client) GetTopics(ctx context.Context) []string {
	client.mu.RLock()
	defer client.mu.RUnlock()

	keys := make([]string, 0, len(client.topics))
	for key := range client.topics {
		keys = append(keys, key)
	}

	return keys
}

// TotalTopics method returns the number of topics the client is subscribed to.
func (client *Client) TotalTopics(ctx context.Context) int {
	client.mu.RLock()
	defer client.mu.RUnlock()
	return len(client.topics)
}

// Close method closes the client and shutdowns all the subscriptions.
func (client *Client) Close(ctx context.Context) error {
	client.mu.Lock()
	defer client.mu.Unlock()

	log.Printf("Closing client %s", client.ID)

	if len(client.topics) > 0 {
		defer time.Sleep(time.Second)
	}

	// loop over all the topics and shut them down
	for key, topic := range client.topics {
		if err := topic.Shutdown(ctx); err != nil {
			log.Println("streamer: ", err)
		}
		delete(client.topics, key)
	}

	close(client.messageChannel)
	return nil
}
