package azrsbdriver

import (
	"context"
	"fmt"

	servicebus "github.com/Azure/azure-service-bus-go"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/azuresb"
)

// Client handles subscriptions to the Azure Service Bus
type Client struct {
	namespace *servicebus.Namespace
}

// Option is a self-refrential function for configuration
type Option func(*Client) error

// NewClient constructs a new namespace for Azure
func NewClient(conn string, opts ...Option) (*Client, error) {
	ns, err := azuresb.NewNamespaceFromConnectionString(conn)
	if err != nil {
		return nil, err
	}
	c := &Client{namespace: ns}
	for _, option := range opts {
		if err := option(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

/*
subscription, err := azuresb.OpenSubscription(ctx,
    busNamespace, busTopic, busSub, nil)
if err != nil {
    log.Fatal(err)
}
defer subscription.Shutdown(ctx)
*/

// Subscribe ...
func (client *Client) Subscribe(ctx context.Context, url string) (*pubsub.Subscription, error) {
	return nil, nil
}

// Close is just a placeholder
// to close the subscription or topic, `subscription.Shutdown(ctx)`
// should be called
func (client *Client) Close(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}
