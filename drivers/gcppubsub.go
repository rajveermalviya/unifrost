package drivers

import (
	"context"

	googlepubsub "cloud.google.com/go/pubsub/apiv1"
	"gocloud.dev/gcp"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/gcppubsub"
	"golang.org/x/oauth2/google"
)

type ClientGCPPubSub struct {
	subClient *googlepubsub.SubscriberClient
	cleanup   func()
	projectID gcp.ProjectID
}

type ConfigGCPPubSub struct {
	Creds *google.Credentials
}

func NewClientGCPPubSub(ctx context.Context, c *ConfigGCPPubSub) (*ClientGCPPubSub, error) {
	projectID, err := gcp.DefaultProjectID(c.Creds)
	if err != nil {
		return nil, err
	}

	conn, cleanup, err := gcppubsub.Dial(ctx, c.Creds.TokenSource)
	if err != nil {
		return nil, err
	}

	subClient, err := gcppubsub.SubscriberClient(ctx, conn)
	if err != nil {
		return nil, err
	}

	return &ClientGCPPubSub{subClient: subClient, cleanup: cleanup, projectID: projectID}, nil
}

func (client *ClientGCPPubSub) Subscribe(ctx context.Context, subscription string) (*pubsub.Subscription, error) {
	return gcppubsub.OpenSubscription(client.subClient, client.projectID, subscription, nil), nil
}

func (client *ClientGCPPubSub) Close(ctx context.Context) error {
	if err := client.subClient.Close(); err != nil {
		return err
	}
	client.cleanup()
	return nil
}
