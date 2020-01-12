package natsdriver

import (
	"context"

	"github.com/nats-io/nats.go"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/natspubsub"
)

// Client handles the subscriptions to NATS subjects
type Client struct {
	conn *nats.Conn
}

// Option is a self-refrential function for configuration parameters
type Option func(*Client) error

// NewClient returns *natsdriver.Client, manages subscription to NATS subjects.
//
// Additional configuration options can be added with natsdriver.Option functions.
func NewClient(ctx context.Context, conn *nats.Conn, opts ...Option) (*Client, error) {

	c := &Client{conn: conn}

	for _, option := range opts {
		if err := option(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// Subscribe subscribes to the NATS subject
func (client *Client) Subscribe(ctx context.Context, subject string) (*pubsub.Subscription, error) {
	return natspubsub.OpenSubscription(client.conn, subject, nil)
}

// Close closes the connection to NATS server
func (client *Client) Close(ctx context.Context) error {
	client.conn.Close()
	return nil
}
