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

package natsdriver

import (
	"context"

	"github.com/nats-io/nats.go"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/natspubsub"
)

// Client handles the subscriptions to NATS subjects
type Client struct {
	conn *nats.Conn
}

// Option is a self-refrential function for configuration parameters
type Option func(*Client) error

// NewClient returns *natsdriver.Client, manages subscription to NATS subjects.
//
// Additional configuration options can be added with natsdriver.Option functions.
func NewClient(ctx context.Context, conn *nats.Conn, opts ...Option) (*Client, error) {

	c := &Client{conn: conn}

	for _, option := range opts {
		if err := option(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// Subscribe subscribes to the NATS subject
func (client *Client) Subscribe(ctx context.Context, subject string) (*pubsub.Subscription, error) {
	return natspubsub.OpenSubscription(client.conn, subject, nil)
}

// Close closes the connection to NATS server
func (client *Client) Close(ctx context.Context) error {
	client.conn.Close()
	return nil
}
