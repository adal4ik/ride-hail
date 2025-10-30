package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"ride-hail/internal/driver-location-service/core/ports/driver"
	"ride-hail/internal/mylogger"

	dto "ride-hail/internal/driver-location-service/core/domain/dto"
	messagebrokerdto "ride-hail/internal/driver-location-service/core/domain/message_broker_dto"
	websocketdto "ride-hail/internal/driver-location-service/core/domain/websocket_dto"
	driven "ride-hail/internal/driver-location-service/core/ports/driven"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Distributor struct {
	// Rabbit MQ
	rideOffers   <-chan amqp.Delivery
	rideStatuses <-chan amqp.Delivery
	// Websocket Handler
	wsManager     driven.WSConnectionMeneger
	driverService driver.IDriverService
	// Driver Messages
	driverMessages chan DriverMessage
	pendingOffers  map[string]*PendingOffer
	pendingMu      sync.RWMutex
	// Tools
	broker driven.IDriverBroker
	ctx    context.Context
	log    mylogger.Logger
}

type DriverMessage struct {
	DriverID string
	Message  []byte
}

type PendingOffer struct {
	RideID       string
	DriverID     string
	OfferID      string
	ExpiresAt    time.Time
	ResponseChan chan<- websocketdto.RideResponseMessage
}

func NewDistributor(
	ctx context.Context,
	rideOffers <-chan amqp.Delivery,
	rideStatuses <-chan amqp.Delivery,
	wsManager driven.WSConnectionMeneger,
	broker driven.IDriverBroker,
	driverService driver.IDriverService,
	log mylogger.Logger,
) *Distributor {
	distributor := &Distributor{
		rideOffers:     rideOffers,
		rideStatuses:   rideStatuses,
		wsManager:      wsManager,
		broker:         broker,
		driverService:  driverService,
		driverMessages: make(chan DriverMessage, 1000),
		pendingOffers:  make(map[string]*PendingOffer),
		ctx:            ctx,
		log:            log,
	}

	// go (*distributor).MessageDistributor()
	return distributor
}

func (d *Distributor) MessageDistributor() error {
	log := d.log.Action("MessageDistributor")
	log.Info("Starting message distributor...")
	for {
		select {
		case requestDelivery := <-d.rideOffers:
			go d.handleRideRequest(requestDelivery)

		case statusDelivery := <-d.rideStatuses:
			go d.handleRideStatus(statusDelivery)

		case driverMsg := <-d.wsManager.GetFanIn():
			log.Info("Getting message from FanIn....")
			go d.handleDriverMessage(driverMsg)

		case <-d.ctx.Done():
			return nil
		}
	}
}

func (d *Distributor) RegisterDriverChannel(driverID string, incoming <-chan []byte) {
	go d.handleDriverConnection(driverID, incoming)
}

func (d *Distributor) handleDriverConnection(driverID string, incoming <-chan []byte) {
	for {
		select {
		case <-d.ctx.Done():
			return
		case message, ok := <-incoming:
			if !ok {
				return
			}
			d.driverMessages <- DriverMessage{
				DriverID: driverID,
				Message:  message,
			}
		}
	}
}

// IMPLEMENT
func (d *Distributor) handleDriverMessage(msg dto.DriverMessage) {
	log := d.log.Action("handleDriverMessage")
	var LocationUpdate websocketdto.LocationUpdateMessage
	if err := json.Unmarshal(msg.Message, &LocationUpdate); err != nil {
		log.Error("Failed to unmarshal message:", err)
		return
	}
	d.driverService.UpdateLocation(context.Background(), dto.NewLocation{
		Latitude:        LocationUpdate.Latitude,
		Longitude:       LocationUpdate.Longitude,
		Accuracy_meters: LocationUpdate.AccuracyMeters,
		Speed_kmh:       LocationUpdate.SpeedKmh,
		Heading_Degrees: LocationUpdate.HeadingDegrees,
	}, msg.DriverID)
	ride_id, err := d.driverService.GetRideIdByDriverId(context.Background(), msg.DriverID)
	if err != nil {
		log.Error("Failed to get ride id from db:", err)
		return
	}
	rmMessage := messagebrokerdto.LocationUpdate{
		DriverID: msg.DriverID,
		RideID:   ride_id,
		Location: messagebrokerdto.Location{
			Lng: LocationUpdate.Longitude,
			Lat: LocationUpdate.Latitude,
		},
		SpeedKmh:       LocationUpdate.SpeedKmh,
		HeadingDegrees: LocationUpdate.HeadingDegrees,
		Timestamp:      time.Now().String(),
	}

	if err := d.broker.PublishJSON(context.Background(), "location_fanout", "location", rmMessage); err != nil {
		log.Error("Failed to Publish location_fanout", err)
	}

}

