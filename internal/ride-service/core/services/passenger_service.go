package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ride-hail/internal/mylogger"
	"ride-hail/internal/ride-service/core/ports"

	"github.com/jackc/pgx/v5"
)

type PassengerService struct {
	mylog          mylogger.Logger
	PassengerRepo  ports.IPassengerRepo
	RidesWebsocket ports.INotifyWebsocket
	ctx            context.Context
}

func NewPassengerService(ctx context.Context,
	log mylogger.Logger,
	PassengerRepo ports.IPassengerRepo,
	RidesWebsocket ports.INotifyWebsocket,
) ports.IPassengerService {
	return &PassengerService{
		ctx:            ctx,
		mylog:          log,
		PassengerRepo:  PassengerRepo,
		RidesWebsocket: RidesWebsocket,
	}
}

// find to exist this user or not
func (ps *PassengerService) IsPassengerExists(passengerId string) (bool, error) {
	log := ps.mylog.Action("FindPassenger")

	ctx, cancel := context.WithTimeout(ps.ctx, time.Second*5)
	defer cancel()
	roles, err := ps.PassengerRepo.Exist(ctx, passengerId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		log.Error("cannot get user", err)
		return false, err
	}

	if !(roles == "PASSENGER" || roles == "ADMIN") {
		return false, fmt.Errorf("you are not a passenger")
	}
	return true, nil
}