# unifrost: A go module that makes it easier to stream pubsub events to the web

[![GoDoc](https://godoc.org/github.com/unifrost/unifrost?status.svg)](https://godoc.org/github.com/unifrost/unifrost)
[![Go Report Card](https://goreportcard.com/badge/github.com/unifrost/unifrost)](https://goreportcard.com/report/unifrost/unifrost)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/3298/badge)](https://bestpractices.coreinfrastructure.org/projects/3298)

⚠ This project is on early stage, it's not ready for production yet ⚠

Previously named gochan

unifrost is a go module for relaying pubsub messages to the web via [SSE(Eventsource)](https://en.wikipedia.org/wiki/Server-sent_events).
It is based on Twitter's implementation for real-time event-streaming in their
new web app.

unifrost is named after bifrost, the rainbow bridge that connects Asgard with
Midgard (Earth), that is MCU reference which is able to transport people both
ways. But because unifrost sends messages from server to client (only one way),
hence unifrost. 😎

It uses the [Go CDK](https://gocloud.dev) as broker neutral pubsub driver that
supports multiple pubsub brokers:

- Google Cloud Pub/Sub
- Amazon Simple Queueing Service
- Azure Service Bus (Pending)
- RabbitMQ
- NATS
- Kafka
- In-Memory (Only for testing)

## Installation

unifrost supports Go modules and built against go version 1.13

```sh
go get github.com/unifrost/unifrost
```

## Documentation

For documentation check [godoc](https://godoc.org/github.com/unifrost/unifrost).

## Usage

unifrost uses Server-Sent-Events, because of this it doesn't require to run a
standalone server, unlike websockets it can be embedded in your api server.
unifrost's stream handler has a ServeHTTP method i.e it implements http.Handler interface
so that it can be used directly or can be wrapped with middlewares like
Authentication easily.

```go
// Golang (psuedo-code)

// Using stream handler directly
streamHandler, err := unifrost.NewStreamHandler(
  ctx,
  &memdriver.Client{},
  unifrost.ConsumerTTL(2*time.Second),
)
log.Fatal("HTTP server error: ", http.ListenAndServe("localhost:3000", streamHandler))
```

```go
// Golang (psuedo-code)

// Using stream handler by wrapping it in auth middleware
streamHandler, err := unifrost.NewStreamHandler(
  ctx,
  &memdriver.Client{},
  unifrost.ConsumerTTL(2*time.Second),
)

mux := http.NewServeMux()
mux.HandleFunc("/events", func (w http.ResponseWriter, r *http.Request) {
    err := Auth(r)
    if err != nil {
      http.Error(w, "unauthorized", http.StatusUnauthorized)
      return
    }

    streamHandler.ServeHTTP(w,r)
})
log.Fatal("HTTP server error: ", http.ListenAndServe("localhost:3000", mux))
```

# Message Protocol

Every message sent by the server is encoded in plaintext in JSON,
it contains topic and the payload.

Every message will be an array of length 2, first index will be the topic string,
second index will be the payload in string type.

When consumer connects to the server, server sends a preflight message that contains
the initial server configuration and list of topics the consumer has already been subscribed.

1. Configuration: it contains the consumer_id and consumer_ttl set by the
   stream handler config
2. Subscriptions associated with the specified consumer id.

Example first message:

```json
[
  "/unifrost/info",

  "{\"config\":{\"consumer_id\":\"unique-id\",\"consumer_ttl_millis\":2000},\"subscriptions\":[]}"
]
```

Example error message:

```json
[
  "/unifrost/error",

  "{\"error\":{\"code\":\"subscription-failure\",\"message\":\"Cannot receive message from subscription, closing subscription\",\"topic\":\"topic3\"}}"
]
```

All the messages are streamed over single channel, i.e using EventSource JS API
`new EventSource().onmessage` or `new EventSource().addEventListener('message', (e) =>{})`
methods will listen to them.

All the info events are streamed over message channel i.e using the EventSource JS API,
`onmessage` or `addEventListener('message', () => {})` method will listen to them.
All the subscription events have event name same as their topic name, so to listen to
topic events you need to add an event-listener on the EventSource object.

# Example

Client example:

```ts
// Typescript (psuedo-code)
const consumerID = 'unique-id';

const sse = new EventSource(`/events?id=${consumerID}`);
// for info events like first-message and errors
sse.addEventListener('message', (e) => {
  const message = JSON.parse(e.data);

  const topic = message[0] as String;
  const payload = message[1] as String;

  // Payload is the exact message from Pub Sub broker, probably JSON.
  // Decode payload
  const data = JSON.parse(payload);
});
```

New consumer is registered explicitly using the `streamHandler.NewConsumer()`
with an auto generated id.
To register a consumer with custom id use `streamHandler.NewCustomConsumer(id)`

This makes it easy to integrate authentication with `unifrost.StreamHandler`.
One possible auth workflow can be, create a new unifrost consumer after login
and return the consumer id to the client to store it in the local storage of the
browser. Further using the consumer id to connect to the stream handler.

If you don't care about authentication, you can also generate a new consumer
automatically everytime a new consumer connects without the `id` parameter
use the following middleware with the streamer.
And handle registering the `id` to your backend from your client.

```go
// Golang (psuedo-code)

mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
    // Auto generate new consumer_id, when new consumer connects.
    q := r.URL.Query()
    if q.Get("id") == "" {
        consumer, _ := streamHandler.NewConsumer(ctx)
        q.Set("id", consumer.ID)
        r.URL.RawQuery = q.Encode()
    }

    streamer.ServeHTTP(w, r)
})
```

When a consumer gets disconnected it has a time window to connect to the server
again with the state unchanged. If consumer ttl is not specified in the
streamer config then default ttl is set to one.

# Managing subscriptions

`unifrost.StreamHandler` provides simple API for subscribing and unsubscribing to topics.

```go
func (s *StreamHandler) Subscribe(ctx context.Context, consumerID string, topic string) error


func (s *StreamHandler) Unsubscribe(ctx context.Context, consumerID string, topic string) error
```

These methods can be used to add or remove subscriptions for a consumer.

If you want to give subscription control to the client look at
[the implementation](examples/nats_example) in the example.

To know more, check out the [example](examples/nats_example)

## Why Server Sent Events (SSE) ?

Why would you choose SSE over WebSockets?

One reason SSEs have been kept in the shadow is because later APIs like
WebSockets provide a richer protocol to perform bi-directional, full-duplex
communication. However, in some scenarios data doesn't need to be sent from the
client. You simply need updates from some server action. A few examples would
be status updates, tweet likes, tweet retweets, tickers, news feeds, or other
automated data push mechanisms (e.g. updating a client-side Web SQL Database or
IndexedDB object store). If you'll need to send data to a server, Fetch API is
always a friend.

SSEs are sent over traditional HTTP. That means they do not require a special
protocol or server implementation to get working. WebSockets on the other hand,
require full-duplex connections and new Web Socket servers to handle the
protocol. In addition, Server-Sent Events have a variety of features that
WebSockets lack by design such as automatic reconnection, event IDs, and the
ability to send arbitrary events.

Because SSE works on top of HTTP, HTTP protocol improvements can also benefit SSE.
For example, the in-development HTTP/3 protocol, built on top of QUIC, could
offer additional performance improvements in the presence of packet loss due to
lack of head-of-line blocking.

## Community:

Join the #unifrost channel on [gophers](https://gophers.slack.com/messages/unifrost)
Slack Workspace for questions and discussions.

## Future Goals:

- Standalone server that can be configured by yaml, while also staying modular.
- Creating a website for documentation & overview, and some examples.

## Users

If you are using unifrost in production please let me know by sending an
[email](mailto:rajveer0malviya@gmail.com) or file an issue.

## Show some love

The best way to show some love towards the project, is to contribute and file
issues.

If you **love** unifrost, you can support by sharing the project on Twitter.

You can also support by sponsoring the project via [PayPal](https://paypal.me/rajveermalviya).

## License

[APACHE v2](LICENSE)
