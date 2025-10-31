package ports

import (
	"context"

	messagebrokerdto "ride-hail/internal/ride-service/core/domain/message_broker_dto"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	DriverResponse = "driver.response.*"
	DriverStatus   = "driver.status.*"
)

type IRidesBroker interface {
	Close() error
	PushMessageToRequest(ctx context.Context, message messagebrokerdto.Ride) error
	PushMessageToStatus(ctx context.Context, msg messagebrokerdto.RideStatus) error

	ConsumeMessageFromDrivers(ctx context.Context, queue, driverName string) (<-chan amqp.Delivery, error)
}
