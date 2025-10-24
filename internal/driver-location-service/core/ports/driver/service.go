package driver

import (
	"context"

	"ride-hail/internal/driver-location-service/core/domain/dto"
)

type IDriverService interface {
	GoOnline(ctx context.Context, coord dto.DriverCoordinatesDTO) (dto.DriverOnlineResponse, error)
	GoOffline(ctx context.Context, driver_id string) (dto.DriverOfflineRespones, error)
	UpdateLocation(ctx context.Context, request dto.NewLocation, driver_id string) (dto.NewLocationResponse, error)
	StartRide(ctx context.Context, requestMessage dto.StartRide) (dto.StartRideResponse, error)
	CompleteRide(ctx context.Context, request dto.RideCompleteForm) (dto.RideCompleteResponse, error)
}
