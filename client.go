package gochan

import (
	"context"
	"log"
	"sync"
	"time"

	"gocloud.dev/pubsub"
)

type Client struct {
	ID           string
	writeChannel chan []byte
	topics       map[string]*pubsub.Subscription
	mu           sync.RWMutex
	timer        *time.Timer
}

func (client *Client) Topics(ctx context.Context) []string {
	client.mu.RLock()
	defer client.mu.RUnlock()

	keys := make([]string, 0, len(client.topics))
	for key := range client.topics {
		keys = append(keys, key)
	}

	return keys
}

func (client *Client) TotalTopics(ctx context.Context) int {
	client.mu.RLock()
	defer client.mu.RUnlock()
	return len(client.topics)
}

func (client *Client) Close(ctx context.Context) error {
	client.mu.Lock()
	defer client.mu.Unlock()

	log.Printf("Closing client %s", client.ID)

	for key, topic := range client.topics {
		if err := topic.Shutdown(ctx); err != nil {
			log.Println("streamer: ", err)
		}
		delete(client.topics, key)
	}

	time.Sleep(2 * time.Second)

	close(client.writeChannel)
	return nil
}
