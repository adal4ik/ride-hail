package services

import (
	"context"
	"fmt"
	"ride-hail/internal/driver-location-service/core/domain/dto"
)

type Distributor struct {
	rideOffers      *chan dto.RideDetails
	driverResponses map[string]chan dto.DriverResponse
	messageDriver   map[string]chan dto.DriverRideOffer
	ctx             context.Context
}

func NewDistributor(ctx context.Context, messageDriver map[string]chan dto.DriverRideOffer, rideOffers *chan dto.RideDetails, driverResponses map[string]chan dto.DriverResponse) *Distributor {
	return &Distributor{
		rideOffers:      rideOffers,
		driverResponses: driverResponses,
		messageDriver:   messageDriver,
		ctx:             ctx,
	}
}
func (d *Distributor) MessageDistributor() error {
	for {
		select {
		case msg := <-*d.rideOffers:
			fmt.Println("Distributing ride offer to driver: ", msg)

		case <-d.ctx.Done():
			// Log shutdown message
			return nil
		}
	}
}
