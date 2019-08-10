package gochan

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"gocloud.dev/pubsub"
)

type subscriptionsBroker struct {
	subs           map[string]*subscription
	mu             sync.RWMutex
	Prefix         string
	PostfixOptions string
}

type subscription struct {
	clients      map[string]*client
	subscription *pubsub.Subscription
	topic        string
	mu           sync.RWMutex
}

type client struct {
	writeChannel chan []byte
	sessID       string
	topics       map[string]bool
	mu           sync.RWMutex
}

func (b *subscriptionsBroker) SubscribeClient(c *client, topic string) error {
	// If the subscription already exists, then just add the client
	// So that when new event is recieved by the subscription, the message
	// will be passed to the client's write channel.

	b.mu.RLock()
	sub, ok := b.subs[topic]
	b.mu.RUnlock()
	if ok {
		sub.mu.Lock()
		sub.clients[c.sessID] = c
		sub.mu.Unlock()

		c.mu.Lock()
		c.topics[topic] = true
		c.mu.Unlock()

		sub.mu.RLock()
		log.Printf("Added new subscriber to topic '%v', total subscribers: %v\n", topic, len(sub.clients))
		sub.mu.RUnlock()

		return nil
	}

	// Use background context because subscriptions needs to stay forever, until shutdown.
	ctx := context.Background()

	// When the subscription does not exists, open a new subscription.
	s, err := pubsub.OpenSubscription(ctx, b.Prefix+topic+b.PostfixOptions)
	if err != nil {
		return err
	}

	newSub := &subscription{
		subscription: s,
		topic:        topic,
		clients: map[string]*client{
			c.sessID: c,
		},
	}

	b.mu.Lock()
	b.subs[topic] = newSub
	b.mu.Unlock()

	c.mu.Lock()
	c.topics[topic] = true
	c.mu.Unlock()

	b.mu.RLock()
	log.Printf("Subscribed to new topic '%v', total subscriptions: %v\n", topic, len(b.subs))
	b.mu.RUnlock()

	log.Printf("Added new subscriber to topic '%v', total subscribers: %v\n", topic, len(newSub.clients))

	// Start a new goroutine that will run forever until the subscription is open.
	go b.streamMessages(ctx, topic)

	return nil
}

func (b *subscriptionsBroker) streamMessages(ctx context.Context, topic string) {
	log.Println("Streaming messages for topic:", topic)

	for {
		// Check if the subscription exists i.e if it is still open,
		// If the subscription doesn't exists the subscription is shutdown,
		// Then break the loop and end the goroutine.
		b.mu.RLock()
		sub, ok := b.subs[topic]
		b.mu.RUnlock()
		if !ok {
			break
		}

		// If there are no clients subscribed to the topic, then shutdown the subscription.
		sub.mu.RLock()
		if len(sub.clients) == 0 {
			log.Printf("No subscribers for topic '%v', shuting down subscription of topic.\n", topic)
			if err := sub.subscription.Shutdown(ctx); err != nil {
				if err != context.Canceled {
					log.Println("Error: Cannot recieve message:", err)
				}
			}
			b.mu.Lock()
			delete(b.subs, topic)
			b.mu.Unlock()
			sub.mu.RUnlock()
			break
		}
		sub.mu.RUnlock()

		// Recieve new message from the subscription.
		sub.mu.RLock()
		msg, err := sub.subscription.Receive(ctx)
		if err != nil {
			if err != context.Canceled {
				log.Println("Error: Cannot recieve message:", err)

				for _, c := range sub.clients {
					c.mu.Lock()
					delete(c.topics, topic)
					c.mu.Unlock()

					d, _ := json.Marshal(map[string]interface{}{
						"error": map[string]string{
							"code":    "subscription-removed",
							"message": fmt.Sprintf("Subscription forcefully removed of topic %v", topic),
						},
					})
					c.writeChannel <- d
				}

				b.mu.Lock()
				delete(b.subs, topic)
				b.mu.Unlock()
			}
			sub.mu.RUnlock()
			break
		}
		sub.mu.RUnlock()

		// Loop over all the clients that are subscribed to this topic and
		// write the recieved message to their writeChannels
		sub.mu.RLock()
		for _, client := range sub.clients {
			d, err := json.Marshal(map[string]string{"topic": topic, "payload": string(msg.Body)})
			if err != nil {
				log.Println("Error: Cannot marshal the message:", err)
			}
			client.writeChannel <- d
		}
		sub.mu.RUnlock()

		msg.Ack()

	}
}

func (b *subscriptionsBroker) UnsubscribeClient(ctx context.Context, c *client, topic string) {
	b.mu.RLock()
	sub, ok := b.subs[topic]
	b.mu.RUnlock()
	if !ok {
		log.Printf("Error: subscription for topic '%v' does not exists.\n", topic)
		return
	}

	sub.mu.Lock()
	delete(sub.clients, c.sessID)
	sub.mu.Unlock()

	c.mu.Lock()
	delete(c.topics, topic)
	c.mu.Unlock()

	sub.mu.RLock()
	log.Printf("Removed subscriber  from topic '%v', total subscribers: %v\n", topic, len(sub.clients))
	sub.mu.RUnlock()

}
