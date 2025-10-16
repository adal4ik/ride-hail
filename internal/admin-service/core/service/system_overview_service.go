package service

import (
	"context"
	"fmt"
	"time"

	"ride-hail/internal/admin-service/core/domain/dto"
	"ride-hail/internal/admin-service/core/ports"
	"ride-hail/internal/mylogger"
)

type SystemOverviewService struct {
	ctx                context.Context
	mylog              mylogger.Logger
	systemOverviewRepo ports.ISystemOverviewRepo
}

func NewSystemOverviewService(ctx context.Context, mylog mylogger.Logger, systemOverviewRepo ports.ISystemOverviewRepo) *SystemOverviewService {
	return &SystemOverviewService{
		ctx:                ctx,
		mylog:              mylog,
		systemOverviewRepo: systemOverviewRepo,
	}
}

func (ds *SystemOverviewService) GetSystemOverview(ctx context.Context) (dto.SystemOverview, error) {
	metrics, err := ds.systemOverviewRepo.GetMetrics(ctx)
	if err != nil {
		return dto.SystemOverview{}, fmt.Errorf("Failed to get metrics")
	}
	driverContribution, err := ds.systemOverviewRepo.GetDriverContribution(ctx)
	if err != nil {
		return dto.SystemOverview{}, fmt.Errorf("Failed to get driver contribution")
	}
	hotspots, err := ds.systemOverviewRepo.GetHotspots(ctx)
	if err != nil {
		return dto.SystemOverview{}, fmt.Errorf("Failed to get hotspots")
	}

	systemOverview := dto.SystemOverview{
		Timestamp:          time.Now().Format(time.RFC3339),
		Metrics:            metrics,
		DriverContribution: driverContribution,
		Hotspots:           hotspots,
	}

	return systemOverview, nil
}

func (ds *SystemOverviewService) GetActiveRides(ctx context.Context) (dto.SystemOverview, error) {
	return dto.SystemOverview{}, nil
}
