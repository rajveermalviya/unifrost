package drivers

import (
	"context"
	"errors"

	"gocloud.dev/pubsub"
)

// SubscriberClient
type SubscriberClient interface {
	Close(context.Context) error
	Subscribe(context.Context, string) (*pubsub.Subscription, error)
}

type Driver int

const (
	DriverMem Driver = iota + 1
	DriverGCP
	DriverRabbit
	DriverNats
	DriverKafka
	// DriverAWS
	// DriverAzure
)

var (
	ErrInvalidDriver          = errors.New("driver: Invalid driver specified")
	ErrInvalidConfigSignature = errors.New("driver: Invalid config signature for specifie driver")
)

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
