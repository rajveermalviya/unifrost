package drivers

import (
	"context"
	"errors"

	"gocloud.dev/pubsub"
)

// SubscriberClient connects to the specific pubsub vendor using specific drivers,
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
