package ports

import (
	"context"
	"ride-hail/internal/ride-service/core/domain/model"

	"github.com/jackc/pgx/v5"
)

type IDB interface {
	GetConn() *pgx.Conn
	IsAlive() error
	Close() error
}

type IRidesRepo interface {
	CreateRide(context.Context, model.Rides) (error)
}
