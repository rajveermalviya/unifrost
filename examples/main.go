package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
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

		topicA, err := pubsub.OpenTopic(ctx, "mem://topicA")
		if err != nil {
			log.Fatal(err)
		}
		topicB, err := pubsub.OpenTopic(ctx, "mem://topicB")
		if err != nil {
			log.Fatal(err)
		}

		for {
			time.Sleep(time.Second * 3)

			log.Println("Sending message")

			eventStringA := fmt.Sprintf("TopicA %v", time.Now())

			err = topicA.Send(ctx, &pubsub.Message{Body: []byte(eventStringA)})
			if err != nil {
				log.Fatal(err)
			}

			eventStringB := fmt.Sprintf("TopicB %v", time.Now())

			err = topicB.Send(ctx, &pubsub.Message{Body: []byte(eventStringB)})
			if err != nil {
				log.Fatal(err)
			}
		}
	}()

	log.Fatal("HTTP server error: ", http.ListenAndServe("localhost:3000", mux))
}
