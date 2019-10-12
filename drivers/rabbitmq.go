package drivers

import (
	"context"
	"fmt"

	"github.com/streadway/amqp"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/rabbitpubsub"
)

// ClientRabbitMQ handles the subscriptions to rabbit amqp topics
type ClientRabbitMQ struct {
	conn *amqp.Connection
}

// ConfigRabbitMQ contains the configuration need to create rabbit amqp client
type ConfigRabbitMQ struct {
	Username string
	Password string
	Address  string
}

// NewClientRabbitMQ constructor creates the client with the specified config
func NewClientRabbitMQ(ctx context.Context, c *ConfigRabbitMQ) (*ClientRabbitMQ, error) {
	url := fmt.Sprintf("amqp://%s:%s@%s/", c.Username, c.Password, c.Address)

	rabbitConn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	return &ClientRabbitMQ{conn: rabbitConn}, nil
}

// Subscribe method subscribes to the rabbit amqp topic
func (client *ClientRabbitMQ) Subscribe(ctx context.Context, topic string) (*pubsub.Subscription, error) {
	return rabbitpubsub.OpenSubscription(client.conn, topic, nil), nil
}

// Close closes the connection to rabbit amqp server
func (client *ClientRabbitMQ) Close(ctx context.Context) error {
	return client.conn.Close()
}
