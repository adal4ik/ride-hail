package services

import (
	"context"
	"fmt"

	"ride-hail/internal/driver-location-service/core/domain/dto"
	driven "ride-hail/internal/driver-location-service/core/ports/driven"
)

// ADD LOGGER
type Distributor struct {
	rideOffers      *chan dto.RideDetails
	rideStatuses    *chan dto.RideStatusUpdate
	driverResponses map[string]chan dto.DriverResponse
	messageDriver   map[string]chan dto.DriverRideOffer
	ctx             context.Context
	broker          driven.IDriverBroker
}

func NewDistributor(
	ctx context.Context,
	messageDriver map[string]chan dto.DriverRideOffer,
	rideOffers *chan dto.RideDetails,
	rideStatuses *chan dto.RideStatusUpdate,
	driverResponses map[string]chan dto.DriverResponse,
	broker driven.IDriverBroker,
) *Distributor {
	return &Distributor{
		rideOffers:      rideOffers,
		rideStatuses:    rideStatuses,
		driverResponses: driverResponses,
		messageDriver:   messageDriver,
		ctx:             ctx,
		broker:          broker,
	}
}

func (d *Distributor) MessageDistributor() error {
	for {
		select {
		case msg := <-*d.rideOffers:
			fmt.Println("Distributing ride offer to driver: ", msg)
			var onlineDrivers []string
			for key := range d.messageDriver {
				onlineDrivers = append(onlineDrivers, key)
			}

			// Logic Might Be Better
			if len(onlineDrivers) == 0 {
				err := d.broker.PublishJSON(d.ctx, "driver_topic", fmt.Sprintf("driver.response.%s", msg.Ride_id),
					struct {
						Ride_id string `json:"ride_id"`
						Status  string `json:"status"`
					}{
						Ride_id: msg.Ride_id,
						Status:  "Not Found",
					},
				)
				if err != nil {
					// Log the err
					fmt.Println(err.Error())
				}
				continue
			}
			condidateDriver := onlineDrivers[0]
			d.messageDriver[condidateDriver] <- dto.DriverRideOffer{
				Type:            "ride_offer",
				Ride_id:         msg.Ride_id,
				Passenger_name:  "John Doe",
				Passenger_phone: "+1234567890",
				Pickup_location: dto.LocationDetail{
					Lat:     msg.Pickup_location.Lat,
					Lng:     msg.Pickup_location.Lng,
					Address: msg.Pickup_location.Address,
				},
			}
			driverResponse := <-d.driverResponses[condidateDriver]
			fmt.Println("Received driver response: ", driverResponse)
			err := d.broker.PublishJSON(d.ctx, "driver_topic", fmt.Sprintf("driver.response.%s", driverResponse.Ride_id), driverResponse)
			if err != nil {
				fmt.Println("Failed to publish driver response: ", err)
			}
		case st := <-*d.rideStatuses:
			fmt.Println("Ride status received (ignored by logic): ", st)

		case <-d.ctx.Done():
			// Log shutdown message
			return nil
		}
	}
}
