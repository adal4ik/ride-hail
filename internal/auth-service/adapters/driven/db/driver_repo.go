package db

import (
	"context"
	"errors"
	"fmt"

	"ride-hail/internal/auth-service/core/domain/models"
	"ride-hail/internal/auth-service/core/myerrors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DriverRepo struct {
	ctx context.Context
	db  *DB
}

func NewDriverRepo(ctx context.Context, db *DB) *DriverRepo {
	return &DriverRepo{
		ctx: ctx,
		db:  db,
	}
}

func (dr *DriverRepo) Create(ctx context.Context, driver models.Driver) (string, error) {
	// Start a new transaction
	tx, err := dr.db.conn.Begin(ctx)
	if err != nil {
		// Check if the database is alive
		if err2 := dr.db.IsAlive(); err2 != nil {
			return "", err2
		}
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure that we roll back in case of any error
	defer tx.Rollback(ctx)

	// Fixed query to insert driver with correct columns
	q := `INSERT INTO drivers (
		username, email, password, license_number, vehicle_type, vehicle_attrs, user_attrs
	) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING driver_id;`

	id := ""
	var vehicleAttrs interface{}
	if driver.VehicleAttrs != nil {
		vehicleAttrs = *driver.VehicleAttrs
	} else {
		vehicleAttrs = nil
	}
	var userAttrs interface{}
	if driver.UserAttrs != nil {
		userAttrs = *driver.UserAttrs
	} else {
		userAttrs = nil
	}

	row := tx.QueryRow(ctx, q,
		driver.Username,
		driver.Email,
		driver.Password,
		driver.LicenseNumber,
		driver.VehicleType,
		vehicleAttrs,
		userAttrs,
	)
	if err = row.Scan(&id); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" { // unique_violation
				// You can inspect pgErr.ConstraintName if you want to differentiate
				switch pgErr.ConstraintName {
				case "drivers_email_key":
					return "", myerrors.ErrEmailRegistered
				case "drivers_license_number_key":
					return "", myerrors.ErrDriverLicenseNumberRegistered
				default:
					return "", fmt.Errorf("unique constraint violation on %s", pgErr.ConstraintName)
				}
			}
		}
		return "", fmt.Errorf("failed to insert driver: %w", err)
	}

	// Commit the transaction
	if err = tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return id, nil
}

func (dr *DriverRepo) GetByEmail(ctx context.Context, email string) (models.Driver, error) {
	q := `
		SELECT 
			driver_id,
			created_at,
			updated_at,
			username,
			email,
			password,
			license_number,
			vehicle_type,
			vehicle_attrs,
			rating,
			total_rides,
			total_earnings,
			status,
			is_verified,
			user_attrs
		FROM 
			drivers
		WHERE
			email = $1
	`

	var d models.Driver
	err := dr.db.conn.QueryRow(ctx, q, email).Scan(
		&d.DriverId,
		&d.CreatedAt,
		&d.UpdatedAt,
		&d.Username,
		&d.Email,
		&d.Password,
		&d.LicenseNumber,
		&d.VehicleType,
		&d.VehicleAttrs,
		&d.Rating,
		&d.TotalRides,
		&d.TotalEarnings,
		&d.Status,
		&d.IsVerified,
		&d.UserAttrs,
	)
	if err != nil {
		// Check if the database is alive
		if err2 := dr.db.IsAlive(); err2 != nil {
			return models.Driver{}, err2
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Driver{}, myerrors.ErrUnknownEmail
		}
		return models.Driver{}, fmt.Errorf("failed to get driver by email: %w", err)
	}

	return d, nil
}
