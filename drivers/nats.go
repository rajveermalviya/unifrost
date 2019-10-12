package drivers

import (
	"context"

	"github.com/nats-io/nats.go"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/natspubsub"
)

// ClientNats handles the subscriptions to NATS subjects
type ClientNats struct {
	conn *nats.Conn
}

// ConfigNats contains the configuration need to connect to NATS server
type ConfigNats struct {
	NatsURL  string
	NatsOpts nats.Option
}

// NewClientNats constructor creates the client with the specified config
func NewClientNats(ctx context.Context, c *ConfigNats) (*ClientNats, error) {

	url := c.NatsURL

	if c.NatsURL == "" {
		url = nats.DefaultURL
	}

	conn, err := nats.Connect(url, c.NatsOpts)
	if err != nil {
		return nil, err
	}

	return &ClientNats{conn: conn}, nil
}

// Subscribe method subscribes to the NATS subject
func (client *ClientNats) Subscribe(ctx context.Context, subject string) (*pubsub.Subscription, error) {
	return natspubsub.OpenSubscription(client.conn, subject, nil)
}

// Close closes the connection to NATS server
func (client *ClientNats) Close(ctx context.Context) error {
	client.conn.Close()
	return nil
}
