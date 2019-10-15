package kafkadriver

import (
	"context"

	"github.com/Shopify/sarama"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/kafkapubsub"
)

// Client handles the subscriptions to Kafka topics
type Client struct {
	config  *sarama.Config
	groupID string
	brokers []string
}

// Options is a self-refrential function for configuration
type Options func(*Client) error

// NewClient constructor creates the client using the specified config
// sarama is used internally to connect to kafka brokers
// brokers is a slice of broker addresses
// config is a sarama.Config struct
// groupID is consumer group id
func NewClient(
	ctx context.Context,
	brokers []string,
	groupID string,
	config *sarama.Config,
	opts ...Options,
) (*Client, error) {

	c := &Client{config: config, brokers: brokers, groupID: groupID}

	for _, option := range opts {
		if err := option(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// Subscribe method subscribes to the kafka topic
func (client *Client) Subscribe(ctx context.Context, topic string) (*pubsub.Subscription, error) {
	return kafkapubsub.OpenSubscription(client.brokers, client.config, client.groupID, []string{topic}, nil)
}

// Close is just a placeholder
func (client *Client) Close(ctx context.Context) error {
	return nil
}
