package driven

import (
	"context"
	"ride-hail/internal/driver-location-service/core/domain/model"
)

type IDriverRepository interface {
	GoOnline(ctx context.Context, coord model.DriverCoordinates) (string, error)
	GoOffline(ctx context.Context, driver_id string) (model.DriverOfflineResponse, error)
	UpdateLocation()
	StartRide(ctx context.Context, requestData model.StartRide) (model.StartRideResponse, error)
	CompleteRide()
}
