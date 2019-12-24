package sqsdriver

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/awssnssqs"
)

// Client handles the communicating with SQS. This holds the config provider needed to make the request.
type Client struct {
	session *session.Session
}

// Option is a self-refrential function for configuration
type Option func(*Client) error

// NewClient establishes session to the AWS cloud service.
func NewClient(ctx context.Context, session *session.Session, opts ...Option) (*Client, error) {
	c := &Client{session: session}
	for _, option := range opts {
		if err := option(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// Subscribe method subscribes to the given SQS url
// https://docs.aws.amazon.com/sdk-for-net/v2/developer-guide/QueueURL.html
func (client *Client) Subscribe(ctx context.Context, url string) (*pubsub.Subscription, error) {
	return awssnssqs.OpenSubscription(ctx, client.session, url, nil), nil
}

// Close is just a placeholder
// to close the subscription or topic, `subscription.Shutdown(ctx)`
// should be called
func (client *Client) Close(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}
