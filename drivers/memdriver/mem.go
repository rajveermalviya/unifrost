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

package memdriver

import (
	"context"

	"gocloud.dev/pubsub"

	// required for initializing mempubsub url opener
	_ "gocloud.dev/pubsub/mempubsub"
)

// Client handles the subscriptions to In-Memory topics
type Client struct{}

// Subscribe method subscribes to the In-Memory topic
func (client *Client) Subscribe(ctx context.Context, topic string) (*pubsub.Subscription, error) {
	return pubsub.OpenSubscription(ctx, "mem://"+topic)
}

// Close is just a placeholder
func (client *Client) Close(ctx context.Context) error {
	return nil
}
