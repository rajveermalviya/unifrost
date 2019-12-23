package azrsbdriver

import (
	"context"

	servicebus "github.com/Azure/azure-service-bus-go"
)

// Client handles subscriptions to the Azure Service Bus
type Client struct {
	namespace *servicebus.Namespace
}

// Option is a self-refrential function for configuration
type Option func(*Client) error

// NewClient ...
func NewClient() (*Client, error) {
	return nil, nil
}

// Subscribe ...
func (client *Client) Subscribe(
	ctx context.Context,
	topic *servicebus.Topic,
	name string,
	opts ...Option) (*servicebus.Subscription, error) {

	return topic.NewSubscription(name)
}

// Close ...
func (client *Client) Close(ctx context.Context) error {
	return nil
}
