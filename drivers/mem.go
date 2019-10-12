package drivers

import (
	"context"

	"gocloud.dev/pubsub"
	_ "gocloud.dev/pubsub/mempubsub"
)

type ClientMem struct{}

type ConfigMem struct{}

func NewClientMem(ctx context.Context, c *ConfigMem) (*ClientMem, error) {
	return &ClientMem{}, nil
}

func (client *ClientMem) Subscribe(ctx context.Context, topic string) (*pubsub.Subscription, error) {
	return pubsub.OpenSubscription(ctx, "mem://"+topic)
}

func (client *ClientMem) Close(ctx context.Context) error {
	return nil
}
