package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/rajveermalviya/gochan"
	"gocloud.dev/pubsub"
	_ "gocloud.dev/pubsub/mempubsub"
)

func main() {
	b, _ := gochan.NewSubscriptionsHandler("mem://", "")

	mux := http.NewServeMux()
	mux.HandleFunc("/update_subscriptions", b.UpdateSubscriptionsHandler)
	mux.HandleFunc("/events", b.EventStreamHandler)

	ctx := context.Background()

	go func() {
		var topics [1000]*pubsub.Topic

		for i := 0; i < 1000; i++ {
			topic, err := pubsub.OpenTopic(ctx, "mem://topic"+strconv.Itoa(i))
			if err != nil {
				log.Fatal(err)
			}
			topics[i] = topic
		}

		log.Println("Opened 1000 topics")

		for {
			log.Println("Sending message")

			var wg sync.WaitGroup

			for j := 0; j < 1000; j++ {
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
			time.Sleep(time.Second * 3)
			wg.Wait()
		}
	}()

	log.Println("Starting server on port 3000")

	log.Fatal("HTTP server error: ", http.ListenAndServe("localhost:3000", mux))
}
