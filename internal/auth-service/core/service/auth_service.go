package service

import (
	"context"
	"ride-hail/internal/admin-service/core/domain/dto"
	"ride-hail/internal/admin-service/core/ports"
	"ride-hail/internal/mylogger"
)

type AuthService struct {
	ctx              context.Context
	mylog            mylogger.Logger
	activeDrivesRepo ports.IActiveRidesRepo
}

func NewAuthService(ctx context.Context, mylog mylogger.Logger, activeDrivesRepo ports.IActiveRidesRepo) *AuthService {
	return &AuthService{
		ctx:              ctx,
		mylog:            mylog,
		activeDrivesRepo: activeDrivesRepo,
	}
}

func (as *AuthService) GetActiveRides(ctx context.Context, page, pageSize, offset int) (dto.ActiveDrives, error) {
	return dto.ActiveDrives{}, nil
}
