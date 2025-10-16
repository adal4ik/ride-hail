package db

import (
	"context"
	"fmt"

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

	// Query 1: Main metrics
	q1 := `
    SELECT 
		COUNT(*) FILTER (WHERE status IN ('EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')) as active_rides,
		COUNT(*) FILTER (WHERE created_at::date = current_date) as total_rides_today,
		COALESCE(SUM(final_fare) FILTER (WHERE status = 'COMPLETED' AND created_at::date = current_date), 0)::float as total_revenue_today,
		COALESCE(AVG(EXTRACT(EPOCH FROM (matched_at - requested_at)) / 60) 
				FILTER (WHERE status = 'COMPLETED' AND matched_at IS NOT NULL), 0)::float as avg_wait_time,
		COALESCE(AVG(EXTRACT(EPOCH FROM (completed_at - started_at)) / 60)
				FILTER (WHERE status = 'COMPLETED' AND completed_at IS NOT NULL), 0)::float as avg_ride_duration,
		CASE 
			WHEN COUNT(*) FILTER (WHERE created_at::date = current_date) > 0 THEN
				COUNT(*) FILTER (WHERE status = 'CANCELLED' AND created_at::date = current_date)::float / 
				COUNT(*) FILTER (WHERE created_at::date = current_date)::float
			ELSE 0 
		END::float as cancellation_rate
	FROM rides;
    `

	// Query 2: Driver metrics
	q2 := `
    SELECT 
        COUNT(*) FILTER (WHERE status = 'AVAILABLE') as available_drivers,
        COUNT(*) FILTER (WHERE status = 'BUSY') as busy_drivers
    FROM drivers;
    `

	// Execute queries
	err := sr.db.GetConn().QueryRow(ctx, q1).Scan(
		&metrics.ActiveRides,
		&metrics.TotalRidesToday,
		&metrics.TotalRevenueToday,
		&metrics.AverageWaitTimeMinutes,
		&metrics.AverageRideDurationMinutes,
		&metrics.CancellationRate,
	)
	if err != nil {
		return dto.MetricsParams{}, fmt.Errorf("failed to get ride metrics: %v", err)
	}

	err = sr.db.GetConn().QueryRow(ctx, q2).Scan(
		&metrics.AvailableDrivers,
		&metrics.BusyDrivers,
	)
	if err != nil {
		return dto.MetricsParams{}, fmt.Errorf("failed to get driver metrics: %v", err)
	}

	return metrics, nil
}

func (sr *SystemOverviewRepo) GetDriverDistribution(ctx context.Context) (dto.DriverDistributionParams, error) {
	driverDistribution := dto.DriverDistributionParams{}
	q := `
    SELECT 
        COUNT(*) FILTER (WHERE vehicle_type = 'ECONOMY') as economy_drivers,
        COUNT(*) FILTER (WHERE vehicle_type = 'PREMIUM') as premium_drivers,
        COUNT(*) FILTER (WHERE vehicle_type = 'XL') as xl_drivers
    FROM drivers;
    `

	// Execute queries
	err := sr.db.GetConn().QueryRow(ctx, q).Scan(
		&driverDistribution.Economy,
		&driverDistribution.Premium,
		&driverDistribution.XL,
	)
	if err != nil {
		return dto.DriverDistributionParams{}, fmt.Errorf("failed to get driver distribution: %v", err)
	}

	return driverDistribution, nil
}

func (sr *SystemOverviewRepo) GetHotspots(ctx context.Context) ([]dto.HotspotsParams, error) {
	return nil, nil
}
