package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/rajveermalviya/unifrost"
	"github.com/rajveermalviya/unifrost/drivers/memdriver"
	"gocloud.dev/pubsub"

	// required for initializing mempubsub
	_ "gocloud.dev/pubsub/mempubsub"
)

var (
	numTopics *int
	interval  *time.Duration
)

func main() {
	numTopics = flag.Int("topics", 100000, "Number of topics")
	interval = flag.Duration("interval", time.Second, "Time interval")
	flag.Parse()

	log.SetFlags(log.Lshortfile)

	ctx := context.Background()

	streamer, err := unifrost.NewStreamer(
		&memdriver.Client{},
		unifrost.ClientTTL(2*time.Second),
	)
	if err != nil {
		log.Fatalln("Error while starting streamer: ", err)
	}
	defer streamer.Close(ctx)

	// add a signal notifier to close the streamer gracefully
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, os.Kill)

	go func() {
		log.Println("sig:", <-sigs)
		log.Println("Gracefully closing the server")

		if err := streamer.Close(ctx); err != nil {
			log.Printf("Error occurred while closing the streamer: %v : closing forcefully", err)
		}

		os.Exit(0)
	}()

	// create a new custom_client for testing
	c, _ := streamer.NewCustomClient(ctx, "custom_client")
	log.Println("New client created:", c.ID)

	mux := http.NewServeMux()
	mux.HandleFunc("/update_subscriptions", updateSubscriptions(streamer))
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		// Auto generate new client id, when new client connects.
		q := r.URL.Query()
		if q.Get("id") == "" {
			c, _ := streamer.NewClient(ctx)
			q.Set("id", c.ID)
			r.URL.RawQuery = q.Encode()
		}

		streamer.ServeHTTP(w, r)
	})

	// Open 1000 topics via the in-memory driver.
	go func() {
		var topics = make([]*pubsub.Topic, *numTopics)

		for i := 0; i < *numTopics; i++ {
			topic, err := pubsub.OpenTopic(ctx, "mem://topic"+strconv.Itoa(i))
			if err != nil {
				log.Fatal(err)
			}
			topics[i] = topic
		}

		log.Printf("Opened %v topics\n", *numTopics)
		log.Printf("Sending message every %vs\n", interval.Seconds())

		for {
			log.Println("Sending message")

			var wg sync.WaitGroup

			for j := 0; j < *numTopics; j++ {
				wg.Add(1)
				go func(i int) {
					eventString := fmt.Sprintf("Topic%v %v", i, time.Now())

					d, _ := json.Marshal(map[string]string{"payload": eventString})

					err := topics[i].Send(ctx, &pubsub.Message{Body: d})
					if err != nil {
						log.Fatal(err)
					}
					wg.Done()
				}(j)
			}
			time.Sleep(*interval)
			wg.Wait()
		}
	}()

	log.Println("Starting server on port 3000")

	log.Fatal("HTTP server error: ", http.ListenAndServe("localhost:3000", mux))
}

func updateSubscriptions(streamer *unifrost.Streamer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != "POST" {
			http.Error(w, "Only POST method allowed", http.StatusMethodNotAllowed)
			return
		}

		h := w.Header()
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")
		h.Set("Access-Control-Allow-Origin", "*")

		type updateSubscriptionsData struct {
			ClientID string   `json:"client_id,omitempty"`
			Add      []string `json:"add,omitempty"`
			Remove   []string `json:"remove,omitempty"`
		}

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
		if reqData.ClientID == "" {
			http.Error(w, "Client ID is required 'client_id'", http.StatusBadRequest)
			return
		}

		ctx := r.Context()

		for _, topic := range reqData.Add {
			if err := streamer.Subscribe(ctx, reqData.ClientID, topic); err != nil {
				if err == unifrost.ErrNoClient {
					http.Error(w, "Invalid Client ID", http.StatusBadRequest)
					return
				}
			}
		}

		for _, topic := range reqData.Remove {
			if err := streamer.Unsubscribe(ctx, reqData.ClientID, topic); err != nil {
				if err == unifrost.ErrNoClient {
					http.Error(w, "Invalid Client ID", http.StatusBadRequest)
					return
				}
			}
		}

		client := streamer.GetClient(ctx, reqData.ClientID)

		log.Printf("Client '%v' subscriptions updated, total topics subscribed: %v \n", client.ID, client.TotalTopics(ctx))

		// Return the ID of the client.
		json.NewEncoder(w).Encode(map[string]string{"client_id": reqData.ClientID})
	}
}
