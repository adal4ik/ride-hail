package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ride-hail/internal/driver-location-service/core/domain/dto"
	driven "ride-hail/internal/driver-location-service/core/ports/driven"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ADD LOGGER
type Distributor struct {
	rideOffers      <-chan amqp.Delivery
	rideStatuses    <-chan amqp.Delivery
	driverResponses map[string]chan dto.DriverResponse
	messageDriver   map[string]chan dto.DriverRideOffer
	ctx             context.Context
	broker          driven.IDriverBroker
	driverService   *DriverService
}

func NewDistributor(
	ctx context.Context,
	messageDriver map[string]chan dto.DriverRideOffer,
	rideOffers <-chan amqp.Delivery,
	rideStatuses <-chan amqp.Delivery,
	driverResponses map[string]chan dto.DriverResponse,
	broker driven.IDriverBroker,
	driverService *DriverService,
) *Distributor {
	return &Distributor{
		rideOffers:      rideOffers,
		rideStatuses:    rideStatuses,
		driverResponses: driverResponses,
		messageDriver:   messageDriver,
		ctx:             ctx,
		broker:          broker,
		driverService:   driverService,
	}
}

func (d *Distributor) MessageDistributor() error {
	for {
		select {
		case requestDelivery := <-d.rideOffers:
			// If No Drivers
			if len(d.messageDriver) == 0 {
				requestDelivery.Nack(false, true)
				time.Sleep(7 * time.Second)
				continue
			}
			// Logic for getting write driver
			var req dto.RideDetails
			if err := json.Unmarshal(requestDelivery.Body, &req); err != nil {
				fmt.Println(err.Error())
				continue
			}
			ctx := context.Background()
			allDrivers, err := d.driverService.FindAppropriateDrivers(ctx, req.Pickup_location.Lng, req.Destination_location.Lat, req.Ride_type)
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			// Got filtered Drivers
			var filteredDrivers []dto.DriverInfo
			for _, driver := range allDrivers {
				if _, ok := d.messageDriver[driver.DriverId]; ok {
					filteredDrivers = append(filteredDrivers, driver)
				}
			}

		case st := <-d.rideStatuses:
			fmt.Println("Ride status received (ignored by logic): ", st)

		case <-d.ctx.Done():
			// Log shutdown message
			return nil
		}
	}
}

func (d *Distributor) AskDrivers(drivers []dto.DriverInfo, rideDetails dto.RideDetails){
	go func(){
		for _, driver := range drivers{
			var driverOffer dto.DriverRideOffer
			driverOffer.Type = "ride_offer"
			driverOffer.Offer_id = "offer_" + time.Now().String()
			driverOffer.Ride_id = rideDetails.Ride_id
			driverOffer.Ride_number = rideDetails.Ride_number

		}
	}
}
