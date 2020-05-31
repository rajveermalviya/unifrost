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
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/google/uuid"

	"github.com/unifrost/unifrost/drivers"
)

// StreamHandler handles all the consumers and subscriptions.
// It implements the http.Handler interface for easy embedding with any API server.
type StreamHandler struct {
	subClient     drivers.SubscriberClient
	consumers     map[string]*Consumer
	consumerTTL   time.Duration
	subscriptions map[string]*subscription

	addConsumerChan          chan consumerMessageg
	closeConsumerChan        chan consumerMessageg
	removeConsumerTopicChan  chan consumerTopicMessage
	addSubscriptionChan      chan subscriptionMessage
	removeSubscriptionChan   chan subscriptionMessage
	shutdownSubscriptionChan chan topicMessage
}

type consumerMessageg struct {
	c   *Consumer
	ack chan struct{}
}

type topicMessage struct {
	topic string
	ack   chan struct{}
}

type consumerTopicMessage struct {
	c     *Consumer
	topic string
	ack   chan struct{}
}

type subscriptionMessage struct {
	topic      string
	consumerID string
	errChan    chan error
}

type subscription struct {
	topic        string
	consumers    map[string]*Consumer
	shutdownChan chan struct{}
}

// Consumer manages all the topic subscriptions.
type Consumer struct {
	ID             string
	messageChannel chan message
	topics         map[string]struct{}
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

var (
	// ErrConsumerNotFound is returned if the consumer-id is not registered in the StreamHandler.
	ErrConsumerNotFound = errors.New("stream handler: consumer doesn't exists")
)

// Option is a self-refrential function for configuration parameters
type Option func(*StreamHandler) error

// NewStreamHandler returns *unifrost.StreamHandler, handles all the consumers and subscriptions.
//
// Additional configuration options can be added with unifrost.Option functions.
func NewStreamHandler(ctx context.Context, subClient drivers.SubscriberClient, options ...Option) (*StreamHandler, error) {

	s := &StreamHandler{
		subClient:                subClient,
		consumers:                map[string]*Consumer{},
		subscriptions:            map[string]*subscription{},
		consumerTTL:              time.Duration(time.Minute),
		addConsumerChan:          make(chan consumerMessageg),
		closeConsumerChan:        make(chan consumerMessageg),
		removeConsumerTopicChan:  make(chan consumerTopicMessage),
		addSubscriptionChan:      make(chan subscriptionMessage),
		removeSubscriptionChan:   make(chan subscriptionMessage),
		shutdownSubscriptionChan: make(chan topicMessage),
	}

	for _, option := range options {
		if err := option(s); err != nil {
			return nil, err
		}
	}

	go s.managementService(ctx)

	return s, nil
}

// ConsumerTTL is an option that is used to set the consumer's TTL
// default TTL is 1 minute
func ConsumerTTL(t time.Duration) Option {
	return func(s *StreamHandler) error {
		s.consumerTTL = t
		return nil
	}
}

func (s *StreamHandler) managementService(ctx context.Context) {
	for {
		select {
		case m := <-s.addConsumerChan:
			s.consumers[m.c.ID] = m.c
			m.ack <- struct{}{}

		case m := <-s.closeConsumerChan:
			s.closeConsumer(ctx, m.c)
			m.ack <- struct{}{}

		case m := <-s.removeConsumerTopicChan:
			delete(m.c.topics, m.topic)
			m.ack <- struct{}{}

		case m := <-s.addSubscriptionChan:
			m.errChan <- s.subscribe(ctx, m.consumerID, m.topic)

		case m := <-s.removeSubscriptionChan:
			m.errChan <- s.unsubscribe(ctx, m.consumerID, m.topic)

		case m := <-s.shutdownSubscriptionChan:
			s.subscriptions[m.topic] = nil
			delete(s.subscriptions, m.topic)
			m.ack <- struct{}{}
		}
	}
}

func (s *StreamHandler) closeConsumer(ctx context.Context, c *Consumer) {
	log.Printf("Closing consumer %s", c.ID)

	if len(c.topics) > 0 {
		// Wait for http connection to terminate
		defer time.Sleep(time.Second)
	}

	// loop over all the topics and shut them down
	for topic := range c.topics {
		if err := s.unsubscribe(ctx, c.ID, topic); err != nil {
			log.Printf("error: unable to unsubscribe from topic, %v", err)
		}
		delete(c.topics, topic)
	}
	c.connected = false
	close(c.messageChannel)

	s.consumers[c.ID] = nil
	delete(s.consumers, c.ID)

	runtime.GC()
}

func (s *StreamHandler) subscribe(ctx context.Context, consumerID, topic string) error {
	consumer, err := s.GetConsumerByID(consumerID)
	if err != nil {
		return err
	}

	_, ok := consumer.topics[topic]
	if ok {
		return nil
	}

	if subscription, ok := s.subscriptions[topic]; ok {
		subscription.consumers[consumerID] = consumer
		consumer.topics[topic] = struct{}{}
		log.Printf("consumer %s subscribed for topic %s", consumerID, topic)
		return nil
	}

	// subscribe to the specified topic
	sub, err := s.subClient.Subscribe(ctx, topic)
	if err != nil {
		return err
	}

	subscription := &subscription{
		topic:        topic,
		consumers:    map[string]*Consumer{consumerID: consumer},
		shutdownChan: make(chan struct{}),
	}

	s.subscriptions[topic] = subscription
	consumer.topics[topic] = struct{}{}

	log.Printf("consumer %s subscribed for topic %s", consumerID, topic)

	go func() {
		log.Printf("no subscription found for topic: %v, creating a new one", topic)

		ctx := context.Background()

		for {
			select {
			case <-subscription.shutdownChan:
				log.Printf("zero consumers for topic: '%v', shutting down subscription", topic)
				err := sub.Shutdown(ctx)
				if err != nil {
					log.Println("error: stream handler: unable to shutdown subscription, ", err)
				}

				ack := make(chan struct{})
				s.shutdownSubscriptionChan <- topicMessage{
					topic: topic,
					ack:   ack,
				}
				<-ack

				return

			default:
				msg, err := sub.Receive(ctx)
				if err != nil {
					// if error is unknown then shutdown the subscription and
					// remove the subscription from the consumer
					log.Println("Error: Cannot receive message:", err)

					d, _ := json.Marshal(map[string]interface{}{
						"error": map[string]string{
							"topic":   topic,
							"code":    "subscription-failure",
							"message": "Cannot receive message from subscription, closing subscription",
						},
					})

					for _, c := range subscription.consumers {
						if c.connected {
							c.messageChannel <- message{event: "message", payload: string(d), messageType: ERROR}

							ack := make(chan struct{})

							s.removeConsumerTopicChan <- consumerTopicMessage{
								c:     c,
								topic: topic,
								ack:   ack,
							}

							<-ack
						}
					}
					msg.Ack()

					close(subscription.shutdownChan)
					continue
				}

				d, _ := json.Marshal(map[string]interface{}{
					"error": map[string]string{
						"topic":   topic,
						"code":    "subscription-failure",
						"message": "Cannot receive message from subscription, closing subscription",
					},
				})

				for _, c := range subscription.consumers {
					if c.connected {
						c.messageChannel <- message{event: "message", payload: string(d), messageType: ERROR}
					}
				}
				msg.Ack()
			}
		}
	}()
	return nil
}

func (s *StreamHandler) unsubscribe(ctx context.Context, consumerID, topic string) error {
	consumer, err := s.GetConsumerByID(consumerID)
	if err != nil {
		return err
	}

	_, ok := consumer.topics[topic]
	if !ok {
		return nil
	}

	subscription, ok := s.subscriptions[topic]
	if !ok {
		return nil
	}

	subscription.consumers[consumerID] = nil
	delete(subscription.consumers, consumerID)
	delete(consumer.topics, topic)

	if len(subscription.consumers) == 0 {
		close(subscription.shutdownChan)
	}

	log.Printf("consumer %s unsubscribed for topic %s", consumerID, topic)
	return nil
}

// ServeHTTP is the http handler for eventsource.
// For connecting query parameter 'id' is required i.e consumer_id.
func (s *StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	consumerID := r.URL.Query().Get("id")
	if len(consumerID) == 0 {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	consumer, err := s.GetConsumerByID(consumerID)
	if err != nil {
		http.Error(w, "invalid consumer id", http.StatusUnauthorized)
		return
	}

	// Stop the timeout timer if it is running.
	if !consumer.connected {
		consumer.connected = true
		consumer.ttlTimer.Stop()
		consumer.timerStopped <- true
	}

	log.Printf("consumer %s connected", consumerID)

	d, _ := json.Marshal(map[string]interface{}{
		"config": map[string]interface{}{
			"consumer_id":         consumerID,
			"consumer_ttl_millis": s.consumerTTL.Milliseconds(),
		},
		"subscriptions": s.GetConsumerTopics(ctx, consumer),
	})

	// First message:
	// consumer_id & consumer_ttl_millis
	d, _ = json.Marshal(
		[2]interface{}{"/unifrost/info", string(d)},
	)

	fmt.Fprint(w, "event: message\n")
	fmt.Fprintf(w, "data: %s\n\n", d)
	flusher.Flush()

	go func() {
		<-ctx.Done()
		log.Printf("consumer %s disconnected", consumerID)

		// When consumer gets disconnected start the timer
		timer := time.NewTimer(s.consumerTTL)

		consumer.ttlTimer = timer
		consumer.connected = false

		// cleanup the resources if the timer is expired
		select {
		case <-consumer.timerStopped:
			return
		case <-timer.C:
			log.Printf("Disconnected consumer %s : timed out, cleaning...", consumerID)
			s.CloseConsumer(context.Background(), consumerID)
		}
	}()

	for data := range consumer.messageChannel {
		if !s.IsConsumerConnected(consumer) {
			break
		}

		topic := data.event

		if data.messageType == ERROR {
			topic = "/unifrost/error"
		}

		d, _ := json.Marshal([2]string{topic, data.payload})

		fmt.Fprint(w, "event: message\n")
		fmt.Fprintf(w, "data: %s\n\n", d)
		flusher.Flush()
	}

	d, _ = json.Marshal([2]string{"/unifrost/closed", "consumer closed"})

	fmt.Fprint(w, "event: message\n")
	fmt.Fprintf(w, "data: %s\n\n", d)
	flusher.Flush()
}

// Subscribe subscribes the specified consumer to the specified topic.
// If specified consumer doesn't exists ErrConsumerNotFound error is returned.
func (s *StreamHandler) Subscribe(ctx context.Context, consumerID string, topic string) error {
	errChan := make(chan error)

	s.addSubscriptionChan <- subscriptionMessage{
		consumerID: consumerID,
		topic:      topic,
		errChan:    errChan,
	}

	return <-errChan
}

// Unsubscribe method unsubscribes the specified consumer to the specified topic
// and shutdowns the subscription.
// If specified consumer doesn't exists ErrConsumerNotFound error is returned.
func (s *StreamHandler) Unsubscribe(ctx context.Context, consumerID string, topic string) error {
	errChan := make(chan error)

	s.removeSubscriptionChan <- subscriptionMessage{
		consumerID: consumerID,
		topic:      topic,
		errChan:    errChan,
	}

	return <-errChan
}

// NewConsumer creates a new consumer with an autogenerated consumer id.
func (s *StreamHandler) NewConsumer(ctx context.Context) (*Consumer, error) {
	c := &Consumer{
		ID:             uuid.New().String(),
		messageChannel: make(chan message, 10),
		topics:         map[string]struct{}{},
		ttlTimer:       time.NewTimer(s.consumerTTL * time.Millisecond),
		connected:      true,
	}

	ack := make(chan struct{})

	s.addConsumerChan <- consumerMessageg{
		c:   c,
		ack: ack,
	}

	<-ack

	return c, nil
}

// NewCustomConsumer creates a new consumer with the specified consumer id.
func (s *StreamHandler) NewCustomConsumer(ctx context.Context, consumerID string) (*Consumer, error) {
	c := &Consumer{
		ID:             consumerID,
		messageChannel: make(chan message, 10),
		topics:         map[string]struct{}{},
		ttlTimer:       time.NewTimer(s.consumerTTL * time.Millisecond),
		connected:      true,
	}

	ack := make(chan struct{})

	s.addConsumerChan <- consumerMessageg{
		c:   c,
		ack: ack,
	}

	<-ack

	return c, nil
}

// GetConsumerByID returns a pointer consumer struct.
//
// If the consumer id specified is invalid or doesn't exists
// an error 'unifrost.ErrConsumerNotFound' is returned
func (s *StreamHandler) GetConsumerByID(consumerID string) (*Consumer, error) {
	c, ok := s.consumers[consumerID]
	if !ok {
		return nil, ErrConsumerNotFound
	}

	return c, nil
}

// TotalConsumers returns the number of consumer connected to the stream handler.
func (s *StreamHandler) TotalConsumers(ctx context.Context) int {
	return len(s.consumers)
}

// CloseConsumer closes the specified consumer and removes it.
func (s *StreamHandler) CloseConsumer(ctx context.Context, consumerID string) error {
	c, err := s.GetConsumerByID(consumerID)
	if err != nil {
		return err
	}

	ack := make(chan struct{})

	s.closeConsumerChan <- consumerMessageg{
		c:   c,
		ack: ack,
	}

	<-ack

	return nil
}

// GetConsumerTopics returns a slice of all the topics the consumer is subscribed to.
func (s *StreamHandler) GetConsumerTopics(ctx context.Context, c *Consumer) []string {
	topics := make([]string, 0, len(c.topics))
	for key := range c.topics {
		topics = append(topics, key)
	}

	return topics
}

// TotalConsumerTopics returns the number of topics the consumer is subscribed to.
func (s *StreamHandler) TotalConsumerTopics(ctx context.Context, c *Consumer) int {
	return len(c.topics)
}

// IsConsumerConnected reports whether consumer is connected to the server.
func (s *StreamHandler) IsConsumerConnected(c *Consumer) bool {
	return c.connected
}

// Close closes the StreamHandler and also closes all the connected consumers.
func (s *StreamHandler) Close(ctx context.Context) error {
	log.Println("Closing stream handler")

	// loop over all the consumers and close them.
	for _, c := range s.consumers {
		s.CloseConsumer(ctx, c.ID)
	}

	return s.subClient.Close(ctx)
}
