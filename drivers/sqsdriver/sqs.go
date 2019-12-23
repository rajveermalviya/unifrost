package sqsdriver

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// Client handles the receiving from queues
type Client struct {
	subClient *sqs.SQS
}

// Option is a self-refrential function for configuration
type Option func(*Client) error

// NewClient ...
func NewClient(ctx context.Context, prov client.ConfigProvider, opts ...Option) (*Client, error) {
	svc := sqs.New(prov)

	c := &Client{subClient: svc}

	for _, option := range opts {
		if err := option(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// Subscribe ...
func (client *Client) Subscribe(ctx context.Context, topic string) (interface{}, error) {
	return nil, nil
}

// Close is just a placeholder
func (client *Client) Close(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}
