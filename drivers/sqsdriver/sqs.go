package sqsdriver

import (
	"context"
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go/aws/client"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/awssnssqs"
)

// Client handles the communicating with SQS. This holds the config provider needed to make the request.
type Client struct {
	urlOpener *awssnssqs.URLOpener
}

// Option is a self-refrential function for configuration
type Option func(*Client) error

// NewClient ...
func NewClient(ctx context.Context, prov client.ConfigProvider, opts ...Option) (*Client, error) {
	c := &Client{&awssnssqs.URLOpener{ConfigProvider: prov}}
	for _, option := range opts {
		if err := option(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// Subscribe method subscribes to the given SQS url
func (client *Client) Subscribe(ctx context.Context, sqsURL string) (*pubsub.Subscription, error) {
	url, err := url.Parse(sqsURL)
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
