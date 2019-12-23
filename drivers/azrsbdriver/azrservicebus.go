package azrsbdriver

import (
	"context"
	"fmt"
	"net/url"

	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/azuresb"
)

// Client handles subscriptions to the Azure Service Bus
type Client struct {
	urlOpener *azuresb.URLOpener
}

// Option is a self-refrential function for configuration
type Option func(*Client) error

// NewClient constructs a new URL Opener client
func NewClient(urlOpener *azuresb.URLOpener, opts ...Option) (*Client, error) {
	c := &Client{urlOpener}
	for _, option := range opts {
		if err := option(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// Subscribe ...
func (client *Client) Subscribe(ctx context.Context, subURL string) (*pubsub.Subscription, error) {
	url, err := url.Parse(subURL)
	if err != nil {
		return nil, err
	}
	return client.urlOpener.OpenSubscriptionURL(ctx, url)
}

// Close is just a placeholder
// to close the subscription or topic, `subscription.Shutdown(ctx)`
// should be called
func (client *Client) Close(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}
