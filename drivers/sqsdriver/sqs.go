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

// Package sqsdriver contains Amazon SQS driver for unifrost.StreamHandler
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

// Option is a self-refrential function for configuration parameters
type Option func(*Client) error

// NewClient returns *sqsdriver.Client, establishes session to the AWS cloud service.
//
// Additional configuration options can be added with sqsdriver.Option functions.
func NewClient(ctx context.Context, session *session.Session, opts ...Option) (*Client, error) {
	c := &Client{session: session}
	for _, option := range opts {
		if err := option(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// Subscribe subscribes to the given SQS url
//
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
