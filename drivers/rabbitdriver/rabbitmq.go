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

// Option is a self-refrential function for configuration parameters
type Option func(*Client) error

// NewClient returns *rabbitmq.Client, also creates a RabbitMQ broker subscription client.
//
// Additional configuration options can be added with rabbitpubsub.Option functions.
func NewClient(ctx context.Context, rabbitConn *amqp.Connection, opts ...Option) (*Client, error) {
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
