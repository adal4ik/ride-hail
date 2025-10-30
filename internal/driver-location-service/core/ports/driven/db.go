package driven

import (
	"context"

	"ride-hail/internal/driver-location-service/core/domain/model"
)

type IDriverRepository interface {
	GoOnline(ctx context.Context, coord model.DriverCoordinates) (string, error)
	GoOffline(ctx context.Context, driver_id string) (model.DriverOfflineResponse, error)
	UpdateLocation(ctx context.Context, driver_id string, newLocation model.NewLocation) (model.NewLocationResponse, error)
	StartRide(ctx context.Context, requestData model.StartRide) (model.StartRideResponse, error)
	CompleteRide(ctx context.Context, requestData model.RideCompleteForm) (model.RideCompleteResponse, error)
	FindDrivers(ctx context.Context, longtitude, latitude float64, vehicleType string) ([]model.DriverInfo, error)
	CalculateRideDetails(ctx context.Context, driverLocation model.Location, passagerLocation model.Location) (float64, error)
	UpdateDriverStatus(ctx context.Context, driver_id string, status string) error
	CheckDriverById(ctx context.Context, driver_id string) (bool, error)
	GetDriverIdByRideId(ctx context.Context, ride_id string) (string, error)
	GetRideIdByDriverId(ctx context.Context, driver_id string) (string, error)
	GetRideDetailsByRideId(ctx context.Context, ride_id string) (model.RideDetails, error)
}
