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
	countQuery := `
    SELECT COUNT(*)
    FROM rides r
    WHERE r.status IN ('EN_ROUTE', 'ARRIVED', 'IN_PROGRESS');
    `

	totalCount := 0
	err := ar.db.GetConn().QueryRow(ctx, countQuery).Scan(&totalCount)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get total count: %v", err)
	}

	// Get paginated active rides with driver locations
	query := `
    SELECT
        r.ride_id,
        r.ride_number,
        r.status,
        r.passenger_id,
        r.driver_id,
        pickup_c.address as pickup_address,
        dest_c.address as destination_address,
        r.started_at,
        -- Calculate estimated completion (you might want to store this in rides table)
        (r.started_at + INTERVAL '30 minutes') as estimated_completion,
        COALESCE(driver_c.latitude, 0.0) as current_lat,
        COALESCE(driver_c.longitude, 0.0) as current_lng,
        -- Calculate distance completed (you might want to store this)
        0.0 as distance_completed_km,
        -- Calculate remaining distance (you might want to store this)
        pickup_c.distance_km as distance_remaining_km
    FROM rides r
    -- Join for pickup address
    JOIN coordinates pickup_c ON r.pickup_coord_id = pickup_c.coord_id
    -- Join for destination address  
    JOIN coordinates dest_c ON r.destination_coord_id = dest_c.coord_id
    -- Left join for driver's current location
    LEFT JOIN coordinates driver_c ON (
        r.driver_id = driver_c.entity_id
        AND driver_c.entity_type = 'DRIVER'
        AND driver_c.is_current = true
    )
    WHERE r.status IN ('EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')
    ORDER BY r.started_at DESC
    LIMIT $1 OFFSET $2;
    `

	offset := (page - 1) * pageSize
	rows, err := ar.db.GetConn().Query(ctx, query, pageSize, offset)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to query active rides: %v", err)
	}
	defer rows.Close()

	var rides []dto.Ride
	for rows.Next() {
		var ride dto.Ride
		err := rows.Scan(
			&ride.RideID,
			&ride.RideNumber,
			&ride.Status,
			&ride.PassengerID,
			&ride.DriverID,
			&ride.PickupAddress,
			&ride.DestinationAddress,
			&ride.StartedAt,
			&ride.EstimatedCompletion,
			&ride.CurrentDriverLocation.Latitude,
			&ride.CurrentDriverLocation.Longitude,
			&ride.DistanceCompletedKm,
			&ride.DistanceRemainingKm,
		)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to scan ride: %v", err)
		}
		rides = append(rides, ride)
	}

	if err := rows.Err(); err != nil {
		return 0, nil, fmt.Errorf("error iterating rows: %v", err)
	}

	return totalCount, rides, nil
}
