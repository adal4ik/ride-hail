package driven

import (
	"context"
	messagebrokerdto "ride-hail/internal/driver-location-service/core/domain/message_broker_dto"
)

type IDriverBroker interface {
	Close() error
	PushMessage(ctx context.Context, message messagebrokerdto.Ride) error
}
