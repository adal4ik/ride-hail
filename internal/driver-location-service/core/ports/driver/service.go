package driver

import (
	"context"
	"ride-hail/internal/driver-location-service/core/domain/dto"
)

type IDriverService interface {
	GoOnline(ctx context.Context, coord dto.DriverCoordinatesDTO) (dto.DriverOnlineResponse, error)
	GoOffline(ctx context.Context, driver_id string) (dto.DriverOfflineRespones, error)
	UpdateLocation()
	StartRide()
	CompleteRide()
}
