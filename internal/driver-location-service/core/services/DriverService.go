package services

import (
	"context"
	"encoding/json"
	"fmt"

	"ride-hail/internal/driver-location-service/core/domain/dto"
	"ride-hail/internal/driver-location-service/core/domain/model"
	websocketdto "ride-hail/internal/driver-location-service/core/domain/websocket_dto"
	"ride-hail/internal/driver-location-service/core/ports/driven"
	ports "ride-hail/internal/driver-location-service/core/ports/driven"
	"ride-hail/internal/mylogger"
)

type DriverService struct {
	repositories driven.IDriverRepository
	log          mylogger.Logger
	broker       ports.IDriverBroker
}

func NewDriverService(repositories driven.IDriverRepository, log mylogger.Logger, broker ports.IDriverBroker) *DriverService {
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

func (ds *DriverService) FindAppropriateDrivers(ctx context.Context, longtitude, latitude float64, vehicleType string) ([]dto.DriverInfo, error) {
	drivers, err := ds.repositories.FindDrivers(ctx, longtitude, latitude, vehicleType)
	if err != nil {
		fmt.Println("Service Error Arrived ", err)
		return []dto.DriverInfo{}, err
	}
	var results []dto.DriverInfo
	for _, driver := range drivers {
		var result dto.DriverInfo
		result.DriverId = driver.DriverId
		result.Email = driver.Email
		result.Latitude = driver.Latitude
		result.Longitude = driver.Longitude
		result.Rating = driver.Rating
		result.Name = driver.Name
		result.Distance = driver.Distance
		if err := json.Unmarshal(driver.Vehicle, &result.Vehicle); err != nil {
			fmt.Println("Service Error Arrived ", err)
			return []dto.DriverInfo{}, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (ds *DriverService) CalculateRideDetails(ctx context.Context, driverLocation dto.Location, passagerLocation dto.Location) (float64, int, error) {
	distance, err := ds.repositories.CalculateRideDetails(ctx,
		model.Location{
			Latitude:  driverLocation.Latitude,
			Longitude: driverLocation.Longitude,
		},
		model.Location{
			Latitude:  passagerLocation.Latitude,
			Longitude: passagerLocation.Longitude,
		},
	)
	if err != nil {
		return 0, 0, err
	}
	minutes := int(distance / 45)
	return distance, minutes, nil
}

func (d *DriverService) UpdateDriverStatus(ctx context.Context, driver_id string, status string) error {
	return d.repositories.UpdateDriverStatus(ctx, driver_id, status)
}

func (d *DriverService) CheckDriverById(ctx context.Context, driver_id string) (bool, error) {
	return d.repositories.CheckDriverById(ctx, driver_id)
}

func (d *DriverService) GetDriverIdByRideId(ctx context.Context, ride_id string) (string, error) {
	// This is a placeholder implementation. Replace with actual logic to get driver ID by ride ID.
	// For example, you might query the database to find the driver associated with the given ride ID.
	return d.repositories.GetDriverIdByRideId(ctx, ride_id)
}

func (d *DriverService) GetRideIdByDriverId(ctx context.Context, driver_id string) (string, error) {
	// This is a placeholder implementation. Replace with actual logic to get ride ID by driver ID.
	return d.repositories.GetRideIdByDriverId(ctx, driver_id)
}

func (d *DriverService) GetRideDetailsByRideId(ctx context.Context, ride_id string) (websocketdto.RideDetailsMessage, error) {
	// This is a placeholder implementation. Replace with actual logic to get ride details by ride ID.
	// For example, you might query the database to find the ride details associated with the given ride ID.
	rideDetailsModel, err := d.repositories.GetRideDetailsByRideId(ctx, ride_id)
	fmt.Println("Ride Details Model: ", rideDetailsModel)
	fmt.Println("User phone", string(rideDetailsModel.PassengerAttrs))
	if err != nil {
		return websocketdto.RideDetailsMessage{}, err
	}
	var rideDetails websocketdto.RideDetailsMessage
	rideDetails.RideID = rideDetailsModel.Ride_id
	rideDetails.PassengerName = rideDetailsModel.PassengerName
	rideDetails.PickupLocation = websocketdto.Location{
		Latitude:  rideDetailsModel.PickupLocation.Latitude,
		Longitude: rideDetailsModel.PickupLocation.Longitude,
		Address:   rideDetailsModel.PickupLocation.Address,
	}
	tempStruct := struct {
		PhoneNumer string `json:"phone"`
	}{}
	if err := json.Unmarshal(rideDetailsModel.PassengerAttrs, &tempStruct); err != nil {
		return websocketdto.RideDetailsMessage{}, err
	}
	rideDetails.PassengerPhone = tempStruct.PhoneNumer
	return rideDetails, nil
}
