package drivers

import (
	"context"

	"gocloud.dev/pubsub"
)

// ClientMem handles the subscriptions to In-Memory topics
type ClientMem struct{}

// ConfigMem is just placeholder
type ConfigMem struct{}

// NewClientMem constructor creates the client with the specified config
func NewClientMem(ctx context.Context, c *ConfigMem) (*ClientMem, error) {
	return &ClientMem{}, nil
}

// Subscribe method subscribes to the In-Memory topic
func (client *ClientMem) Subscribe(ctx context.Context, topic string) (*pubsub.Subscription, error) {
	return pubsub.OpenSubscription(ctx, "mem://"+topic)
}

// Close is just a placeholder
func (client *ClientMem) Close(ctx context.Context) error {
	return nil
}
