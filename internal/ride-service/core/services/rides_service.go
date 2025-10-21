package services

import (
	"context"
	"fmt"
	"time"

	"ride-hail/internal/mylogger"
	"ride-hail/internal/ride-service/core/domain/dto"
	messagebrokerdto "ride-hail/internal/ride-service/core/domain/message_broker_dto"
	"ride-hail/internal/ride-service/core/domain/model"
	"ride-hail/internal/ride-service/core/ports"
)

const (
	DEFUALT_RATE_PER_MIN = 60

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
	RidesWebsocket ports.IRidesWebsocket
	ctx            context.Context
}

func NewRidesService(ctx context.Context,
	mylog mylogger.Logger,
	RidesRepo ports.IRidesRepo,
	RidesBroker ports.IRidesBroker,
	RidesWebsocket ports.IRidesWebsocket,
) ports.IRidesService {
	return &RidesService{
		ctx:            ctx,
		mylog:          mylog,
		RidesRepo:      RidesRepo,
		RidesBroker:    RidesBroker,
		RidesWebsocket: RidesWebsocket,
	}
}

func (rs *RidesService) CreateRide(req dto.RidesRequestDto) (dto.RidesResponseDto, error) {
	m := model.Rides{}
	log := rs.mylog.Action("CreateRide")
	ctx, cancel := context.WithTimeout(rs.ctx, time.Second*15)
	defer cancel()

	distance, err := rs.RidesRepo.GetDistance(ctx, req)
	if err != nil {
		log.Error("cannot get distance between two points", err)
		return dto.RidesResponseDto{}, err
	}
	numberOfRides, err := rs.RidesRepo.GetNumberRides(ctx)
	if err != nil {
		log.Error("cannot get number of rides", err)
		return dto.RidesResponseDto{}, err
	}

	RideNumber := fmt.Sprintf("RIDE_%d%d%d_%0*d", time.Now().Year(), time.Now().Month(), time.Now().Day(), 3, numberOfRides)
	var EstimatedFare float64

	switch req.RideType {
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
	m = model.Rides{
		RideNumber:    RideNumber,
		PassengerId:   req.PassengerId,
		Status:        "REQUESTED",
		EstimatedFare: EstimatedFare,
		FinalFare:     EstimatedFare,
		Priority:      10,
	}
	m.PickupCoordinate = model.Coordinates{
		EntityId:        req.PassengerId,
		EntityType:      "PASSENGER",
		Address:         req.PickUpAddress,
		Latitude:        req.PickUpLatitude,
		Longitude:       req.PickUpLongitude,
		FareAmount:      m.EstimatedFare,
		DistanceKm:      distance,
		DurationMinutes: 0,
		IsCurrent:       true,
	}

	m.DestinationCoordinate = model.Coordinates{
		EntityId:        req.PassengerId,
		EntityType:      "PASSENGER",
		Address:         req.DestinationAddress,
		Latitude:        req.DestinationLatitude,
		Longitude:       req.DestinationLongitude,
		FareAmount:      m.EstimatedFare,
		DistanceKm:      distance,
		DurationMinutes: 0,
		IsCurrent:       true,
	}
	log.Debug("creating a ride", "passenger-id", req.PassengerId, "estimated-fare", EstimatedFare)
	ctx, cancel = context.WithTimeout(rs.ctx, time.Second*15)
	defer cancel()
	ride_id, err := rs.RidesRepo.CreateRide(ctx, m)
	if err != nil {
		return dto.RidesResponseDto{}, err
	}

	// publish message to rabbitmq
	log.Info("Inserting ride to BM")

	rideMsg := messagebrokerdto.Ride{}

	if err := rs.RidesBroker.PushMessageToDrivers(rs.ctx, rideMsg); err != nil {
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
		EstimatedDurationMinutes: distance / DEFUALT_RATE_PER_MIN,
	}
	return res, nil
}
