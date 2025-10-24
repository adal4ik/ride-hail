package db

import (
	"context"

	"ride-hail/internal/ride-service/core/ports"
)

type PassengerRepo struct {
	db *DB
}

func NewPassengerRepo(db *DB) ports.IPassengerRepo {
	return &PassengerRepo{
		db: db,
	}
}

func (pr *PassengerRepo) Find(ctx context.Context, passengerId string) (string, error) {
	q := `SELECT role FROM users WHERE user_id = $1`

	conn := pr.db.conn

	role := ""
	row := conn.QueryRow(ctx, q, passengerId)
	if err := row.Scan(&role); err != nil {
		return "", err
	}
	return role, nil
}
