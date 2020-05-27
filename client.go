// Copyright 2019-2020 Rajesh Malviya
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

const (
	// ERROR .
	ERROR = iota
	// EVENT .
	EVENT
)

type message struct {
	event       string
	messageType int
	payload     string
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
