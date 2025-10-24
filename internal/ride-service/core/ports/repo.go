package ports

import (
	"context"
	"ride-hail/internal/ride-service/core/domain/dto"
	"ride-hail/internal/ride-service/core/domain/model"

	"github.com/jackc/pgx/v5"
)

type IDB interface {
	GetConn() *pgx.Conn
	IsAlive() error
	Close() error
}

type IRidesRepo interface {
	CreateRide(context.Context, model.Rides) (string, error)
	GetDistance(context.Context, dto.RidesRequestDto) (float64, error)
	GetNumberRides(context.Context) (int64, error)
}

type IPassengerRepo interface {
	Find(ctx context.Context, passengerId string) (string, error) 
}