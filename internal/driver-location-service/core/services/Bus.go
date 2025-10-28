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
	// RabbitMQ Channels
	rideOffers   <-chan amqp.Delivery
	rideStatuses <-chan amqp.Delivery
	// Websocket Channels
	wsManager driven.WSConnectionMeneger

	// Sevices
	broker        driven.IDriverBroker
	driverService *DriverService

	// Inner Channels
	driverMessage chan DriverMessage
	ctx           context.Context
}

func NewDistributor(
	ctx context.Context,
	messageDriver map[string]chan dto.DriverRideOffer,
	rideOffers <-chan amqp.Delivery,
	rideStatuses <-chan amqp.Delivery,
	driverResponses map[string]chan dto.DriverResponse,
	broker driven.IDriverBroker,
	driverService *DriverService,
	wsManager driven.WSConnectionMeneger,
) *Distributor {
	return &Distributor{
		rideOffers:      rideOffers,
		rideStatuses:    rideStatuses,
		driverResponses: driverResponses,
		messageDriver:   messageDriver,
		ctx:             ctx,
		broker:          broker,
		driverService:   driverService,
		wsManager:       wsManager,
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
			go d.AskDrivers(filteredDrivers, req, requestDelivery)

		case statusDelivery := <-d.rideStatuses:
			fmt.Println("Ride status received (ignored by logic): ", st)
		case driverMsg := <-d.driverMessage:
			// Send message to driver via WebSocket
		case <-d.ctx.Done():
			// Log shutdown message
			return nil
		}
	}
}

func (d *Distributor) RegisterDriverChannel(driverID string, incoming <-chan []byte, outgoing chan<- []byte) {
	go d.handlerDriverConnection(driverID, incoming, outgoing)
}

func (d *Distributor) handlerDriverConnection(driverID string, incoming <-chan []byte, outgoing chan<- []byte) {
	for {
		select {
		case msg, ok := <-incoming:
			if !ok {
				return
			}
			d.driverMessage <- DriverMessage{
				DriverID: driverID,
				Message:  msg,
			}
		case <-d.ctx.Done():
			// Log shutdown message
			return
		}
	}
}

func (d *Distributor) handleDriverMessages(msg DriverMessage) {
	var driverResponse dto.DriverResponse
	if err := json.Unmarshal(msg.Message, &driverResponse); err != nil {
		fmt.Println("Error unmarshalling driver response:", err)
		return
	}
	switch driverResponse.Type {
	case "ride_response":
	}
}

func (d *Bus) AskDrivers(drivers []dto.DriverInfo, rideDetails dto.RideDetails, requestDelivery amqp.Delivery) {
	fmt.Println("Finding a driver")
	var driverMatch dto.DriverMatchResponse
	for _, driver := range drivers {
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
			DistanceToPickUp:  driver.Distance,
			EstimatedDuration: int(driver.Distance / 0.75),
			ExpiredAt:         time.Now().Add(time.Duration(int(driver.Distance/45)+5) * time.Minute).String(),
		}
		d.messageDriver[driver.DriverId] <- driverOffer

		select {
		case answer := <-d.driverResponses[driver.DriverId]:
			if !answer.Accepted {
				continue
			}
		case <-time.After(time.Duration(30) * time.Second):
			fmt.Println("Offer is expired")
			continue
		}

		driverMatch = dto.DriverMatchResponse{
			Ride_id:                   rideDetails.Ride_id,
			Driver_id:                 driver.DriverId,
			Accepted:                  true,
			Estimated_arrival_minutes: int(driver.Distance / 45),
			Driver_location: dto.Location{
				Latitude:  driver.Latitude,
				Longitude: driver.Longitude,
			},
			Driver_info: driver,
		}
		break
	}
	if !driverMatch.Accepted {
		requestDelivery.Nack(false, true)
	} else {
		fmt.Println("Found a driver")
		err := d.broker.PublishJSON(d.ctx, "driver_topic", fmt.Sprintf("driver.response.%s", rideDetails.Ride_id), driverMatch)
		if err != nil {
			fmt.Println("Error publishing driver match response:", err)
			return
		}
		// Update Driver Status
		d.driverService.UpdateDriverStatus(d.ctx, driverMatch.Driver_id, "BUSY")
		// Acknowledge original message
		requestDelivery.Ack(false)
	}
}
