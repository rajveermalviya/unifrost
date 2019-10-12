package drivers

import (
	"context"

	"github.com/nats-io/nats.go"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/natspubsub"
)

type ClientNats struct {
	conn *nats.Conn
}

type ConfigNats struct {
	NatsURL  string
	NatsOpts nats.Option
}

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

func (client *ClientNats) Subscribe(ctx context.Context, subject string) (*pubsub.Subscription, error) {
	return natspubsub.OpenSubscription(client.conn, subject, nil)
}

func (client *ClientNats) Close(ctx context.Context) error {
	client.conn.Close()
	return nil
}
