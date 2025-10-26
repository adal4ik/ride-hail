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
	CancelRide(context.Context, string, string) (string, error)
	GetDistance(context.Context, dto.RidesRequestDto) (float64, error)
	GetNumberRides(context.Context) (int64, error)
	ChangeStatusMatch(context.Context, string, string) (string, string, error)
	FindDistanceAndPassengerId(ctx context.Context, longitude, latitude float64, rideId string) (distance float64, passengerId string, err error)
}

type IPassengerRepo interface {
	Exist(ctx context.Context, passengerId string) (string, error)
}