func (d *Distributor) handleRideRequest(requestDelivery amqp.Delivery) {
	log := d.log.Action("handleRideRequest")
	var req dto.RideDetails

	if err := json.Unmarshal(requestDelivery.Body, &req); err != nil {
		log.Error("Error Unmarshalling request:", err)
		requestDelivery.Nack(false, true)
		return
	}
	if len(d.wsManager.GetConnectedDrivers()) == 0 {
		log.Info("No drivers online to handle ride request (sleeping):", "ride-id", req.Ride_id)
		time.Sleep(7 * time.Second)
		requestDelivery.Nack(false, true)
		return
	}
	log.Info("Processing ride request:", req.Ride_id)
	ctx := context.Background()
	allDrivers, err := d.driverService.FindAppropriateDrivers(ctx,
		req.Pickup_location.Lng,
		req.Destination_location.Lat,
		req.Ride_type,
	)
	if err != nil {
		log.Error("Failed to find appropriate drivers:", err, req.Ride_id)
		// requestDelivery.Nack(false, true)
		return
	}

	var connectedDrivers []dto.DriverInfo
	for _, driver := range allDrivers {
		if d.wsManager.IsDriverConnected(driver.DriverId) {
			connectedDrivers = append(connectedDrivers, driver)
		}
	}
	log.Info(fmt.Sprintf("Found %d connected drivers for ride %s", len(connectedDrivers), req.Ride_id))
	go d.sendRideOffers(connectedDrivers, req, requestDelivery)
}

func (d *Distributor) sendRideOffers(drivers []dto.DriverInfo, rideDetails dto.RideDetails, requestDelivery amqp.Delivery) {
	log := d.log.Action("sendRideOffers")
	_ = log

	for _, driver := range drivers {
		offer := websocketdto.RideOfferMessage{
			WebSocketMessage: websocketdto.WebSocketMessage{
				Type: websocketdto.MessageTypeRideOffer,
			},
			OfferID:    fmt.Sprintf("offer_%s_%s", rideDetails.Ride_id, driver.DriverId),
			RideID:     rideDetails.Ride_id,
			RideNumber: rideDetails.Ride_number,
			PickupLocation: websocketdto.Location{
				Latitude:  rideDetails.Pickup_location.Lat,
				Longitude: rideDetails.Pickup_location.Lng,
				Address:   rideDetails.Pickup_location.Address,
			},
			DestinationLocation: websocketdto.Location{
				Latitude:  rideDetails.Destination_location.Lat,
				Longitude: rideDetails.Destination_location.Lng,
				Address:   rideDetails.Destination_location.Address,
			},
			EstimatedFare:                rideDetails.Estimated_fare,
			DriverEarnings:               rideDetails.Estimated_fare * 0.8,
			DistanceToPickupKm:           driver.Distance,
			EstimatedRideDurationMinutes: int(driver.Distance / 0.75),
			ExpiresAt:                    time.Now().Add(30 * time.Second),
		}
		log.Info("Sending message to driver:", offer)
		d.wsManager.SendToDriver(context.Background(), driver.DriverId, offer)
		driverResponse, err := d.wsManager.GetDriverMessages(driver.DriverId)
		if err != nil {
			log.Error("Failed to get messages for driver", err, driver.DriverId)
			continue
		}
		select {
		case data := <-driverResponse:
			var response websocketdto.RideResponseMessage
			if err := json.Unmarshal(data, &response); err != nil {
				log.Error("Failed to unmarshal driver response:", err, driver.DriverId)
				requestDelivery.Nack(false, true)
				continue
			}
			if response.Accepted {
				d.handleDriverAcceptance(response, rideDetails, requestDelivery, driver)
			} else {
				requestDelivery.Nack(false, true)
			}
			return
		case <-time.After(30 * time.Second):
			log.Info("No driver accepted the ride within timeout")
			requestDelivery.Nack(false, true)
			continue
		}
	}
}

