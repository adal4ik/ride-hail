package db

import (
	"context"

	"ride-hail/internal/admin-service/core/domain/dto"
	"ride-hail/internal/admin-service/core/ports"
)

type SystemOverviewRepo struct {
	db ports.IDB
}

func NewSystemOverviewRepo(db ports.IDB) *SystemOverviewRepo {
	return &SystemOverviewRepo{db: db}
}

func (sr *SystemOverviewRepo) GetMetrics(ctx context.Context) (dto.MetricsParams, error) {
	var metrics dto.MetricsParams

	// 1. Collect Active Rides
	q1 := `
	SELECT
		timestamp
	FROM 
		rides
	WHERE
		order_number = $1
	`
	if err := sr.db.GetConn().QueryRow(ctx, q1).Scan(metrics); err != nil {
	}

	// 2. Collect Available Drivers

	// 3. Collect Busy Drivers

	// 4. Collect Total Rides Today

	// 5. Collect Total Revenue Today

	// 6. Collect Average Wait Time Minutes

	// 7. Collect Average Ride Duration Minutes

	// 8. Collect Cancellation Rate

	return metrics, nil
}

func (sr *SystemOverviewRepo) GetDriverContribution(ctx context.Context) (dto.DriverContributionParams, error) {
	return dto.DriverContributionParams{}, nil
}

func (sr *SystemOverviewRepo) GetHotspots(ctx context.Context) ([]dto.HotspotsParams, error) {
	return nil, nil
}
