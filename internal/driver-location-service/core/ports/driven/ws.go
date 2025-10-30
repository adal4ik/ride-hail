package driven

import (
	"context"
	"ride-hail/internal/driver-location-service/core/domain/dto"
)

type WSConnectionMeneger interface {
	RegisterDriver(ctx context.Context, driverID string, incoming <-chan []byte, outgoing chan<- []byte) error
	UnregisterDriver(ctx context.Context, driverID string)
	IsDriverConnected(driverID string) bool
	SendToDriver(ctx context.Context, driverID string, message any) error
	GetDriversCount(ctx context.Context) int
	GetDriverMessages(driverID string) (<-chan []byte, error)
	GetConnectedDrivers() []string
	GetFanIn() <-chan dto.DriverMessage
}
