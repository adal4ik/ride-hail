package service

import (
	"context"
	"errors"
	"fmt"

	"ride-hail/internal/admin-service/core/domain/dto"
	"ride-hail/internal/admin-service/core/myerrors"
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

func (as *ActiveDrivesService) GetActiveRides(ctx context.Context, page, pageSize int) (dto.ActiveDrives, error) {
	mylog := as.mylog.Action("GetActiveRides")

	totalCount, rides, err := as.activeDrivesRepo.GetActiveRides(ctx, page, pageSize)
	if err != nil {
		if errors.Is(err, myerrors.ErrDBConnClosed) {
			mylog.Error("Failed to connect to connect to db", err)
			return dto.ActiveDrives{}, fmt.Errorf("Internal error, please try again later")
		}

		return dto.ActiveDrives{}, fmt.Errorf("Failed to get active rides: %v", err)
	}

	activeDrives := dto.ActiveDrives{
		Rides:      rides,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
	}

	return activeDrives, nil
}
