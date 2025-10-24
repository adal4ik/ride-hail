package db

import (
	"context"
	"fmt"

	"ride-hail/internal/admin-service/core/domain/dto"
	"ride-hail/internal/admin-service/core/ports"
)

type ActiveDrivesRepo struct {
	db ports.IDB
}

func NewActiveDrivesRepo(db ports.IDB) *ActiveDrivesRepo {
	return &ActiveDrivesRepo{db: db}
}

func (ar *ActiveDrivesRepo) GetActiveRides(ctx context.Context, page, pageSize int) (int, []dto.Ride, error) {
	// Count active rides
	q1 := `
    SELECT COUNT(*)
    FROM rides r
    WHERE r.status IN ('EN_ROUTE', 'ARRIVED', 'IN_PROGRESS');
    `

	// Get paginated active rides with driver locations
	// q2 := `
	// SELECT
	//     r.ride_id,
	//     r.ride_number,
	//     r.status,
	//     r.passenger_id,
	//     r.driver_id,
	//     r.pickup_address,
	//     r.destination_address,
	//     r.started_at,
	//     r.estimated_completion,
	//     COALESCE(c.latitude, 0.0) as current_lat,
	//     COALESCE(c.longitude, 0.0) as current_lng,
	//     r.distance_completed_km,
	//     r.distance_remaining_km
	// FROM rides r
	// LEFT JOIN coordinates c ON (
	//     r.driver_id = c.entity_id
	//     AND c.entity_type = 'DRIVER'
	//     AND c.is_current = true
	// )
	// WHERE r.status IN ('EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')
	// ORDER BY r.started_at DESC
	// LIMIT $1 OFFSET $2;
	// `

	totalCount := 0
	err := ar.db.GetConn().QueryRow(ctx, q1).Scan(&totalCount)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get total count: %v", err)
	}

	var rides []dto.Ride
	// offset := (page - 1) * pageSize

	return totalCount, rides, nil
}
