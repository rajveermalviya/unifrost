# NATS example

To run the example, first run the nats server using docker

```shell
docker run --rm -it -p 4222:4222 nats
```

Then, in another terminal session start the topic publisher

```shell
go run publisher.go
```

You can also configure how many topics the publisher should create
and at what time interval every message is sent.

```shell
go run publisher.go --topics 100 --interval 1s
```

Then start the unifrost example server

```shell
go run main.go
```

Connect to the event streamer using the [EventSource](https://developer.mozilla.org/en/docs/Web/API/EventSource) API in the browser or by using good old curl

```
curl 'localhost:3000/events?id=custom_client'
```

And in another terminal update the subscriptions of the client

```
curl -d '{"client_id": "custom_client", "add": ["topic1", "topic2"], "remove": []}' -XPOST 'localhost:3000/update_subscriptions'
```
