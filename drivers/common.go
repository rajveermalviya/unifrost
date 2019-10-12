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

// Driver is an enum type for indentifying all the drivers
type Driver int

const (
	// DriverMem is an identifier for in-memory driver
	DriverMem Driver = iota + 1
	// DriverGCP is an identifier for GCP PubSub driver
	DriverGCP
	// DriverRabbit is an identifier for RabbitMQ driver
	DriverRabbit
	// DriverNats is an identifier for NATS driver
	DriverNats
	// DriverKafka is an identifier for Kafka driver
	DriverKafka
	// DriverAWS
	// DriverAzure
)

var (
	// ErrInvalidDriver error
	ErrInvalidDriver = errors.New("driver: Invalid driver specified")
	// ErrInvalidConfigSignature error
	ErrInvalidConfigSignature = errors.New("driver: Invalid config signature for specifie driver")
)

// NewSubscriberClient is a function for creating a specific SubscriberClient instance.
func NewSubscriberClient(ctx context.Context, driver Driver, config interface{}) (SubscriberClient, error) {
	switch driver {
	case DriverGCP:
		c, ok := config.(*ConfigGCPPubSub)
		if !ok {
			return nil, ErrInvalidConfigSignature
		}
		return NewClientGCPPubSub(ctx, c)

	case DriverKafka:
		c, ok := config.(*ConfigKafka)
		if !ok {
			return nil, ErrInvalidConfigSignature
		}
		return NewClientKafka(ctx, c)

	case DriverNats:
		c, ok := config.(*ConfigNats)
		if !ok {
			return nil, ErrInvalidConfigSignature
		}
		return NewClientNats(ctx, c)

	case DriverRabbit:
		c, ok := config.(*ConfigRabbitMQ)
		if !ok {
			return nil, ErrInvalidConfigSignature
		}
		return NewClientRabbitMQ(ctx, c)

	case DriverMem:
		c, ok := config.(*ConfigMem)
		if !ok {
			return nil, ErrInvalidConfigSignature
		}
		return NewClientMem(ctx, c)

	default:
		return nil, ErrInvalidDriver
	}
}
