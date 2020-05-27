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

package drivers

import (
	"context"
	"errors"

	"gocloud.dev/pubsub"
)

// SubscriberClient connects to the specific pubsub broker using specific drivers,
// it also hides the specific driver implementation for subscribing to topics
// And gives a single interface for all driver types.
type SubscriberClient interface {
	// Subscribe method subscribes to the specified topic
	// and returns the gocloud pubsub Subscription
	Subscribe(context.Context, string) (*pubsub.Subscription, error)

	Close(context.Context) error
}

var (
	// ErrInvalidDriver error
	ErrInvalidDriver = errors.New("driver: Invalid driver specified")
	// ErrInvalidConfigSignature error
	ErrInvalidConfigSignature = errors.New("driver: Invalid config signature for specifie driver")
)
