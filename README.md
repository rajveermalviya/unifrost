# Gochan: A little package that can stream events to the web

gochan is a small package for relaying pubsub messages to the web via
[SSE(Eventsource)](https://en.wikipedia.org/wiki/Server-sent_events).
It is loosely based on Twitter's implementation for real-time event-streaming
in their new desktop web app.

It uses the [Go CDK](https://gocloud.dev) as vendor neutral pubsub driver that supports multiple pubusub vendors:

- Google Cloud Pub/Sub
- Amazon Simple Queueing Service
- Azure Service Bus
- RabbitMQ
- NATS
- Kafka
- In-Memory (Testing)

## Documentation

For documentation check [godoc](https://godoc.org/github.com/rajveermalviya/gochan).

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

## Future Goals:

- Support additional pusub vendors by contributing back to [Go CDK](https://gocloud.dev)
- Standalone server that can be configured by sweet yaml, while also staying modular.
- Support additional clients like Websockets and GRPC.
- Become a [CNCF](https://cncf.io) project (...maybe)

## Show some love

If you are able to contribute, that is the best way to show some love towards the project.

If you **love** gochan, you can support by sharing the project on Twitter or sponsor the project via [PayPal](https://paypal.me/rajveermalviya).
