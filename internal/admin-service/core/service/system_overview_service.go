package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ride-hail/internal/admin-service/core/domain/dto"
	"ride-hail/internal/admin-service/core/myerrors"
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
	mylog := ds.mylog.Action("GetSystemOverview")
	metrics, err := ds.systemOverviewRepo.GetMetrics(ctx)
	if err != nil {
		if errors.Is(err, myerrors.ErrDBConnClosed) {
			mylog.Error("Failed to connect to connect to db", err)
			return dto.SystemOverview{}, myerrors.ErrDBConnClosedMsg
		}

		return dto.SystemOverview{}, fmt.Errorf("Failed to get metrics: %v", err)
	}
	driverDistribution, err := ds.systemOverviewRepo.GetDriverDistribution(ctx)
	if err != nil {
		if errors.Is(err, myerrors.ErrDBConnClosed) {
			mylog.Error("Failed to connect to connect to db", err)
			return dto.SystemOverview{}, myerrors.ErrDBConnClosedMsg
		}

		return dto.SystemOverview{}, fmt.Errorf("Failed to get driver distribution: %v", err)
	}
	hotspots, err := ds.systemOverviewRepo.GetHotspots(ctx)
	if err != nil {
		if errors.Is(err, myerrors.ErrDBConnClosed) {
			mylog.Error("Failed to connect to connect to db", err)
			return dto.SystemOverview{}, myerrors.ErrDBConnClosedMsg
		}

		return dto.SystemOverview{}, fmt.Errorf("Failed to get hotspots: %v", err)
	}

	systemOverview := dto.SystemOverview{
		Timestamp:          time.Now().Format(time.RFC3339),
		Metrics:            metrics,
		DriverDistribution: driverDistribution,
		Hotspots:           hotspots,
	}

	return systemOverview, nil
}
