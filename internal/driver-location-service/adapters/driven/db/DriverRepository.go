package db

import (
	"context"
	"ride-hail/internal/driver-location-service/core/domain/model"
	"time"
)

type DriverRepository struct {
	db *DataBase
}

func NewDriverRepository(db *DataBase) *DriverRepository {
	return &DriverRepository{db: db}
}

func (dr *DriverRepository) GoOnline(ctx context.Context, coord model.DriverCoordinates) (string, error) {
	UpdateQuery := `
		UPDATE 	coordinates coord
		SET latitude = $1, longitude = $2
		FROM drivers
		WHERE coord.coord_id = drivers.coord AND drivers.driver_id = $3
	`
	_, err := dr.db.GetConn().Exec(ctx, UpdateQuery, coord.Latitude, coord.Longitude, coord.Driver_id)
	if err != nil {
		return "", err
	}
	UpdateDriverStatus :=
		`
		UPDATE drivers
		SET status = 'AVAILABLE'
		WHERE driver_id = $1
	`
	_, err = dr.db.GetConn().Exec(ctx, UpdateDriverStatus, coord.Driver_id)
	if err != nil {
		return "", err
	}
	CreateQuery :=
		`
		INSERT INTO driver_sessions(driver_id)
		VALUES ($1)
		RETURNING driver_session_id
	`

	// Update driver status

	var session_id string
	dr.db.GetConn().QueryRow(ctx, CreateQuery, coord.Driver_id).Scan(&session_id)
	return session_id, err
}

func (dr *DriverRepository) GoOffline(ctx context.Context, driver_id string) (model.DriverOfflineResponse, error) {
	var results model.DriverOfflineResponse
	// Getting the summaries
	SelectQuery :=
		`
		SELECT driver_session_id, extract(EPOCH from (NOW() - started_at))/3600.0, total_rides, total_earnings
		FROM driver_sessions
		WHERE driver_id = $1
	`
	err := dr.db.GetConn().QueryRow(ctx, SelectQuery, driver_id).Scan(
		&results.Session_id,
		&results.Session_summary.Duration_hours,
		&results.Session_summary.Rides_completed,
		&results.Session_summary.Earnings,
	)
	if err != nil {
		return model.DriverOfflineResponse{}, err
	}
	// Update Session ended_at time
	UpdateQuery :=
		`
		UPDATE driver_sessions
		SET ended_at = NOW()
		WHERE driver_id = $1
	`
	_, err = dr.db.GetConn().Exec(ctx, UpdateQuery, driver_id)
	if err != nil {
		return model.DriverOfflineResponse{}, err
	}
	// Update Driver Status
	UpdateStatusQuery :=
		`
		UPDATE drivers
		SET status = 'OFFLINE'
		WHERE driver_id = $1
	`
	_, err = dr.db.GetConn().Exec(ctx, UpdateStatusQuery, driver_id)
	return results, err
}

func (dr *DriverRepository) UpdateLocation() {
}

func (dr *DriverRepository) StartRide(ctx context.Context, requestData model.StartRide) (model.StartRideResponse, error) {
	UpdateRideStatusQuery :=
		`
		UPDATE rides
		SET status = 'IN_PROGRESS'
		WHERE ride_id = $1
	`
	_, err := dr.db.GetConn().Exec(ctx, UpdateRideStatusQuery, requestData.Ride_id)
	if err != nil {
		return model.StartRideResponse{}, err
	}
	UpdateDriverStatusQuery :=
		`
		UPDATE drivers
		SET status = 'BUSY'
		WHERE driver_id = $1
	`
	_, err = dr.db.GetConn().Exec(ctx, UpdateDriverStatusQuery, requestData.Driver_location.Driver_id)
	if err != nil {
		return model.StartRideResponse{}, err
	}
	// Created AT ??  WHERE to update it
	var response model.StartRideResponse
	response.Ride_id = requestData.Ride_id
	response.Status = "BUSY"
	response.Started_at = time.Now().String()
	return response, nil
}

func (dr *DriverRepository) CompleteRide() {
}
