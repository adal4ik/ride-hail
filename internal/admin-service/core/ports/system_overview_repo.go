package ports

import (
	"context"

	"ride-hail/internal/admin-service/core/domain/dto"
)

type ISystemOverviewRepo interface {
	GetMetrics(ctx context.Context) (dto.MetricsParams, error)
	GetDriverDistribution(ctx context.Context) (dto.DriverDistributionParams, error)
	GetHotspots(ctx context.Context) ([]dto.HotspotsParams, error)
}
