package gochan

import (
	"context"
	"encoding/json"
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
		totalSubscribers := len(sub.clients)
		sub.mu.RUnlock()

		log.Printf("Added new subscriber to topic '%v', total subscribers: %v\n", topic, totalSubscribers)
		return nil
	}

	// Use background context because subscriptions need to stay forever, until shutdown.
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
	totalSubscriptions := len(b.subs)
	b.mu.RUnlock()

	log.Printf("Subscribed to new topic '%v', total subscriptions: %v\n", topic, totalSubscriptions)

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
			log.Printf("Error: subscription for topic '%v' does not exists.\n", topic)
			break
		}

		// Recieve new message from the subscription.
		sub.mu.RLock()
		msg, err := sub.subscription.Receive(ctx)
		sub.mu.RUnlock()
		if err != nil {
			if err != context.Canceled {
				log.Println("Error: Cannot recieve message:", err)
				b.mu.Lock()
				delete(b.subs, topic)
				b.mu.Unlock()
			}
			break
		}

		// Loop over all the clients that are subscribed to this topic and
		// write the recieved message to their writeChannels
		sub.mu.RLock()
		clients := sub.clients
		sub.mu.RUnlock()

		for _, client := range clients {
			d, err := json.Marshal(map[string]string{"topic": topic, "payload": string(msg.Body)})
			if err != nil {
				log.Println("Error: Cannot marshal the message:", err)
			}
			client.writeChannel <- d
		}

		msg.Ack()

	}
}

func (b *subscriptionsBroker) UnsubscribeClient(ctx context.Context, c *client, topic string) error {
	b.mu.RLock()
	sub, ok := b.subs[topic]
	b.mu.RUnlock()
	if !ok {
		log.Printf("Error: subscription for topic '%v' does not exists.\n", topic)
		return nil
	}

	sub.mu.Lock()
	delete(sub.clients, c.sessID)
	sub.mu.Unlock()

	c.mu.Lock()
	delete(c.topics, topic)
	c.mu.Unlock()

	sub.mu.RLock()
	totalSubs := len(sub.clients)
	sub.mu.RUnlock()

	log.Printf("Removed subscriber  from topic '%v', total subscribers: %v\n", topic, totalSubs)

	// If there are no clients subscribed to the topic, then shutdown the subscription.
	if totalSubs == 0 {
		log.Printf("No subscribers for topic '%v', shuting down subscription of topic.\n", topic)
		sub.mu.RLock()
		err := sub.subscription.Shutdown(ctx)
		sub.mu.RUnlock()
		if err != nil {
			if err != context.Canceled {
				return err
			}
		}
		b.mu.Lock()
		delete(b.subs, topic)
		b.mu.Unlock()
	}

	return nil
}
