package drivers

import (
	"context"

	"github.com/Shopify/sarama"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/kafkapubsub"
)

// ClientKafka handles the subscriptions to Kafka topics
type ClientKafka struct {
	config    *sarama.Config
	group     string
	addresses []string
}

// ConfigKafka contains the configuration need to connect to Kafka cluster
type ConfigKafka struct {
	Config    *sarama.Config
	Group     string
	Addresses []string
}

// NewClientKafka constructor creates the client with the specified config
func NewClientKafka(ctx context.Context, c *ConfigKafka) (*ClientKafka, error) {
	return &ClientKafka{config: c.Config, addresses: c.Addresses, group: c.Group}, nil
}

// Subscribe method subscribes to the kafka topic
func (client *ClientKafka) Subscribe(ctx context.Context, topic string) (*pubsub.Subscription, error) {
	return kafkapubsub.OpenSubscription(client.addresses, client.config, client.group, []string{topic}, nil)
}

// Close is just a placeholder
func (client *ClientKafka) Close(ctx context.Context) error {
	return nil
}
