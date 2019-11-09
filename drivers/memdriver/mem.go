package memdriver

import (
	"context"

	"gocloud.dev/pubsub"

	// required for initializing mempubsub url opener
	_ "gocloud.dev/pubsub/mempubsub"
)

// Client handles the subscriptions to In-Memory topics
type Client struct{}

// Subscribe method subscribes to the In-Memory topic
func (client *Client) Subscribe(ctx context.Context, topic string) (*pubsub.Subscription, error) {
	return pubsub.OpenSubscription(ctx, "mem://"+topic)
}

// Close is just a placeholder
func (client *Client) Close(ctx context.Context) error {
	return nil
}
