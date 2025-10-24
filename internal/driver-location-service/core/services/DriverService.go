package services

import (
	"context"

	"ride-hail/internal/driver-location-service/core/domain/dto"
	"ride-hail/internal/driver-location-service/core/domain/model"
	"ride-hail/internal/driver-location-service/core/ports/driven"
	ports "ride-hail/internal/driver-location-service/core/ports/driven"
	"ride-hail/internal/mylogger"
)

type DriverService struct {
	repositories driven.IDriverRepository
	log          *mylogger.Logger
	broker       ports.IDriverBroker
}

func NewDriverService(repositories driven.IDriverRepository, log *mylogger.Logger, broker ports.IDriverBroker) *DriverService {
	return &DriverService{repositories: repositories, log: log, broker: broker}
}

func (ds *DriverService) GoOnline(ctx context.Context, coordDTO dto.DriverCoordinatesDTO) (dto.DriverOnlineResponse, error) {
	var response dto.DriverOnlineResponse
	var coord model.DriverCoordinates
	coord.Driver_id = coordDTO.Driver_id
	coord.Latitude = coordDTO.Latitude
	coord.Longitude = coordDTO.Longitude

	session_id, err := ds.repositories.GoOnline(ctx, coord)
	if err != nil {
		return dto.DriverOnlineResponse{}, err
	}
	response.Session_id = session_id
	response.Status = "AVAILABLE"
	response.Message = "You are now online and ready to accept rides"
	return response, nil
}

func (ds *DriverService) GoOffline(ctx context.Context, driver_id string) (dto.DriverOfflineRespones, error) {
	results, err := ds.repositories.GoOffline(ctx, driver_id)
	if err != nil {
		return dto.DriverOfflineRespones{}, err
	}
	var response dto.DriverOfflineRespones
	response.Session_id = results.Session_id
	response.Status = "OFFLINE"
	response.Message = "You are now offline"
	response.Session_summary.Duration_hours = results.Session_summary.Duration_hours
	response.Session_summary.Earnings = results.Session_summary.Earnings
	response.Session_summary.Rides_completed = results.Session_summary.Rides_completed
	return response, nil
}

func (ds *DriverService) UpdateLocation(ctx context.Context, request dto.NewLocation, driver_id string) (dto.NewLocationResponse, error) {
	var requestDAO model.NewLocation
	requestDAO.Accuracy_meters = request.Accuracy_meters
	requestDAO.Heading_Degrees = request.Heading_Degrees
	requestDAO.Latitude = request.Latitude
	requestDAO.Longitude = request.Longitude
	requestDAO.Speed_kmh = request.Speed_kmh
	response, err := ds.repositories.UpdateLocation(ctx, driver_id, requestDAO)
	if err != nil {
		return dto.NewLocationResponse{}, err
	}
	var responseDTO dto.NewLocationResponse
	responseDTO.Coordinate_id = response.Coordinate_id
	responseDTO.Updated_at = response.Updated_at
	return responseDTO, nil
}

func (ds *DriverService) StartRide(ctx context.Context, requestMessage dto.StartRide) (dto.StartRideResponse, error) {
	var requestedData model.StartRide
	requestedData.Ride_id = requestMessage.Ride_id
	requestedData.Driver_location.Driver_id = requestMessage.Driver_location.Driver_id
	requestedData.Driver_location.Latitude = requestMessage.Driver_location.Latitude
	requestedData.Driver_location.Longitude = requestMessage.Driver_location.Longitude
	results, err := ds.repositories.StartRide(ctx, requestedData)
	if err != nil {
		return dto.StartRideResponse{}, err
	}

	var response dto.StartRideResponse
	response.Message = "Ride started successfully"
	response.Ride_id = results.Ride_id
	response.Started_at = results.Started_at
	response.Status = results.Status
	return response, nil
}

func (ds *DriverService) CompleteRide(ctx context.Context, request dto.RideCompleteForm) (dto.RideCompleteResponse, error) {
	var requestDAO model.RideCompleteForm
	requestDAO.Ride_id = request.Ride_id
	requestDAO.ActualDistancekm = request.ActualDistancekm
	requestDAO.ActualDurationm = request.ActualDurationm
	requestDAO.FinalLocation.Latitude = request.FinalLocation.Latitude
	requestDAO.FinalLocation.Longitude = request.FinalLocation.Longitude
	results, err := ds.repositories.CompleteRide(ctx, requestDAO)
	if err != nil {
		return dto.RideCompleteResponse{}, err
	}
	var response dto.RideCompleteResponse
	response.Message = results.Message
	response.Ride_id = results.Ride_id
	response.Status = results.Status
	response.DriverEarning = results.DriverEarning
	response.CompletedAt = results.CompletedAt
	return response, nil
}
