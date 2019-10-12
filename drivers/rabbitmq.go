package drivers

import (
	"context"
	"fmt"

	"github.com/streadway/amqp"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/rabbitpubsub"
)

type ClientRabbitMQ struct {
	conn *amqp.Connection
}

type ConfigRabbitMQ struct {
	Username string
	Password string
	Address  string
}

func NewClientRabbitMQ(ctx context.Context, c *ConfigRabbitMQ) (*ClientRabbitMQ, error) {
	url := fmt.Sprintf("amqp://%s:%s@%s/", c.Username, c.Password, c.Address)

	rabbitConn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	return &ClientRabbitMQ{conn: rabbitConn}, nil
}

func (client *ClientRabbitMQ) Subscribe(ctx context.Context, topic string) (*pubsub.Subscription, error) {
	return rabbitpubsub.OpenSubscription(client.conn, topic, nil), nil
}

func (client *ClientRabbitMQ) Close(ctx context.Context) error {
	return client.conn.Close()
}
