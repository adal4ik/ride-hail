package db

import (
	"context"
	"time"

	"ride-hail/internal/driver-location-service/core/domain/model"
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
		WHERE coord.entity_id = drivers.driver_id AND drivers.driver_id = $3;
	`
	_, err := dr.db.GetConn().Exec(ctx, UpdateQuery, coord.Latitude, coord.Longitude, coord.Driver_id)
	if err != nil {
		return "", err
	}
	UpdateDriverStatus := `
		UPDATE drivers
		SET status = 'AVAILABLE'
		WHERE driver_id = $1;
	`
	_, err = dr.db.GetConn().Exec(ctx, UpdateDriverStatus, coord.Driver_id)
	if err != nil {
		return "", err
	}
	CreateQuery := `
		INSERT INTO driver_sessions(driver_id)
		VALUES ($1)
		RETURNING driver_session_id;
	`

	var session_id string
	dr.db.GetConn().QueryRow(ctx, CreateQuery, coord.Driver_id).Scan(&session_id)
	return session_id, err
}

func (dr *DriverRepository) GoOffline(ctx context.Context, driver_id string) (model.DriverOfflineResponse, error) {
	var results model.DriverOfflineResponse
	// Getting the summaries
	SelectQuery := `
		SELECT driver_session_id, extract(EPOCH from (NOW() - started_at))/3600.0, total_rides, total_earnings
		FROM driver_sessions
		WHERE driver_id = $1;
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
	UpdateQuery := `
		UPDATE driver_sessions
		SET ended_at = NOW()
		WHERE driver_id = $1;
	`
	_, err = dr.db.GetConn().Exec(ctx, UpdateQuery, driver_id)
	if err != nil {
		return model.DriverOfflineResponse{}, err
	}
	// Update Driver Status
	UpdateStatusQuery := `
		UPDATE drivers
		SET status = 'OFFLINE'
		WHERE driver_id = $1;
	`
	_, err = dr.db.GetConn().Exec(ctx, UpdateStatusQuery, driver_id)
	return results, err
}

func (dr *DriverRepository) UpdateLocation(ctx context.Context, driver_id string, newLocation model.NewLocation) (model.NewLocationResponse, error) {
	NewLocationQuery := `
		INSERT INTO location_history(coord_id, driver_id, latitude, longitude, accuracy_meters, speed_kmh, heading_degrees, ride_id)
		VALUE (
			(SELECT coord_id FROM coordinates WHERE entity_id = $1),
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			(SELECT ride_id FROM rides WHERE driver_id = $1 AND status not in ('CANCELLED', 'COMPLETED'));
		)
	`
	_, err := dr.db.GetConn().Exec(ctx, NewLocationQuery, driver_id, newLocation.Latitude, newLocation.Longitude, newLocation.Accuracy_meters, newLocation.Speed_kmh, newLocation.Heading_Degrees)
	if err != nil {
		return model.NewLocationResponse{}, err
	}
	var response model.NewLocationResponse
	CoordinatesQuery := `
		UPDATE coordinates
		SET latitude = $1, 
			longitude = $2,
			updated_at = NOW()
		WHERE entity_id = $3
		RETURNING coord_id, updated_at;
	`
	err = dr.db.GetConn().QueryRow(ctx, CoordinatesQuery, newLocation.Latitude, newLocation.Longitude, driver_id).Scan(&response.Coordinate_id, &response.Updated_at)
	if err != nil {
		return model.NewLocationResponse{}, err
	}
	return response, nil
}

func (dr *DriverRepository) StartRide(ctx context.Context, requestData model.StartRide) (model.StartRideResponse, error) {
	UpdateRideStatusQuery := `
		UPDATE rides
		SET status = 'IN_PROGRESS'
		WHERE ride_id = $1;
	`
	_, err := dr.db.GetConn().Exec(ctx, UpdateRideStatusQuery, requestData.Ride_id)
	if err != nil {
		return model.StartRideResponse{}, err
	}
	UpdateDriverStatusQuery := `
		UPDATE drivers
		SET status = 'BUSY'
		WHERE driver_id = $1;
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

func (dr *DriverRepository) CompleteRide(ctx context.Context, requestData model.RideCompleteForm) (model.RideCompleteResponse, error) {
	var response model.RideCompleteResponse
	response.Status = "AVAILABLE"
	response.Ride_id = requestData.Ride_id
	response.Message = "Ride completed successfully"

	RidesQuery := `
		UPDATE rides
		SET status = 'COMPLETED',
		WHERE ride_id = $1;
	`
	_, err := dr.db.GetConn().Exec(ctx, RidesQuery, requestData.Ride_id)
	if err != nil {
		return model.RideCompleteResponse{}, err
	}

	CoordinatesQuery := `
		UPDATE coordinates
		SET distance_km = $1,
			duration_minutes = $2,
			latitude = $3,
			longitude = $4
		FROM rides
		WHERE coordinates.coord_id = rides.destination_coord_id && rides.ride_id = $5;
	`

	_, err = dr.db.GetConn().Exec(ctx, CoordinatesQuery, requestData.ActualDistancekm, requestData.ActualDurationm, requestData.FinalLocation.Latitude, requestData.FinalLocation.Longitude, requestData.Ride_id)
	if err != nil {
		return model.RideCompleteResponse{}, err
	}

	UpdateDriverStatusQuery := `
		UPDATE drivers
		SET status = 'AVAILABLE'
		FROM rides
		WHERE drivers.driver_id = rides.driver_id && rides.ride_id = $1;
	`
	_, err = dr.db.GetConn().Exec(ctx, UpdateDriverStatusQuery, requestData.Ride_id)
	if err != nil {
		return model.RideCompleteResponse{}, err
	}

	DriverEarningsQuery := `
		SELECT final_fare FROM rides WHERE ride_id = $1;
	`

	err = dr.db.GetConn().QueryRow(ctx, DriverEarningsQuery, requestData.Ride_id).Scan(&response.DriverEarning)
	if err != nil {
		return model.RideCompleteResponse{}, err
	}

	response.CompletedAt = time.Now().String()
	return response, nil
}
