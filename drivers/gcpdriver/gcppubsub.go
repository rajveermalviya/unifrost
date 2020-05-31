// Copyright 2019-2020 Rajesh Malviya
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package gcpdriver contains Google Cloud Pub/Sub driver for unifrost.StreamHandler
package gcpdriver

import (
	"context"

	googlepubsub "cloud.google.com/go/pubsub/apiv1"
	"gocloud.dev/gcp"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/gcppubsub"
	"google.golang.org/grpc"
)

// Client handles the subscriptions to the GCP PubSub broker topics
type Client struct {
	subClient *googlepubsub.SubscriberClient
	projectID gcp.ProjectID
}

// Option is a self-refrential function for configuration parameters
type Option func(*Client) error

// NewClient returns *gcpdriver.Client, also creates a GCP pub-sub broker client.
//
// conn is the client connection returned by gcppubsub.Dial for connecting to GCP pub-sub broker,
// projectID is the project ID for your GCP project.
// Additional configuration options can be added with gcppubsub.Option functions.
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

// Close closes the GRPC connection to GCP PubSub broker
func (client *Client) Close(ctx context.Context) error {
	return client.subClient.Close()
}
