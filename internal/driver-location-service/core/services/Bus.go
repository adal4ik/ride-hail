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
			fmt.Println("New ride request received", string(requestDelivery.Body))
			fmt.Println("Available drivers: ", len(d.messageDriver))
			if len(d.messageDriver) == 0 {
				requestDelivery.Nack(false, true)
				time.Sleep(7 * time.Second)
				continue
			}
			// Logic for getting write driver
			var req dto.RideDetails
			if err := json.Unmarshal(requestDelivery.Body, &req); err != nil {
				fmt.Println(err.Error())
				requestDelivery.Nack(false, true)
				continue
			}
			ctx := context.Background()
			allDrivers, err := d.driverService.FindAppropriateDrivers(ctx, req.Pickup_location.Lng, req.Destination_location.Lat, req.Ride_type)
			if err != nil {
				fmt.Println(err.Error())
				requestDelivery.Nack(false, true)
				continue
			}
			// Got filtered Drivers
			var filteredDrivers []dto.DriverInfo
			for _, driver := range allDrivers {
				if _, ok := d.messageDriver[driver.DriverId]; ok {
					filteredDrivers = append(filteredDrivers, driver)
				}
			}
			if len(filteredDrivers) == 0 {
				requestDelivery.Nack(false, true)
				time.Sleep(7 * time.Second)
				continue
			}
			// Ask Drivers
			d.AskDrivers(filteredDrivers, req, requestDelivery)

		case st := <-d.rideStatuses:
			fmt.Println("Ride status received (ignored by logic): ", st)

		case <-d.ctx.Done():
			// Log shutdown message
			return nil
		}
	}
}

func (d *Distributor) AskDrivers(drivers []dto.DriverInfo, rideDetails dto.RideDetails, requestDelivery amqp.Delivery) {
	go func() {
		fmt.Println("Finding a driver")
		var driverMatch dto.DriverMatchResponse
		for _, driver := range drivers {
			ctx := context.Background()
			distanceToPickUp, minutes, err := d.driverService.CalculateRideDetails(
				ctx,
				dto.Location{
					Latitude:  driver.Latitude,
					Longitude: driver.Longitude,
				},
				dto.Location{
					Latitude:  rideDetails.Pickup_location.Lat,
					Longitude: rideDetails.Pickup_location.Lng,
				},
			)
			if err != nil {
				fmt.Println(err.Error())
				break
			}
			driverOffer := dto.DriverRideOffer{
				Type:        "ride_offer",
				Offer_id:    "offer_" + time.Now().String(),
				Ride_id:     rideDetails.Ride_id,
				Ride_number: rideDetails.Ride_number,
				Pickup_location: dto.LocationDetail{
					Lat:     rideDetails.Pickup_location.Lat,
					Lng:     rideDetails.Pickup_location.Lng,
					Address: rideDetails.Pickup_location.Address,
				},
				Destination_location: dto.LocationDetail{
					Lat:     rideDetails.Destination_location.Lat,
					Lng:     rideDetails.Destination_location.Lng,
					Address: rideDetails.Destination_location.Address,
				},
				Estimated_fare:    rideDetails.Estimated_fare,
				Driver_earnings:   rideDetails.Estimated_fare * 0.8,
				DistanceToPickUp:  distanceToPickUp,
				EstimatedDuration: minutes,
				ExpiredAt:         time.Now().Add(time.Duration(minutes+5) * time.Minute).String(),
			}
			d.messageDriver[driver.DriverId] <- driverOffer
			accepted := 0
			for {
				select {
				case answer := <-d.driverResponses[driver.DriverId]:
					if answer.Accepted {
						accepted = 1
					}
					break
				case <-time.After(time.Duration(30) * time.Second):
					fmt.Println("Offer is expired")
					accepted = 2
					break
				default:
					// do nothing
				}
				if accepted != 0 {
					break
				}
			}

			if accepted == 1 {
				driverMatch = dto.DriverMatchResponse{
					Ride_id:                   rideDetails.Ride_id,
					Driver_id:                 driver.DriverId,
					Accepted:                  true,
					Estimated_arrival_minutes: minutes,
					Driver_location: dto.Location{
						Latitude:  driver.Latitude,
						Longitude: driver.Longitude,
					},
					Driver_info:    driver,
					Correlation_id: rideDetails.Correlation_id,
				}
				break
			}
		}
		if !driverMatch.Accepted {
			requestDelivery.Nack(false, true)
		} else {
			fmt.Println("Found a driver")
			responseBody, err := json.Marshal(driverMatch)
			if err != nil {
				fmt.Println("Error marshalling driver match response:", err)
				return
			}
			err = d.broker.PublishJSON(d.ctx, "drive_topic", fmt.Sprintf("driver.response.%s", rideDetails.Ride_id), responseBody)
			if err != nil {
				fmt.Println("Error publishing driver match response:", err)
				return
			}
			// Update Driver Status
			d.driverService.UpdateDriverStatus(d.ctx, driverMatch.Driver_id, "BUSY")
			// Acknowledge original message
			requestDelivery.Ack(false)
		}
	}()
}
