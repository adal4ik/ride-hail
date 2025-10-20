package services

import (
	"context"
	"ride-hail/internal/ride-service/core/domain/dto"
	"ride-hail/internal/ride-service/core/domain/model"
	"ride-hail/internal/ride-service/core/ports"
)

type RidesService struct {
	RidesRepo      ports.IRidesRepo
	RidesBroker    ports.IRidesBroker
	RidesWebsocket ports.IRidesWebsocket
	ctx            context.Context
}

func NewRidesService(ctx context.Context,
	RidesRepo ports.IRidesRepo,
	RidesBroker ports.IRidesBroker,
	RidesWebsocket ports.IRidesWebsocket,
) ports.IRidesService {
	return &RidesService{
		ctx: ctx,
		RidesRepo:      RidesRepo,
		RidesBroker:    RidesBroker,
		RidesWebsocket: RidesWebsocket,
	}
}

// implement me
func (rs *RidesService) CreateRide(dto.RidesRequestDto) (dto.RidesResponseDto, error) {
	m := model.Rides{}
	ctx, cancel  := context.WithCancel(rs.ctx)
	defer cancel()
	
	_, err := rs.RidesRepo.CreateRide(ctx, m)
	if err != nil {
		return dto.RidesResponseDto{}, err
	}
	return dto.RidesResponseDto{}, nil
}
