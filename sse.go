// Package gochan is a small package for relaying pubsub messages to the web
// via SSE(EventSource).
// It is loosely based on Twitter's implementation for real-time event-streaming
// in their new desktop web app.
//
// It uses GO CDK (gocloud.dev) for pubsub, so it supports various vendors ->
//   Google Cloud Pub/Sub
//   Amazon Simple Queueing Service (SQS)
//   Azure Service Bus
//   RabbitMQ
//   NATS
//   Kafka
//   In-memory (testing).
//
// It provides two net/http handler functions ->
//  EventStreamHandler: It is a SSE compatible http handler, that streams all the messages.
//  UpdateSubscriptionsHandler: It update the client's subscriptions.
package gochan

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// EventStreamBroker manages all the clients and their respective subscriptions.
type EventStreamBroker struct {
	clients            map[string]*client
	subscriptionBroker *subscriptionsBroker
	mu                 sync.RWMutex
}

// EventStreamHandler is a SSE compatible handler function that is used to stream
// pubsub events to the client.
func (b *EventStreamBroker) EventStreamHandler(w http.ResponseWriter, r *http.Request) {

	// Make request SSE compatible.
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("Access-Control-Allow-Origin", "*")

	// Make sure that the writer supports flushing.
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	topics := strings.Split(r.URL.Query().Get("topics"), ",")
	if len(topics) == 0 {
		http.Error(w, "Atleast one topic is required.", http.StatusBadRequest)
		return
	}

	// Create a new client
	client := &client{
		sessID:       uuid.New().String(),
		writeChannel: make(chan []byte),
		topics:       make(map[string]bool),
	}

	b.mu.Lock()
	b.clients[client.sessID] = client
	b.mu.Unlock()

	b.mu.RLock()
	totalClients := len(b.clients)
	b.mu.RUnlock()

	log.Printf("New client with sessID '%v', total clients: %v \n", client.sessID, totalClients)

	// Write the session ID as the first message for using with 'UpdateSubscriptionsHandler'
	d, _ := json.Marshal(map[string]interface{}{"topic": "/httpsub/config", "payload": map[string]string{"session_id": client.sessID}})
	fmt.Fprintf(w, "data: %s\n\n", d)
	flusher.Flush()

	ctx := r.Context()

	for _, topic := range topics {
		if err := b.subscriptionBroker.SubscribeClient(client, topic); err != nil {
			log.Println("Error:", err)

			d, _ := json.Marshal(map[string]interface{}{
				"error": map[string]string{
					"code":    "subscription-failure",
					"message": fmt.Sprintf("Cannot subscribe to topic %v", topic),
				},
			})
			client.writeChannel <- d
		}
	}

	go func() {
		<-ctx.Done()

		client.mu.RLock()
		topics := client.topics
		client.mu.RUnlock()

		for topic := range topics {
			if err := b.subscriptionBroker.UnsubscribeClient(ctx, client, topic); err != nil {
				log.Println(err)
			}
		}

		b.mu.Lock()
		delete(b.clients, client.sessID)
		b.mu.Unlock()

		b.mu.RLock()
		totalClients := len(b.clients)
		b.mu.RUnlock()

		log.Printf("Client removed '%v', total clients: %v \n", client.sessID, totalClients)
	}()

	for {
		fmt.Fprintf(w, "data: %s\n\n", <-client.writeChannel)
		flusher.Flush()
	}

}

type updateSubscriptionsData struct {
	SessID string   `json:"session_id,omitempty"`
	Add    []string `json:"add,omitempty"`
	Remove []string `json:"remove,omitempty"`
}

// UpdateSubscriptionsHandler is used to update the client's subscriptions.
func (b *EventStreamBroker) UpdateSubscriptionsHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		http.Error(w, "Only POST method allowed", http.StatusMethodNotAllowed)
		return
	}

	h := w.Header()
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("Access-Control-Allow-Origin", "*")

	// Incoming request data
	var reqData updateSubscriptionsData

	// Decode JSON body
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&reqData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If the ID isn't provided, that means it is a new client
	// So generate an ID and create a new client.
	if reqData.SessID == "" {
		http.Error(w, "Session ID is required 'session_id'", http.StatusBadRequest)
		return
	}

	b.mu.RLock()
	client, ok := b.clients[reqData.SessID]
	b.mu.RUnlock()
	if !ok {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	for _, topic := range reqData.Add {
		if err := b.subscriptionBroker.SubscribeClient(client, topic); err != nil {
			log.Println("Error:", err)

			d, _ := json.Marshal(map[string]interface{}{
				"error": map[string]string{
					"code":    "subscription-failure",
					"message": fmt.Sprintf("Cannot subscribe to topic %v", topic),
				},
			})
			client.writeChannel <- d
		}
	}

	for _, topic := range reqData.Remove {
		if err := b.subscriptionBroker.UnsubscribeClient(ctx, client, topic); err != nil {
			log.Println("Error:", err)
		}
	}

	client.mu.RLock()
	totalTopics := len(client.topics)
	client.mu.RUnlock()

	log.Printf("Client '%v' subscriptions updated, total topics subscribed: %v \n", client.sessID, totalTopics)

	// Return the ID of the client.
	enc := json.NewEncoder(w)
	if err := enc.Encode(map[string]string{"session_id": reqData.SessID}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// NewSubscriptionsHandler is used to create a new EventStreamBroker
// that has the http handler functions.
// In Go CDK there are URL for connecting to pubsub.
//  Example:
//   'prefix' + 'topic' + 'postfixOpts'
//   'mem://' + '/topic/A' (here Opts are empty)
//   'mem://topicA'
//   'awssqs://sqs.us-east-2.amazonaws.com/123456789012/myqueue?region=us-east-2'
//   'azuresb://mytopic?subscription=mysubscription'
//   'rabbit://myqueue'
//   'nats://example.mysubject'
//   'kafka://my-group?topic=my-topic'
// prefix is the vendor prefix, please check GO CDK documentation.
// postfixOpts are options.
func NewSubscriptionsHandler(prefix string, postfixOpts string) (*EventStreamBroker, error) {

	if prefix == "" {
		return nil, errors.New("Prefix cannot be empty")
	}

	return &EventStreamBroker{
		subscriptionBroker: &subscriptionsBroker{
			Prefix:         prefix,
			PostfixOptions: postfixOpts,
			subs:           make(map[string]*subscription),
		},
		clients: make(map[string]*client),
	}, nil
}
