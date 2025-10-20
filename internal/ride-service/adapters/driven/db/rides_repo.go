package db

import (
	"context"
	"ride-hail/internal/ride-service/core/domain/model"
	"ride-hail/internal/ride-service/core/ports"

	"github.com/jackc/pgx/v5"
)

type RidesRepo struct {
	db ports.IDB
}

func NewRidesRepo(db ports.IDB) ports.IRidesRepo {
	return &RidesRepo{
		db: db,
	}
}

func (rr RidesRepo) CreateRide(ctx context.Context, m model.Rides) (string, error) {
	conn := rr.db.GetConn()
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "",err
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
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`

	row := tx.QueryRow(ctx, q1,
		m.PassengerId,
		"passenger",
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
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`

	row = tx.QueryRow(ctx, q2,
		m.PassengerId,
		"passenger",
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
		pickup_coordinate_id, 
		destination_coordinate_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	
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
