package db

import (
	"context"

	"ride-hail/internal/ride-service/core/domain/dto"
	"ride-hail/internal/ride-service/core/domain/model"
	"ride-hail/internal/ride-service/core/ports"

	"github.com/jackc/pgx/v5"
)

type RidesRepo struct {
	db *DB
}

func NewRidesRepo(db *DB) ports.IRidesRepo {
	return &RidesRepo{
		db: db,
	}
}

func (rr *RidesRepo) GetDistance(ctx context.Context, req dto.RidesRequestDto) (float64, error) {
	q := `SELECT ST_Distance(ST_MakePoint($1, $2)::geography, ST_MakePoint($3, $4)::geography) / 1000 as distance_km`

	db := rr.db.conn
	row := db.QueryRow(ctx, q, req.PickUpLongitude, req.PickUpLatitude, req.DestinationLongitude, req.DestinationLatitude)
	distance := 0.0
	err := row.Scan(&distance)
	if err != nil {
		return 0.0, err
	}
	return distance, nil
}

func (rr *RidesRepo) GetNumberRides(ctx context.Context) (int64, error) {
	q := `
	SELECT 
		COUNT(*) 
	FROM 
		rides
	WHERE
		created_at::date = current_date
	`
	db := rr.db.conn
	row := db.QueryRow(ctx, q)
	var count int64 = 0
	err := row.Scan(&count)
	if err != nil {
		return 0.0, err
	}
	return count, nil
}

func (rr *RidesRepo) CreateRide(ctx context.Context, m model.Rides) (string, error) {
	conn := rr.db.conn
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", err
	}
	// pick up coordinates
	q1 := `INSERT INTO coordinates(
			entity_id, 
			entity_type,
			address, 
			latitude, 
			longitude, 
			fare_amount, 
			distance_km, 
			duration_minutes, 
			is_current
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING coord_id`

	row := tx.QueryRow(ctx, q1,
		m.PassengerId,
		m.PickupCoordinate.EntityType,
		m.PickupCoordinate.Address,
		m.PickupCoordinate.Latitude,
		m.PickupCoordinate.Longitude,
		m.PickupCoordinate.FareAmount,
		m.PickupCoordinate.DistanceKm,
		m.PickupCoordinate.DurationMinutes,
		m.PickupCoordinate.IsCurrent,
	)
	PickupCoordinateId := ""
	if err := row.Scan(&PickupCoordinateId); err != nil {
		tx.Rollback(ctx)
		return "", err
	}
	// destination coordinates
	q2 := `INSERT INTO coordinates(
			entity_id, 
			entity_type,
			address, 
			latitude, 
			longitude, 
			fare_amount, 
			distance_km, 
			duration_minutes, 
			is_current
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING coord_id`

	row = tx.QueryRow(ctx, q2,
		m.PassengerId,
		m.DestinationCoordinate.EntityType,
		m.DestinationCoordinate.Address,
		m.DestinationCoordinate.Latitude,
		m.DestinationCoordinate.Longitude,
		m.DestinationCoordinate.FareAmount,
		m.DestinationCoordinate.DistanceKm,
		m.DestinationCoordinate.DurationMinutes,
		m.DestinationCoordinate.IsCurrent,
	)
	DestinationCoordinateId := ""
	if err := row.Scan(&DestinationCoordinateId); err != nil {
		tx.Rollback(ctx)
		return "", err
	}
	// rides
	q3 := `INSERT INTO rides(
		ride_number,
		passenger_id,
		status,
		priority, 
		estimated_fare,
		final_fare, 
		pickup_coord_id, 
		destination_coord_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING ride_id`

	row = tx.QueryRow(ctx, q3,
		m.RideNumber,
		m.PassengerId,
		m.Status,
		m.Priority,
		m.EstimatedFare,
		m.FinalFare,
		PickupCoordinateId,
		DestinationCoordinateId,
	)

	RideId := ""
	if err := row.Scan(&RideId); err != nil {
		tx.Rollback(ctx)
		return "", err
	}

	return RideId, tx.Commit(ctx)
}
