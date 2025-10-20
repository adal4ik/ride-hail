package services

import (
	"context"
	"fmt"
	"ride-hail/internal/mylogger"
	"ride-hail/internal/ride-service/core/domain/dto"
	messagebrokerdto "ride-hail/internal/ride-service/core/domain/message_broker_dto"
	"ride-hail/internal/ride-service/core/domain/model"
	"ride-hail/internal/ride-service/core/ports"
)

type RidesService struct {
	RidesRepo      ports.IRidesRepo
	RidesBroker    ports.IRidesBroker
	RidesWebsocket ports.IRidesWebsocket
	ctx            context.Context
	mylog          mylogger.Logger
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

// implement me
func (rs *RidesService) CreateRide(ride dto.RidesRequestDto) (dto.RidesResponseDto, error) {
	mylog := rs.mylog.Action("create ride")

	mylog.Info("Inserting ride to DB")

	m := model.Rides{}
	ctx, cancel := context.WithCancel(rs.ctx)
	defer cancel()

	_, err := rs.RidesRepo.CreateRide(ctx, m)
	if err != nil {
		return dto.RidesResponseDto{}, err
	}

	// publish message to rabbitmq
	mylog.Info("Inserting ride to BM")

	rideMsg := messagebrokerdto.Ride{}

	if err := rs.RidesBroker.PushMessage(rs.ctx, rideMsg); err != nil {
		mylog.Error("Failed to publish message", err)
		return dto.RidesResponseDto{}, fmt.Errorf("cannot send message to broker: %w", err)
	}

	return dto.RidesResponseDto{}, nil
}
