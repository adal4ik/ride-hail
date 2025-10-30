package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"ride-hail/internal/mylogger"
	"ride-hail/internal/ride-service/core/domain/dto"
	"ride-hail/internal/ride-service/core/domain/model"
	websocketdto "ride-hail/internal/ride-service/core/domain/websocket_dto"
	"ride-hail/internal/ride-service/core/ports"

	messagebrokerdto "ride-hail/internal/ride-service/core/domain/message_broker_dto"
)

const (
	DEFUALT_RATE_PER_MIN = 40

	ECONOMY = "ECONOMY"
	PREMIUM = "PREMIUM"
	XL      = "XL"

	ECONOMY_BASE = 500 // base
	PREMIUM_BASE = 800
	XL_BASE      = 1000

	ECONOMY_RATE_PER_KM = 100 // 100₸/km
	PREMIUM_RATE_PER_KM = 120
	XL_RATE_PER_KM      = 150

	ECONOMY_RATE_PER_MIN = 50 // 50₸/min
	PREMIUM_RATE_PER_MIN = 60
	XL_RATE_PER_MIN      = 75
)

type RidesService struct {
	mylog          mylogger.Logger
	RidesRepo      ports.IRidesRepo
	RidesBroker    ports.IRidesBroker
	RidesWebsocket ports.INotifyWebsocket
	ctx            context.Context
}

func NewRidesService(ctx context.Context,
	log mylogger.Logger,
	RidesRepo ports.IRidesRepo,
	RidesBroker ports.IRidesBroker,
	RidesWebsocket ports.INotifyWebsocket,
) ports.IRidesService {
	return &RidesService{
		ctx:            ctx,
		mylog:          log,
		RidesRepo:      RidesRepo,
		RidesBroker:    RidesBroker,
		RidesWebsocket: RidesWebsocket,
	}
}

