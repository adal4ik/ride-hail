package ports

import (
	"context"
	messagebroker "ride-hail/internal/ride-service/core/domain/message_broker_dto"

	amqp "github.com/rabbitmq/amqp091-go"
)

type IRidesBroker interface {
	Close() error
	PushMessageToDrivers(ctx context.Context, message messagebroker.Ride) error
	ConsumeMessageFromDrivers(ctx context.Context, queue, driverName string) (<-chan amqp.Delivery, error)
}
