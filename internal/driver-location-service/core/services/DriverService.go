package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ride-hail/internal/driver-location-service/core/domain/dto"
	"ride-hail/internal/driver-location-service/core/domain/model"
	"ride-hail/internal/driver-location-service/core/myerrors"
	"ride-hail/internal/driver-location-service/core/ports/driven"
	"ride-hail/internal/mylogger"

	messagebrokerdto "ride-hail/internal/driver-location-service/core/domain/message_broker_dto"

	websocketdto "ride-hail/internal/driver-location-service/core/domain/websocket_dto"

	ports "ride-hail/internal/driver-location-service/core/ports/driven"
)

const maxPickupDistanceMeters = 100.0

const maxCompleteDistanceMeters = 100.0

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

func (ds *DriverService) StartRide(ctx context.Context, msg dto.StartRide) (dto.StartRideResponse, error) {
	l := ds.log.Action("service.start_ride")
	l.Info("start", "ride_id", msg.Ride_id, "driver_id", msg.Driver_location.Driver_id)

	d, err := ds.repositories.GetDestinationAndDriverCoords(ctx, msg.Ride_id, msg.Driver_location.Driver_id)
	if err != nil {
		if errors.Is(err, myerrors.ErrDBConnClosed) {
			l.Error("Failed to connect to connect to db", err)
			return dto.StartRideResponse{}, myerrors.ErrDBConnClosedMsg
		}
		l.Error("get coords failed", err)
		return dto.StartRideResponse{}, fmt.Errorf("failed to get coordinates: %w", err)
	}
	l.Info("distance calculated", "meters", fmt.Sprintf("%.2f", d))

	// 2️⃣ проверяем порог
	if d > maxPickupDistanceMeters {
		l.Warn("driver too far from pickup", "distance_m", fmt.Sprintf("%.2f", d))
		return dto.StartRideResponse{}, fmt.Errorf("driver too far from pickup (%.1fm > %.0fm)", d, maxPickupDistanceMeters)
	}

	// 3️⃣ запускаем транзакционный апдейт
	res, err := ds.repositories.StartRideTx(ctx, msg.Driver_location.Driver_id, msg.Ride_id)
	if err != nil {
		l.Error("repository.StartRideTx failed", err)
		return dto.StartRideResponse{}, err
	}
	// Driver.statis.{driver_id}
	driverStatus := messagebrokerdto.DriverStatus{
		DriverID:  msg.Driver_location.Driver_id,
		RideID:    msg.Ride_id,
		Status:    "BUSY",
		Timestamp: time.Now().String(),
	}
	ds.broker.PublishJSON(context.Background(), "driver_topic", fmt.Sprintf("driver.status.%s", msg.Driver_location.Driver_id), driverStatus)
	l.Info("Driver status send to rabbitmq", msg.Driver_location.Driver_id)

	l.Info("success", "ride_id", res.Ride_id, "status", res.Status, "started_at", res.Started_at)
	return dto.StartRideResponse{
		Message:    "Ride started successfully",
		Ride_id:    res.Ride_id,
		Status:     res.Status,
		Started_at: res.Started_at,
	}, nil
}