// implement me
func (rs *RidesService) CreateRide(req dto.RidesRequestDto) (dto.RidesResponseDto, error) {
	m := model.Rides{}
	log := rs.mylog.Action("CreateRide")

	if err := validateRideRequest(req); err != nil {
		return dto.RidesResponseDto{}, err
	}

	ctx, cancel := context.WithTimeout(rs.ctx, time.Second*15)
	defer cancel()
	// estimate distance between pick up and destination points
	distance, err := rs.RidesRepo.GetDistance(ctx, req)
	if err != nil {
		log.Error("cannot get distance between two points", err)
		return dto.RidesResponseDto{}, err
	}

	// only for ride-number
	numberOfRides, err := rs.RidesRepo.GetNumberRides(ctx)
	if err != nil {
		log.Error("cannot get number of rides", err)
		return dto.RidesResponseDto{}, err
	}

	RideNumber := fmt.Sprintf("RIDE_%d%d%d_%0*d", time.Now().Year(), time.Now().Month(), time.Now().Day(), 3, numberOfRides+1)

	var (
		EstimatedFare float64 = 0
		Priority      int     = 1
	)

	switch *req.RideType {
	case ECONOMY:
		EstimatedFare = ECONOMY_BASE + (distance * ECONOMY_RATE_PER_KM) + (DEFUALT_RATE_PER_MIN * ECONOMY_RATE_PER_MIN)
	case PREMIUM:
		EstimatedFare = PREMIUM_BASE + (distance * PREMIUM_RATE_PER_KM) + (DEFUALT_RATE_PER_MIN * PREMIUM_RATE_PER_MIN)
	case XL:
		EstimatedFare = XL_BASE + (distance * XL_RATE_PER_KM) + (DEFUALT_RATE_PER_MIN * XL_RATE_PER_MIN)
	default:
		log.Warn("unkown ride type", "type", req.RideType)
		return dto.RidesResponseDto{}, fmt.Errorf("unkown ride type")
	}

	// PRIORITY estimate
	if EstimatedFare >= 10000 {
		Priority = 10
	} else if EstimatedFare <= 1000 {
		Priority = 1
	} else {
		Priority = int(EstimatedFare) / 1000
	}
	// math.Round()

	m = model.Rides{
		RideNumber:    RideNumber,
		PassengerId:   *req.PassengerId,
		Status:        "REQUESTED",
		EstimatedFare: EstimatedFare,
		FinalFare:     EstimatedFare,
		Priority:      Priority,
	}

	m.PickupCoordinate = model.Coordinates{
		EntityId:        *req.PassengerId,
		EntityType:      "PASSENGER",
		Address:         *req.PickUpAddress,
		Latitude:        *req.PickUpLatitude,
		Longitude:       *req.PickUpLongitude,
		FareAmount:      m.EstimatedFare,
		DistanceKm:      distance,
		DurationMinutes: 0,
		IsCurrent:       true,
	}
	m.DestinationCoordinate = model.Coordinates{
		EntityId:        *req.PassengerId,
		EntityType:      "PASSENGER",
		Address:         *req.DestinationAddress,
		Latitude:        *req.DestinationLatitude,
		Longitude:       *req.DestinationLongitude,
		FareAmount:      m.EstimatedFare,
		DistanceKm:      distance,
		DurationMinutes: 0,
		IsCurrent:       true,
	}
	log.Info("creating a ride", "RideNumber", RideNumber, "passenger-id", req.PassengerId, "estimated-fare", EstimatedFare, "distance", distance)
	ctx, cancel = context.WithTimeout(rs.ctx, time.Second*15)
	defer cancel()
	ride_id, err := rs.RidesRepo.CreateRide(ctx, m)
	if err != nil {
		return dto.RidesResponseDto{}, err
	}

	// publish message to rabbitmq
	log.Info("Inserting ride to BM")

	log.Debug("Debugging", "RideNumber", RideNumber)

	rideMsg := messagebrokerdto.Ride{
		RideID:         ride_id,
		RideNumber:     RideNumber,
		RideType:       *req.RideType,
		EstimatedFare:  EstimatedFare,
		MaxDistanceKm:  distance,
		TimeoutSeconds: 30,
		Priority:       Priority,
		CorrelationID:  generateCorrelationID(),
	}

	rideMsg.PickupLocation = messagebrokerdto.Location{
		Lat:     *req.PickUpLatitude,
		Lng:     *req.PickUpLongitude,
		Address: *req.PickUpAddress,
	}

	rideMsg.DestinationLocation = messagebrokerdto.Location{
		Lat:     *req.DestinationLatitude,
		Lng:     *req.DestinationLongitude,
		Address: *req.DestinationAddress,
	}

	if err := rs.RidesBroker.PushMessageToRequest(rs.ctx, rideMsg); err != nil {
		log.Error("Failed to publish message", err)
		return dto.RidesResponseDto{}, fmt.Errorf("cannot send message to broker: %w", err)
	}

	log.Info("successfully created a ride", "ride-id", ride_id)
	res := dto.RidesResponseDto{
		RideId:                   ride_id,
		RideNumber:               RideNumber,
		Status:                   "REQUESTED",
		EstimatedFare:            EstimatedFare,
		EstimatedDistanceKm:      distance,
		EstimatedDurationMinutes: distance * 1000 / DEFUALT_RATE_PER_MIN,
	}
	return res, nil
}

var (
	ErrEmptyField       = errors.New("field id empty")
	ErrInvalidLatitute  = errors.New("invalid latititude [-90, 90]")
	ErrInvalidLongitude = errors.New("invalid longitude  [-180, 180]")
	ErrInvalidAdress    = errors.New("maximum 255 characters allowed")
)

