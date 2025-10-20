package service

import (
	"context"

	"ride-hail/internal/admin-service/core/domain/dto"
	"ride-hail/internal/admin-service/core/ports"
	"ride-hail/internal/mylogger"
)

type ActiveDrivesService struct {
	ctx              context.Context
	mylog            mylogger.Logger
	activeDrivesRepo ports.IActiveRidesRepo
}

func NewActiveDrivesService(ctx context.Context, mylog mylogger.Logger, activeDrivesRepo ports.IActiveRidesRepo) *ActiveDrivesService {
	return &ActiveDrivesService{
		ctx:              ctx,
		mylog:            mylog,
		activeDrivesRepo: activeDrivesRepo,
	}
}

func (as *ActiveDrivesService) GetActiveRides(ctx context.Context, page, pageSize, offset int) (dto.ActiveDrives, error) {
	return dto.ActiveDrives{}, nil
}