func (d *Distributor) handleLocationUpdate(driverID string, update websocketdto.LocationUpdateMessage) {
	log := d.log.Action("handleLocationUpdate")
	ctx := context.Background()
	data := dto.NewLocation{
		Latitude:        update.Latitude,
		Longitude:       update.Longitude,
		Accuracy_meters: update.AccuracyMeters,
		Speed_kmh:       update.SpeedKmh,
		Heading_Degrees: update.HeadingDegrees,
	}
	_, err := d.driverService.UpdateLocation(ctx, data, driverID)
	if err != nil {
		log.Error("Failed to update driver location", err)
		return
	}
	locationUpdate := messagebrokerdto.LocationUpdate{
		DriverID: driverID,
		RideID:   "asdasda",
		Location: messagebrokerdto.Location{
			Lng: update.Longitude,
			Lat: update.Latitude,
		},
		SpeedKmh:       update.SpeedKmh,
		HeadingDegrees: update.HeadingDegrees,
		Timestamp:      time.Now().String(),
	}

	if err := d.broker.PublishJSON(ctx, "location_updates", "", locationUpdate); err != nil {
		log.Error("Failed to publish location update", err)
	}
}

func (d *Distributor) handleDriverAcceptance(response websocketdto.RideResponseMessage, rideDetails dto.RideDetails, requestDelivery amqp.Delivery, driver dto.DriverInfo) {
	log := d.log.Action("handleDriverAcceptance")
	_ = log
	driverMatch := dto.DriverMatchResponse{
		Ride_id:                   rideDetails.Ride_id,
		Driver_id:                 driver.DriverId,
		Accepted:                  true,
		Estimated_arrival_minutes: 5,
		Driver_location: dto.Location{
			Latitude:  response.CurrentLocation.Latitude,
			Longitude: response.CurrentLocation.Longitude,
		},
		Driver_info: dto.DriverInfoForResponse{
			Name:    driver.Name,
			Vehicle: driver.Vehicle,
			Rating:  driver.Rating,
		},
	}
	d.driverService.UpdateDriverStatus(context.Background(), driverMatch.Driver_id, "BUSY")

	requestDelivery.Ack(false)
	d.broker.PublishJSON(d.ctx, "driver_topic", fmt.Sprintf("driver.response.%s", driver.DriverId), driverMatch)

	log.Info("Ride accepted by driver", rideDetails.Ride_id, driverMatch.Driver_id)
}

func (d *Distributor) handleRideStatus(statusDelivery amqp.Delivery) {
	log := d.log.Action("handleRideStatus")
	var status messagebrokerdto.RideStatus
	if err := json.Unmarshal(statusDelivery.Body, &status); err != nil {
		log.Error("Failed to unmarshal the ride response message: ", err)
		statusDelivery.Nack(false, true)
		return
	}
	log.Info("Received ride status update:", status.RideId, status)
	driverID, err := d.driverService.GetDriverIdByRideId(context.Background(), status.RideId)
	if err != nil {
		log.Error("Failed to get driver ID by ride ID:", err, status.RideId)
		statusDelivery.Nack(false, true)
		return
	}
	rideDetails, err := d.driverService.GetRideDetailsByRideId(context.Background(), status.RideId)
	if err != nil {
		log.Error("Failed to get ride details by ride ID:", err, status.RideId)
		statusDelivery.Nack(false, true)
		return
	}
	rideDetails.WebSocketMessage = websocketdto.WebSocketMessage{
		Type: websocketdto.MessageTypeRideDetails,
	}
	d.wsManager.SendToDriver(context.Background(), driverID, rideDetails)
	log.Info("Processing ride status update:", status.RideId)
	statusDelivery.Ack(false)
}
