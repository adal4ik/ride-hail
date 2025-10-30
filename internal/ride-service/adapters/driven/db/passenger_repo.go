package db

import (
	"context"
	"fmt"

	"ride-hail/internal/ride-service/core/ports"

	"github.com/jackc/pgx/v5"
)

type PassengerRepo struct {
	db *DB
}

func NewPassengerRepo(db *DB) ports.IPassengerRepo {
	return &PassengerRepo{
		db: db,
	}
}

func (pr *PassengerRepo) Exist(ctx context.Context, passengerId string) (string, error) {
	q := `SELECT role FROM users WHERE user_id = $1`

	conn := pr.db.conn

	role := ""
	row := conn.QueryRow(ctx, q, passengerId)
	if err := row.Scan(&role); err != nil {
		return "", err
	}
	return role, nil
}

func (pr *PassengerRepo) CompleteRide(ctx context.Context, rideId string, rating, tips uint) error {
	q := `
		UPDATE 
			drivers
		SET 
			rating = $1, 
			total_earnings = total_earning + $2
		WHERE 
			driver_id = (SELECT 
							driver_id 
						FROM 
							rides 
						WHERE ride_id = $3`

	conn := pr.db.conn
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
defer		tx.Rollback(ctx)

	cmd, err := tx.Exec(ctx, q, rating, tips, rideId)
	if err != nil {
		tx.Rollback(ctx)
		return err
	}

	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("didn't affected to rows")
	}
	return tx.Commit(ctx)
}