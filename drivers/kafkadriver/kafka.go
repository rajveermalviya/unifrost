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

// Package kafkadriver contains Apache Kafka message bus driver for unifrost.StreamHandler
package kafkadriver

import (
	"context"

	"github.com/Shopify/sarama"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/kafkapubsub"
)

// Client handles the subscriptions to Kafka broker topics
type Client struct {
	config  *sarama.Config
	groupID string
	brokers []string
}

// Option is a self-refrential function for configuration parameters
type Option func(*Client) error

// NewClient returns *kafkadriver.Client, also connects to the kafka cluster.
//
// This driver uses https://github.com/Shopify/sarama internally to connect to kafka brokers.
// A slice of broker addresses are required by sarama to connect to the kafka cluster, specify only one if single node.
// Configuration is handled bysarama.Config to subscribe to a kafka topic. https://godoc.org/github.com/Shopify/sarama#Config
// groupID is consumer group id.
// Additional configuration options can be added with kafkadriver.Option functions.
func NewClient(
	ctx context.Context,
	brokers []string,
	groupID string,
	config *sarama.Config,
	opts ...Option,
) (*Client, error) {

	c := &Client{config: config, brokers: brokers, groupID: groupID}

	for _, option := range opts {
		if err := option(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// Subscribe method subscribes to the kafka topic
func (client *Client) Subscribe(ctx context.Context, topic string) (*pubsub.Subscription, error) {
	return kafkapubsub.OpenSubscription(client.brokers, client.config, client.groupID, []string{topic}, nil)
}

// Close is just a placeholder
func (client *Client) Close(ctx context.Context) error {
	return nil
}