func validateRideRequest(req dto.RidesRequestDto) error {
	if err := validatePassengerId(req.PassengerId); err != nil {
		return fmt.Errorf("invalid passenger id: %v", err)
	}

	// ctx, cancel := context.WithTimeout(rs.ctx, time.Second*15)
	// defer cancel()

	// count, err := rs.RidesRepo.CheckDuplicate(ctx, req.PassengerId)

	// if err != nil {
	// 	log.Error("cannot check for duplication", err)
	// 	return dto.RidesResponseDto{}, err
	// }

	// if count > 0 {
	// 	return dto.RidesResponseDto{}, fmt.Errorf("cannot create duplicated ride")
	// }

	if err := validateLatLng(req.PickUpLatitude, req.PickUpLongitude); err != nil {
		return fmt.Errorf("invalid pickup coords: %v", err)
	}
	if err := validateAddress(req.PickUpAddress); err != nil {
		return fmt.Errorf("invalid pickup address: %v", err)
	}

	if err := validateLatLng(req.DestinationLatitude, req.DestinationLongitude); err != nil {
		return fmt.Errorf("invalid destination coords: %v", err)
	}
	if err := validateAddress(req.DestinationAddress); err != nil {
		return fmt.Errorf("invalid destination address: %v", err)
	}

	if err := validateRideType(req.RideType); err != nil {
		return fmt.Errorf("invalid ride type: %v", err)
	}

	return nil
}

func validatePassengerId(passengerId *string) error {
	if passengerId == nil || *passengerId == "" {
		return ErrEmptyField
	}

	return nil
}

func validateLatLng(lat, lng *float64) error {
	if lat == nil || lng == nil {
		return ErrEmptyField
	}

	if lat == nil || math.Abs(*lat) > 90 {
		return ErrInvalidLatitute
	}
	if lng == nil || math.Abs(*lng) > 180 {
		return ErrInvalidLongitude
	}

	return nil
}

func validateAddress(s *string) error {
	if s == nil {
		return ErrEmptyField
	}

	if len(*s) > 255 {
		return ErrInvalidAdress
	}
	return nil
}

func getAllowedRideTypes() []string {
	return []string{"ECONOMY", "PREMIUM", "XL"}
}

var AllowedRideTypes = map[string]bool{
	"ECONOMY": true,
	"PREMIUM": true,
	"XL":      true,
}

func validateRideType(s *string) error {
	if s == nil || *s == "" {
		return ErrEmptyField
	}
	sn := strings.ToUpper(*s)
	if ok := AllowedRideTypes[sn]; !ok {
		return fmt.Errorf("unknown ride type. Allowed ride types are: %v", getAllowedRideTypes())
	}
	return nil
}

func (rs *RidesService) CancelRide(req dto.RidesCancelRequestDto, rideId string) (dto.RideCancelResponseDto, error) {
	log := rs.mylog.Action("CreateRide")

	ctx, cancel := context.WithTimeout(rs.ctx, time.Second*15)
	defer cancel()

	log.Info("params", "rideId", rideId, "reason", req.Reason)

	driverId, err := rs.RidesRepo.CancelRide(ctx, rideId, req.Reason)
	if err != nil {
		log.Error("Failed to cancel ride", err)
		return dto.RideCancelResponseDto{}, err
	}
	log.Info("Ride cancelled successfully")

	cancelledAt := time.Now().Format(time.RFC3339)

	res := dto.RideCancelResponseDto{
		RideId:      rideId,
		Status:      "CANCELLED",
		CancelledAt: cancelledAt,
		Message:     "Ride cancelled successfully",
	}

	if driverId != "" {
		m2 := messagebrokerdto.RideStatus{
			RideId:    rideId,
			Status:    "CANCELLED",
			Timestamp: cancelledAt,
			DriverID:  driverId,
		}

		ctx, cancel = context.WithTimeout(rs.ctx, time.Second*15)
		defer cancel()

		err = rs.RidesBroker.PushMessageToStatus(ctx, m2)
		if err != nil {
			return dto.RideCancelResponseDto{}, err
		}
	}

	return res, nil
}

