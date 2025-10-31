package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"ride-hail/internal/ride-service/core/domain/dto"
	"ride-hail/internal/ride-service/core/domain/model"
	"ride-hail/internal/ride-service/core/ports"

	messagebrokerdto "ride-hail/internal/ride-service/core/domain/message_broker_dto"

	websocketdto "ride-hail/internal/ride-service/core/domain/websocket_dto"

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
		// Check if the database is alive
		if err2 := rr.db.IsAlive(); err2 != nil {
			return 0, err2
		}
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
		// Check if the database is alive
		if err2 := rr.db.IsAlive(); err2 != nil {
			return 0, err2
		}
		return 0.0, err
	}
	return count, nil
}

func (rr *RidesRepo) CheckDuplicate(ctx context.Context, passengerId string) (int, error) {
	q := `SELECT COUNT(8) FROM rides WHERE passenger_id = $1 AND status IN ('REQUESTED', 'MATCHED', 'EN_ROUTE', 'ARRIVED');`
	db := rr.db.conn

	row := db.QueryRow(ctx, q, passengerId)
	var count int = 0
	err := row.Scan(&count)
	if err != nil {
		// Check if the database is alive
		if err2 := rr.db.IsAlive(); err2 != nil {
			return 0, err2
		}
		return 0, err
	}

	return count, nil
}

func (rr *RidesRepo) CreateRide(ctx context.Context, m model.Rides) (string, error) {
	conn := rr.db.conn
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		// Check if the database is alive
		if err2 := rr.db.IsAlive(); err2 != nil {
			return "", err2
		}
		return "", err
	}
	defer tx.Rollback(ctx) // Safe rollback if not committed

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
		return "", err
	}

	return RideId, tx.Commit(ctx)
}

func (rr *RidesRepo) ChangeStatusMatch(ctx context.Context, rideID, driverID string) (string, string, error) {
	conn := rr.db.conn
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		// Check if the database is alive
		if err2 := rr.db.IsAlive(); err2 != nil {
			return "", "", err2
		}

		return "", "", err
	}
	defer tx.Rollback(ctx) // Safe rollback if not committed

	q := `UPDATE 
			rides 
		SET 
			driver_id = $1,
			status = 'MATCHED'
		WHERE ride_id = $2`
	_, err = tx.Exec(ctx, q, driverID, rideID)
	if err != nil {
		return "", "", err
	}

	var (
		passengerId string = ""
		rideNumber  string = ""
	)

	q = `SELECT passenger_id, ride_number FROM rides WHERE ride_id = $1`
	row := tx.QueryRow(ctx, q, rideID)

	err = row.Scan(&passengerId, &rideNumber)
	if err != nil {
		return "", "", err
	}
	return passengerId, rideNumber, tx.Commit(ctx)
}

func (rr *RidesRepo) FindDistanceAndPassengerId(ctx context.Context, longitude, latitude float64, rideId string) (float64, string, error) {
	q := `SELECT
			ST_Distance(ST_MakePoint(c.longitude, c.latitude)::geography, ST_MakePoint($1, $2)::geography),
			r.passenger_id
		FROM rides r 
		JOIN coordinates c ON r.pickup_coord_id = c.coord_id 
		WHERE r.ride_id = $3`

	conn := rr.db.conn

	row := conn.QueryRow(ctx, q, longitude, latitude, rideId)
	var (
		distance    float64
		passengerId string
	)
	if err := row.Scan(&distance, &passengerId); err != nil {
		// Check if the database is alive
		if err2 := rr.db.IsAlive(); err2 != nil {
			return 0, "", err2
		}
		return 0.0, "", err
	}

	return distance, passengerId, nil
}

func (rr *RidesRepo) CancelRide(ctx context.Context, rideId, reason string) (model.Rides, error) {
	q1 := `
    SELECT  
        driver_id, 
        status,
		final_fare
    FROM 
        rides
    WHERE 
        ride_id = $1`

	q2 := `
    UPDATE rides
    SET 
        status = 'CANCELLED', 
        cancelled_at = NOW(),
        cancellation_reason = $2
    WHERE ride_id = $1 AND status != 'COMPLETED'`

	// Use sql.NullString or pointers to handle NULL values

	// Start transaction first to maintain consistency
	tx, err := rr.db.conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return model.Rides{}, err
	}
	defer tx.Rollback(ctx) // Safe rollback if not committed

	var ride model.Rides
	// Perform SELECT within the same transaction
	row := tx.QueryRow(ctx, q1, rideId)
	if err := row.Scan(&ride.DriverId, &ride.Status, &ride.FinalFare); err != nil {
		return model.Rides{}, fmt.Errorf("failed to fetch ride details: %w", err)
	}

	// Validate business rules
	if ride.Status == "CANCELLED" {
		return model.Rides{}, fmt.Errorf("ride already cancelled")
	}

	// Perform the update
	if _, err := tx.Exec(ctx, q2, rideId, reason); err != nil {
		return model.Rides{}, fmt.Errorf("failed to cancel ride: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return model.Rides{}, fmt.Errorf("failed to commit: %w", err)
	}

	return ride, nil
}

