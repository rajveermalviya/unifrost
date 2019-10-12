# Gochan: A little package that can stream events to the web

[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Frajveermalviya%2Fgochan.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Frajveermalviya%2Fgochan?ref=badge_shield)

⚠ This project is on early stage, it's not ready for production yet ⚠

gochan is a small package for relaying pubsub messages to the web via
[SSE(Eventsource)](https://en.wikipedia.org/wiki/Server-sent_events).
It is based on Twitter's implementation for real-time event-streaming
in their new web app.

It uses the [Go CDK](https://gocloud.dev) as vendor neutral pubsub driver that supports multiple pubusub vendors:

- Google Cloud Pub/Sub
- Amazon Simple Queueing Service (Pending)
- Azure Service Bus (Pending)
- RabbitMQ
- NATS
- Kafka
- In-Memory (Only for testing)

## Installation

Go chan supports Go modules and built against go version 1.13

```shell
go get github.com/rajveermalviya/gochan
```

## Documentation

For documentation check [godoc](https://godoc.org/github.com/rajveermalviya/gochan).

## Usage

Check out the [example](examples/1000topics)

## Future Goals:

- Support additional pusub vendors by contributing back to [Go CDK](https://gocloud.dev)
- Standalone server that can be configured by yaml, while also staying modular.
- Become a [CNCF](https://cncf.io) project (...maybe)

## Users

If you are using gochan in production please let me know by sending an [email](mailto:rajveer0malviya@gmail.com) or file an issue

## Show some love

The best way to show some love towards the project, is to contribute and file issues.

If you **love** gochan, you can support by sharing the project on Twitter.

You can also support by sponsoring the project via [PayPal](https://paypal.me/rajveermalviya).

## License

[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Frajveermalviya%2Fgochan.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Frajveermalviya%2Fgochan?ref=badge_large)
