# Gochan: Stream pubsub messages to the web

gochan is a small library for relaying pubsub messages to the web via Server-Sent-Events.
It is loosely based on Twitter's implementation for real-time event-streaming
in their desktop website redesign.

It uses the [Go CDK](https://gocloud.dev) as vendor neutral pubsub driver that supports multiple pubusub vendors:

- Google Cloud Pub/Sub
- Amazon Simple Queueing Service
- Azure Service Bus
- RabbitMQ
- NATS
- Kafka
- In-Memory

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/rajveermalviya/gochan"
	"gocloud.dev/pubsub"
	_ "gocloud.dev/pubsub/<driver>"
)

func main() {
    b, _ := gochan.NewSubscriptionsHandler("<driver-prefix>", "<driver-opts>")
    // b, _ := gochan.NewSubscriptionsHandler("mem://", "") // Use mem only for testing.

	mux := http.NewServeMux()
	mux.HandleFunc("/update_subscriptions", b.UpdateSubscriptionsHandler)
	mux.HandleFunc("/events", b.EventStreamHandler)

	ctx := context.Background()

	go func() {

		topicA, err := pubsub.OpenTopic(ctx, "<driver-prefix>"+"topicA")
		// topicA, err := pubsub.OpenTopic(ctx, "mem://"+"topicA")
		if err != nil {
			log.Fatal(err)
		}
		topicB, err := pubsub.OpenTopic(ctx, "<driver-prefix>"+"topicB")
		// topicB, err := pubsub.OpenTopic(ctx, "mem://"+"topicB")
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

```

## Sponsor

If you **love** this library, you can support me by sposoring via [PayPal](https://paypal.me/rajveermalviya).