func (ds *DriverService) CompleteRide(ctx context.Context, request dto.RideCompleteForm) (dto.RideCompleteResponse, error) {
	l := ds.log.Action("service.complete_ride")
	l.Info("start", "ride_id", request.Ride_id)

	dId, err := ds.GetDriverIdByRideId(ctx, request.Ride_id)
	if err != nil {
		if errors.Is(err, myerrors.ErrDBConnClosed) {
			l.Error("Failed to connect to connect to db", err)
			return dto.RideCompleteResponse{}, myerrors.ErrDBConnClosedMsg
		}
		return dto.RideCompleteResponse{}, err
	}

	request.FinalLocation.Driver_id = dId
	d, err := ds.repositories.GetDestinationAndDriverCoords(ctx, request.Ride_id, request.FinalLocation.Driver_id)
	if err != nil {
		if errors.Is(err, myerrors.ErrDBConnClosed) {
			l.Error("Failed to connect to connect to db", err)
			return dto.RideCompleteResponse{}, myerrors.ErrDBConnClosedMsg
		}
		l.Error("get destination/driver coords failed", err)
		return dto.RideCompleteResponse{}, fmt.Errorf("failed to get coordinates: %w", err)
	}

	if d > maxCompleteDistanceMeters {
		l.Warn("too far to complete", "distance_m", fmt.Sprintf("%.2f", d))
		return dto.RideCompleteResponse{}, fmt.Errorf("driver too far from destination (%.1fm > %.0fm)", d, maxCompleteDistanceMeters)
	}

	reqDAO := model.RideCompleteForm{
		Ride_id:          request.Ride_id,
		ActualDistancekm: request.ActualDistancekm,
		ActualDurationm:  request.ActualDurationm,
		FinalLocation: model.Location{
			Latitude:  request.FinalLocation.Latitude,
			Longitude: request.FinalLocation.Longitude,
			Driver_id: request.FinalLocation.Driver_id,
		},
	}

	resDAO, err := ds.repositories.CompleteRideTx(ctx, reqDAO)
	if err != nil {
		if errors.Is(err, myerrors.ErrDBConnClosed) {
			l.Error("Failed to connect to connect to db", err)
			return dto.RideCompleteResponse{}, myerrors.ErrDBConnClosedMsg
		}
		l.Error("repository.CompleteRideTx failed", err)
		return dto.RideCompleteResponse{}, err
	}
	// Driver.statis.{driver_id}
	driverStatus := messagebrokerdto.DriverStatus{
		DriverID:  dId,
		RideID:    request.Ride_id,
		Status:    "AVAILABLE",
		Timestamp: time.Now().String(),
	}
	ds.broker.PublishJSON(context.Background(), "driver_topic", fmt.Sprintf("driver.status.%s", dId), driverStatus)
	l.Info("Driver status send to rabbitmq", dId)

	resp := dto.RideCompleteResponse{
		Message:       resDAO.Message,
		Ride_id:       resDAO.Ride_id,
		Status:        resDAO.Status,
		DriverEarning: resDAO.DriverEarning,
		CompletedAt:   resDAO.CompletedAt,
	}
	l.Info("completed", "ride_id", resp.Ride_id, "status", resp.Status)
	return resp, nil
}

func (ds *DriverService) FindAppropriateDrivers(ctx context.Context, longtitude, latitude float64, vehicleType string) ([]dto.DriverInfo, error) {
	drivers, err := ds.repositories.FindDrivers(ctx, longtitude, latitude, vehicleType)
	if err != nil {
		if errors.Is(err, myerrors.ErrDBConnClosed) {
			return []dto.DriverInfo{}, myerrors.ErrDBConnClosedMsg
		}
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
		if errors.Is(err, myerrors.ErrDBConnClosed) {
			return websocketdto.RideDetailsMessage{}, myerrors.ErrDBConnClosedMsg
		}
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

func (d *DriverService) CheckDriverStatus(ctx context.Context, driver_id string) (string, error) {
	return d.repositories.CheckDriverStatus(ctx, driver_id)
}

func (ds *DriverService) RequireActiveRide(ctx context.Context, driverID string) error {
	ok, err := ds.repositories.HasActiveRide(ctx, driverID)
	if err != nil {
		if errors.Is(err, myerrors.ErrDBConnClosed) {
			return myerrors.ErrDBConnClosedMsg
		}
		return err
	}
	if !ok {
		return model.ErrNoActiveRide
	}
	return nil
}

func (ds *DriverService) PayDriverMoney(ctx context.Context, driver_id string, amount float64) error {
	return ds.repositories.PayDriverMoney(ctx, driver_id, amount)
}
