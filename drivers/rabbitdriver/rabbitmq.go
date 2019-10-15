package rabbitdriver

import (
	"context"

	"github.com/streadway/amqp"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/rabbitpubsub"
)

// Client handles the subscriptions to rabbit amqp topics
type Client struct {
	conn *amqp.Connection
}

// Options is a self-refrential function for configuration
type Options func(*Client) error

// NewClient constructor creates connects to the RabbitMQ API using the specified url
// url should be in AMQP URI format
func NewClient(ctx context.Context, url string, opts ...Options) (*Client, error) {

	rabbitConn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	return &Client{conn: rabbitConn}, nil
}

// Subscribe method subscribes to the rabbit amqp topic
func (client *Client) Subscribe(ctx context.Context, topic string) (*pubsub.Subscription, error) {
	return rabbitpubsub.OpenSubscription(client.conn, topic, nil), nil
}

// Close closes the connection to rabbit amqp server
func (client *Client) Close(ctx context.Context) error {
	return client.conn.Close()
}
