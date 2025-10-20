package ports

import (
	"context"
	messagebrokerdto "ride-hail/internal/ride-service/core/domain/message_broker_dto"
)

type IRidesBroker interface {
	Close() error
	PushMessage(ctx context.Context, message messagebrokerdto.Ride) error
}