// ChangeStatus will return passenger id, ride number and driver information
func (rr *RidesRepo) ChangeStatus(ctx context.Context, msg messagebrokerdto.DriverStatusUpdate) (string, string, float64, websocketdto.DriverInfo, error) {
	conn := rr.db.conn

	// Start transaction first to maintain consistency
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		if err2 := rr.db.IsAlive(); err2 != nil {
			return "", "", 0, websocketdto.DriverInfo{}, err2
		}
		return "", "", 0, websocketdto.DriverInfo{}, err
	}
	defer tx.Rollback(ctx) // Safe rollback if not committed

	q1 := `
    SELECT  
        r.passenger_id, 
        r.ride_number,
		d.username,
		d.rating,
		d.vehicle_attrs,
		r.final_fare
    FROM 
        rides r
	JOIN drivers d 
	ON d.driver_id = r.driver_id 
    WHERE 
        ride_id = $1`

	q2 := `
	UPDATE rides
    SET 
        status = $2
    WHERE ride_id = $1`
	var (
		driverInfo  websocketdto.DriverInfo
		jsonData    []byte
		passengerId sql.NullString
		rideNumber  sql.NullString
		finalFare   sql.NullFloat64
	)
	driverInfo.DriverID = msg.DriverId

	// first get the passenger id
	row := tx.QueryRow(ctx, q1, msg.RideId)
	if err := row.Scan(
		&passengerId,
		&rideNumber,
		&driverInfo.Name,
		&driverInfo.Rating,
		&jsonData,
		&finalFare,
	); err != nil {
		return "", "", 0, websocketdto.DriverInfo{}, fmt.Errorf("failed to fetch ride details: %w", err)
	}

	if err := json.Unmarshal(jsonData, &driverInfo.Vehicle); err != nil {
		return "", "", 0, websocketdto.DriverInfo{}, fmt.Errorf("failed to unmarshal vehile details: %w", err)
	}

	// Check for values
	if !passengerId.Valid {
		return "", "", 0, websocketdto.DriverInfo{}, fmt.Errorf("driver id not found")
	}

	if !rideNumber.Valid {
		return "", "", 0, websocketdto.DriverInfo{}, fmt.Errorf("ride number not found")
	}

	if !finalFare.Valid {
		return "", "", 0, websocketdto.DriverInfo{}, fmt.Errorf("ride number not found")
	}

	// Perform the update
	if _, err := tx.Exec(ctx, q2, msg.RideId, msg.Status); err != nil {
		// Check if the database is alive
		if err2 := rr.db.IsAlive(); err2 != nil {
			return "", "", 0, websocketdto.DriverInfo{}, err2
		}
		return "", "", 0, websocketdto.DriverInfo{}, fmt.Errorf("failed to update status: %w", err)
	}

	return passengerId.String, rideNumber.String, finalFare.Float64, driverInfo, tx.Commit(ctx)
}

func (rr *RidesRepo) CancelEveryPossibleRides(ctx context.Context) error {
	conn := rr.db.conn

	// Start transaction first to maintain consistency
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		if err2 := rr.db.IsAlive(); err2 != nil {
			return err2
		}
		return err
	}
	defer tx.Rollback(ctx) // Safe rollback if not committed

	q := `UPDATE rides SET status = 'CANCELLED' WHERE status IN ('REQUESTED', 'MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')`

	_, err = tx.Exec(ctx, q)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (pr *RidesRepo) GetCancelPossibleRides(ctx context.Context) ([]model.Rides, error) {
	q := `SELECT ride_id, status, final_fare FROM rides WHERE status IN ('REQUESTED', 'MATCHED', 'IN_PROGRESS', 'EN_ROUTE', 'ARRIVED');`
	cn := pr.db.conn

	rows, err := cn.Query(ctx, q)
	if err != nil {
		// Check if the database is alive
		if err2 := pr.db.IsAlive(); err2 != nil {
			return nil, err2
		}
		return []model.Rides{}, err
	}

	var rides []model.Rides
	for rows.Next() {
		var ride model.Rides
		if err := rows.Scan(&ride.ID, &ride.Status, &ride.FinalFare); err != nil {
			continue
		}
		rides = append(rides, ride)
	}

	return rides, nil
}