func (rs *RidesService) SetStatusMatch(rideId, driverId string) (string, string, error) {
	ctx, cancel := context.WithTimeout(rs.ctx, time.Second*15)
	defer cancel()
	log := rs.mylog.Action("SetStatusMatch")
	log.Info("sex", "rideId", rideId, "driverId", driverId)
	passengerId, rideNumber, err := rs.RidesRepo.ChangeStatusMatch(ctx, rideId, driverId)
	if err != nil {
		// TODO: add handle error
		return "", "", err
	}
	m2 := messagebrokerdto.RideStatus{
		RideId:    rideId,
		Status:    "IN_PROGRESS",
		Timestamp: time.Now().Format(time.RFC3339),
		DriverID:  driverId,
	}
	ctx, cancel = context.WithTimeout(rs.ctx, time.Second*15)
	defer cancel()

	err = rs.RidesBroker.PushMessageToStatus(ctx, m2)
	if err != nil {
		return "", "", err
	}

	return passengerId, rideNumber, nil
}

func (rs *RidesService) EstimateDistance(rideId string, longitude, latitude, speed float64) (string, string, float64, error) {
	log := rs.mylog.Action("FindPassenger")

	ctx, cancel := context.WithTimeout(rs.ctx, time.Second*5)
	defer cancel()

	distance, passengerId, err := rs.RidesRepo.FindDistanceAndPassengerId(ctx, longitude, latitude, rideId)
	if err != nil {
		log.Error("cannot get user or something", err)
		return "", "", 0.0, err
	}
	if IsCloseToZero(speed) {
		speed = DEFUALT_RATE_PER_MIN
	}

	t := time.Now().Add(time.Duration(distance / speed)).Format(time.RFC3339)

	return passengerId, t, distance, nil
}

func (rs *RidesService) CancelEveryPossibleRides() error {
	log := rs.mylog.Action("CancelEveryPossibleRides")
	ctx, cancel := context.WithTimeout(rs.ctx, time.Second*5)
	defer cancel()

	err := rs.RidesRepo.CancelEveryPossibleRides(ctx)
	if err != nil {
		log.Error("cannot cancel every rides", err)
		return err
	}
	log.Info("canceled every possible rides")
	return nil
}

// Generate a new UUID as a correlation ID
func generateCorrelationID() string {
	// Define the character set (lowercase, uppercase, and digits)
	charSet := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	// Seed the random number generator for true randomness
	rand.New(rand.NewSource(time.Now().UnixNano()))

	// Generate a random length for the string, ensuring at least 3 characters
	n := rand.Intn(3) + 3

	// Pre-allocate the slice for the correlation ID
	b := make([]rune, n)

	// Create the random part of the ID
	for i := range b {
		b[i] = charSet[rand.Intn(len(charSet))]
	}
	return "req_" + string(b)
}

const Epsilon = 1e-9

func IsCloseToZero(f float64) bool {
	return math.Abs(f) < Epsilon
}

// type DriverInfo struct {
// 	DriverID string  `json:"driver_id"`
// 	Name     string  `json:"name"`
// 	Rating   float64 `json:"rating"`
// 	Vehicle  Vehicle `json:"vehicle"`
// }

// // To Passenger - Match Notification:
// type RideStatusUpdateDto struct {
// 	RideID        string     `json:"ride_id"`
// 	RideNumber    string     `json:"ride_number"`
// 	Status        string     `json:"status"`
// 	DriverInfo    DriverInfo `json:"driver_info"`
// 	CorrelationID string     `json:"correlation_id"`
// }

func (ps *RidesService) UpdateRideStatus(msg messagebrokerdto.DriverStatusUpdate) (string, websocketdto.Event, error) {
	log := ps.mylog.Action("UpdateRideStatus")

	ctx, cancel := context.WithTimeout(ps.ctx, time.Second*15)
	defer cancel()

	passengerId, rideNumber, driverInfo, err := ps.RidesRepo.ChangeStatus(ctx, msg)
	if err != nil {
		log.Error("Failed to cancel ride", err)
		return "", websocketdto.Event{}, err
	}

	data := websocketdto.RideStatusUpdateDto{
		RideID:        msg.RideId,
		Status:        msg.Status,
		CorrelationID: generateCorrelationID(),
		DriverInfo:    driverInfo,
		RideNumber:    rideNumber,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return "", websocketdto.Event{}, err
	}

	res := websocketdto.Event{
		Type: "ride_status_update",
		Data: jsonData,
	}

	return passengerId, res, nil
}
