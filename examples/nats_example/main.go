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

package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/unifrost/unifrost"
	"github.com/unifrost/unifrost/drivers/natsdriver"
)

func main() {
	log.SetFlags(log.Lshortfile)

	ctx := context.Background()

	natsConn, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalln("Error while connecting to nats server: ", err)
	}

	subClient, err := natsdriver.NewClient(ctx, natsConn)
	if err != nil {
		log.Fatalln("Error while creating new client: ", err)
	}

	s, err := unifrost.NewStreamHandler(
		ctx,
		subClient,
		unifrost.ConsumerTTL(2*time.Second),
	)
	if err != nil {
		log.Fatalln("Error while starting streamer: ", err)
	}

	// add a signal notifier to close the streamer gracefully
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)

	go func() {
		log.Println("sig:", <-sigs)
		log.Println("Gracefully closing the server")

		if err := s.Close(ctx); err != nil {
			log.Printf("Error occurred while closing the streamer: %v : closing forcefully", err)
		}

		os.Exit(0)
	}()

	// create a new custom consumer for testing
	c, _ := s.NewCustomConsumer(ctx, "unique_id")
	log.Println("New consumer created:", c.ID)

	mux := http.NewServeMux()
	mux.HandleFunc("/update_subscriptions", updateSubscriptions(s))
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		// Auto generate new consumer id, when new consumer connects.
		q := r.URL.Query()
		if q.Get("id") == "" {
			c, _ := s.NewConsumer(ctx)
			q.Set("id", c.ID)
			r.URL.RawQuery = q.Encode()
		}

		s.ServeHTTP(w, r)
	})

	log.Println("Starting server on port 3000")

	log.Fatal("HTTP server error: ", http.ListenAndServe("localhost:3000", mux))
}

// updateSubscriptions is a helper handler to use the server without any
// authentication for subscribing and unsubscribing to topics for a consumer
func updateSubscriptions(streamer *unifrost.StreamHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != "POST" {
			http.Error(w, "Only POST method allowed", http.StatusMethodNotAllowed)
			return
		}

		h := w.Header()
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")

		type updateSubscriptionsData struct {
			ConsumerID string   `json:"consumer_id,omitempty"`
			Add        []string `json:"add,omitempty"`
			Remove     []string `json:"remove,omitempty"`
		}

		// Incoming request data
		var reqData updateSubscriptionsData

		// Decode JSON body
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&reqData); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// If the ID isn't provided, that means it is a new consumer
		// So generate an ID and create a new consumer.
		if reqData.ConsumerID == "" {
			http.Error(w, "consumer id is required 'consumer_id'", http.StatusBadRequest)
			return
		}

		consumer, err := streamer.GetConsumerByID(reqData.ConsumerID)
		if err == unifrost.ErrConsumerNotFound {
			http.Error(w, "invalid consumer ID", http.StatusUnauthorized)
			return
		}

		ctx := r.Context()

		for _, topic := range reqData.Add {
			if err := streamer.Subscribe(ctx, reqData.ConsumerID, topic); err != nil {
				if err != nil {
					log.Printf("unable to subscribe, %v", err)
					continue
				}
			}
		}

		for _, topic := range reqData.Remove {
			if err := streamer.Unsubscribe(ctx, reqData.ConsumerID, topic); err != nil {
				if err != nil {
					log.Printf("unable to subscribe, %v", err)
					continue
				}
			}
		}

		log.Printf("consumer '%v' subscriptions updated, total topics subscribed: %v \n", consumer.ID, streamer.GetConsumerTopics(ctx, consumer))

		// Return the ID of the consumer.
		json.NewEncoder(w).Encode(map[string]string{"consumer_id": reqData.ConsumerID})
	}
}
