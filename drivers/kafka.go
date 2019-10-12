package drivers

import (
	"context"

	"github.com/Shopify/sarama"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/kafkapubsub"
)

type ClientKafka struct {
	config    *sarama.Config
	group     string
	addresses []string
}

type ConfigKafka struct {
	Config    *sarama.Config
	Group     string
	Addresses []string
}

func NewClientKafka(ctx context.Context, c *ConfigKafka) (*ClientKafka, error) {
	return &ClientKafka{config: c.Config, addresses: c.Addresses, group: c.Group}, nil
}

func (client *ClientKafka) Subscribe(ctx context.Context, topic string) (*pubsub.Subscription, error) {
	return kafkapubsub.OpenSubscription(client.addresses, client.config, client.group, []string{topic}, nil)
}

func (client *ClientKafka) Close(ctx context.Context) error {
	return nil
}
