package db

import (
	"context"
	"fmt"
	"ride-hail/internal/driver-location-service/core/domain/model"
	"time"

	"github.com/jackc/pgx/v5"
)

type DriverRepository struct {
	db *DataBase
}

func NewDriverRepository(db *DataBase) *DriverRepository {
	return &DriverRepository{db: db}
}

func (dr *DriverRepository) GoOnline(ctx context.Context, coord model.DriverCoordinates) (string, error) {
	InsertCoordQuery := `
		INSERT INTO coordinates(entity_id, entity_type, address, latitude, longitude)
		VALUES ($1, 'DRIVER', 'Car', $2, $3);
	`
	_, err := dr.db.GetConn().Exec(ctx, InsertCoordQuery, coord.Driver_id, coord.Latitude, coord.Longitude)
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
		VALUES (
			(SELECT coord_id FROM coordinates WHERE entity_id = $1),
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			(SELECT ride_id FROM rides WHERE driver_id = $1 AND status not in ('CANCELLED', 'COMPLETED'))
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

func (dr *DriverRepository) FindDrivers(ctx context.Context, longtitude, latitude float64, vehicleType string) ([]model.DriverInfo, error) {
	Query := `
	SELECT d.driver_id, d.email, d.username, d.vehicle_attrs, d.rating, c.latitude, c.longitude,
       ST_Distance(
         ST_MakePoint(c.longitude, c.latitude)::geography,
         ST_MakePoint($1, $2)::geography
       ) / 1000 as distance_km
	FROM drivers d
	JOIN coordinates c ON c.entity_id = d.driver_id
  		AND c.entity_type = 'DRIVER'
  		AND c.is_current = true
	WHERE d.status = 'AVAILABLE'
 		AND d.vehicle_type = $3
  		AND ST_DWithin(
        	ST_MakePoint(c.longitude, c.latitude)::geography,
        	ST_MakePoint($1, $2)::geography,
        	5000  -- 5km radius
      	)
	ORDER BY distance_km, d.rating DESC
	LIMIT 10;
	`
	rows, err := dr.db.GetConn().Query(ctx, Query, longtitude, latitude, vehicleType)
	if err != nil {
		fmt.Println("Repository Error Arrived ", err)
		return []model.DriverInfo{}, err
	}
	var result []model.DriverInfo
	for rows.Next() {
		var dInfo model.DriverInfo
		err := rows.Scan(&dInfo.DriverId, &dInfo.Email, &dInfo.Name, &dInfo.Vehicle, &dInfo.Rating, &dInfo.Latitude, &dInfo.Longitude, &dInfo.Distance)
		if err != nil {
			fmt.Println("Repository Error Arrived ", err)
			return []model.DriverInfo{}, err
		}
		fmt.Println("Reading rows", dInfo)
		result = append(result, dInfo)
	}
	return result, nil
}

func (dr *DriverRepository) CalculateRideDetails(ctx context.Context, driverLocation model.Location, passagerLocation model.Location) (float64, error) {
	q := `SELECT ST_Distance(ST_MakePoint($1, $2)::geography, ST_MakePoint($3, $4)::geography) / 1000 as distance_km`

	db := dr.db.conn
	row := db.QueryRow(ctx, q, driverLocation.Longitude, driverLocation.Latitude, passagerLocation.Longitude, passagerLocation.Latitude)
	distance := 0.0
	err := row.Scan(&distance)
	if err != nil {
		return 0.0, err
	}
	return distance, nil
}

func (dr *DriverRepository) UpdateDriverStatus(ctx context.Context, driver_id string, status string) error {
	UpdateDriverStatusQuery := `
		UPDATE drivers
		SET status = $1
		WHERE driver_id = $2;
	`
	_, err := dr.db.GetConn().Exec(ctx, UpdateDriverStatusQuery, status, driver_id)
	return err
}

func (dr *DriverRepository) CheckDriverById(ctx context.Context, driver_id string) (bool, error) {
	Query := `
		SELECT EXISTS(SELECT 1 FROM drivers WHERE driver_id = $1);
	`
	var exists bool
	err := dr.db.conn.QueryRow(ctx, Query, driver_id).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (dr *DriverRepository) GetDriverIdByRideId(ctx context.Context, ride_id string) (string, error) {
	Query := `
        SELECT driver_id FROM rides WHERE ride_id = $1;
    `
	var driver_id *string // Use a pointer to string
	err := dr.db.conn.QueryRow(ctx, Query, ride_id).Scan(&driver_id)
	// Check for errors
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", pgx.ErrNoRows
		}
		return "", fmt.Errorf("error querying driver for ride_id %s: %w", ride_id, err)
	}

	// If driver_id is nil, it means the value was NULL in the database
	if driver_id == nil {
		return "", pgx.ErrNoRows
	}

	return *driver_id, nil // Dereference the pointer to return the driver_id string
}
func (dr *DriverRepository) GetRideIdByDriverId(ctx context.Context, driver_id string) (string, error) {
	Query := `
		SELECT ride_id FROM rides WHERE driver_id = $1 AND status NOT IN ('CANCELLED', 'COMPLETED');
	`
	var ride_id string
	err := dr.db.conn.QueryRow(ctx, Query, driver_id).Scan(&ride_id)
	if err != nil {
		return "", err

	}
	return ride_id, nil
}

func (dr *DriverRepository) GetRideDetailsByRideId(ctx context.Context, ride_id string) (model.RideDetails, error) {
	Query := `
		SELECT r.ride_id, u.username, u.user_attrs ,
		       pc.latitude AS pickup_latitude, pc.longitude AS pickup_longitude, pc.address AS pickup_address
		FROM rides r	
		JOIN users u ON r.passenger_id = u.user_id
		JOIN coordinates pc ON r.pickup_coord_id = pc.coord_id
		WHERE r.ride_id = $1;
		`
	var details model.RideDetails
	err := dr.db.conn.QueryRow(ctx, Query, ride_id).Scan(
		&details.Ride_id,
		&details.PassengerName,
		&details.PassengerAttrs,
		&details.PickupLocation.Latitude,
		&details.PickupLocation.Longitude,
		&details.PickupLocation.Address,
	)
	if err != nil {
		fmt.Println(err.Error())
		return model.RideDetails{}, err
	}
	return details, nil
}

/*
SELECT d.driver_id, d.email, d.username, d.vehicle_attrs, d.rating, c.latitude, c.longitude,
       ST_Distance(
         ST_MakePoint(c.longitude, c.latitude)::geography,
         ST_MakePoint(76.88970, 43.238949)::geography
       ) / 1000 as distance_km
	FROM drivers d
	JOIN coordinates c ON c.entity_id = d.driver_id
  		AND c.entity_type = 'DRIVER'
  		AND c.is_current = true
	WHERE d.status = 'AVAILABLE'
 		AND d.vehicle_type = 'ECONOMY'
  		AND ST_DWithin(
        	ST_MakePoint(c.longitude, c.latitude)::geography,
        	ST_MakePoint(76.88970, 43.238949)::geography,
        	5000  -- 5km radius
      	)
	ORDER BY distance_km, d.rating DESC
	LIMIT 10;

*/
