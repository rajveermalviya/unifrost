package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/rajveermalviya/gochan"
	"github.com/rajveermalviya/gochan/drivers"
	"gocloud.dev/pubsub"
	_ "gocloud.dev/pubsub/mempubsub"
)

var (
	numTopics int           = 1000
	interval  time.Duration = time.Second
)

func main() {

	log.SetFlags(log.Lshortfile)

	ctx := context.Background()

	if len(os.Args) > 1 {
		numTopics, _ = strconv.Atoi(os.Args[1])
	}

	if len(os.Args) > 2 {
		i, _ := strconv.Atoi(os.Args[2])
		interval = time.Duration(i)
	}

	streamer, err := gochan.NewStreamer(ctx, &gochan.ConfigStreamer{
		Driver:       drivers.DriverMem,
		DriverConfig: &drivers.ConfigMem{},
	})
	defer streamer.Close(ctx)

	c, _ := streamer.NewCustomClient(ctx, "custom_client")
	log.Println("New client created:", c.ID)

	if err != nil {
		log.Fatalln("Error while starting streamer: ", err)
	}

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

	go func() {
		var topics = make([]*pubsub.Topic, numTopics)

		for i := 0; i < numTopics; i++ {
			topic, err := pubsub.OpenTopic(ctx, "mem://topic"+strconv.Itoa(i))
			if err != nil {
				log.Fatal(err)
			}
			topics[i] = topic
		}

		log.Printf("Opened %v topics\n", numTopics)
		log.Printf("Sending message every %vs\n", interval.Seconds())

		for {
			log.Println("Sending message")

			var wg sync.WaitGroup

			for j := 0; j < numTopics; j++ {
				wg.Add(1)
				go func(i int) {
					eventString := fmt.Sprintf("Topic%v %v", i, time.Now())

					err := topics[i].Send(ctx, &pubsub.Message{Body: []byte(eventString)})
					if err != nil {
						log.Fatal(err)
					}
					wg.Done()
				}(j)
			}
			time.Sleep(interval)
			wg.Wait()
		}
	}()

	log.Println("Starting server on port 3000")

	log.Fatal("HTTP server error: ", http.ListenAndServe("localhost:3000", mux))
}

func updateSubscriptions(streamer *gochan.Streamer) http.HandlerFunc {
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
				if err == gochan.ErrNoClient {
					http.Error(w, "Invalid Client ID", http.StatusBadRequest)
					return
				}
			}
		}

		for _, topic := range reqData.Remove {
			if err := streamer.Unsubscribe(ctx, reqData.ClientID, topic); err != nil {
				if err == gochan.ErrNoClient {
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
