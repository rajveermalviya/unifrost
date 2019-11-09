package gcpdriver

import (
	"context"

	googlepubsub "cloud.google.com/go/pubsub/apiv1"
	"gocloud.dev/gcp"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/gcppubsub"
	"google.golang.org/grpc"
)

// Client handles the subscriptions to the GCP PubSub Subscriptions
type Client struct {
	subClient *googlepubsub.SubscriberClient
	projectID gcp.ProjectID
}

// Option is a self-refrential function type for configuration
type Option func(*Client) error

// NewClient constructor connects to the GCP PubSub API and creates a client
func NewClient(ctx context.Context, conn *grpc.ClientConn, projectID gcp.ProjectID, opts ...Option) (*Client, error) {
	subClient, err := gcppubsub.SubscriberClient(ctx, conn)
	if err != nil {
		return nil, err
	}

	c := &Client{subClient: subClient, projectID: projectID}

	for _, option := range opts {
		if err := option(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// Subscribe method subscribes to the GCP PubSub subscription
func (client *Client) Subscribe(ctx context.Context, subscription string) (*pubsub.Subscription, error) {
	return gcppubsub.OpenSubscription(client.subClient, client.projectID, subscription, nil), nil
}

// Close closes the GRPC connection to GCP PubSub API server
func (client *Client) Close(ctx context.Context) error {
	return client.subClient.Close()
}
