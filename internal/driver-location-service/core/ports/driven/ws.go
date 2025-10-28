package driven

import "context"

type WSConnectionMeneger interface {
	RegisterDriver(ctx context.Context, driverID string, incoming chan<- []byte, outgoing <-chan []byte) error
	UnregisterDriver(ctx context.Context, driverID string)
	IsDriverConnected(driverID string) bool
	SendToDriver(ctx context.Context, driverID, message []byte) error
	BroadcastToDrivers(ctx context.Context, message []byte, driverIDs []string) error
}
