# 1000topics example

This example will create 1000 topics using the In-memory driver.

To run the example

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
