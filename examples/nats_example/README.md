# NATS example

To run the example, first run the nats server using docker

```sh
docker run --rm -it -p 4222:4222 nats
```

Then, in another terminal session start the topic publisher

```sh
cd publisher
go run publisher.go
```

You can also configure how many topics the publisher should create
and at what time interval every message is sent.

```sh
go run publisher.go --topics 100 --interval 1s
```

Then start the unifrost example server

```sh
go run main.go
```

Connect to the event streamer using the [EventSource](https://developer.mozilla.org/en/docs/Web/API/EventSource) API
in the browser or by using good old curl

```sh
curl 'localhost:3000/events?id=custom_client'
```

And in another terminal update the subscriptions of the client

```sh
curl -d '{"client_id": "custom_client", "add": ["topic1", "topic2"], "remove": []}' -XPOST 'localhost:3000/update_subscriptions'
```
