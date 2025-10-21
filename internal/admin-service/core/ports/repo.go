package ports

import (
	"context"

	"ride-hail/internal/admin-service/core/domain/dto"

	"github.com/jackc/pgx/v5"
)

type IDB interface {
	GetConn() *pgx.Conn
	IsAlive() error
	Close() error
}

type ISystemOverviewRepo interface {
	GetMetrics(ctx context.Context) (dto.MetricsParams, error)
	GetDriverDistribution(ctx context.Context) (dto.DriverDistributionParams, error)
	GetHotspots(ctx context.Context) ([]dto.HotspotsParams, error)
}

type IActiveRidesRepo interface {
	GetActiveRides(ctx context.Context, page, pageSize int) (int, []dto.Ride, error)
}
