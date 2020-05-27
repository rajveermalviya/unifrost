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

package rabbitdriver

import (
	"context"

	"github.com/streadway/amqp"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/rabbitpubsub"
)

// Client handles the subscriptions to rabbit amqp topics
type Client struct {
	conn *amqp.Connection
}

// Option is a self-refrential function for configuration parameters
type Option func(*Client) error

// NewClient returns *rabbitmq.Client, also creates a RabbitMQ broker subscription client.
//
// Additional configuration options can be added with rabbitpubsub.Option functions.
func NewClient(ctx context.Context, rabbitConn *amqp.Connection, opts ...Option) (*Client, error) {
	return &Client{conn: rabbitConn}, nil
}

// Subscribe method subscribes to the rabbit amqp topic
func (client *Client) Subscribe(ctx context.Context, topic string) (*pubsub.Subscription, error) {
	return rabbitpubsub.OpenSubscription(client.conn, topic, nil), nil
}

// Close closes the connection to rabbit amqp server
func (client *Client) Close(ctx context.Context) error {
	return client.conn.Close()
}
