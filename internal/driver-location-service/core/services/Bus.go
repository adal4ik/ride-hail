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
	// Tools
	broker driven.IDriverBroker
	ctx    context.Context
	log    mylogger.Logger
	wg     sync.WaitGroup
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
		ctx:            ctx,
		log:            log,
		wg:             sync.WaitGroup{},
	}
	return distributor
}

func (d *Distributor) MessageDistributor() error {
	log := d.log.Action("MessageDistributor")
	log.Info("Starting message distributor...")
	for {
		select {
		case requestDelivery := <-d.rideOffers:
			d.wg.Add(1)
			go d.handleRideRequest(requestDelivery)

		case statusDelivery := <-d.rideStatuses:
			d.wg.Add(1)
			go d.handleRideStatus(statusDelivery)

		case driverMsg := <-d.wsManager.GetFanIn():
			d.wg.Add(1)
			go d.handleDriverMessage(driverMsg)

		case <-d.ctx.Done():
			log.Info("Shutting down...")
			d.wg.Wait()
			return nil
		}
	}
}

func (d *Distributor) handleDriverMessage(msg dto.DriverMessage) {
	log := d.log.Action("handleDriverMessage")
	var LocationUpdate websocketdto.LocationUpdateMessage
	if err := json.Unmarshal(msg.Message, &LocationUpdate); err != nil {
		log.Error("Failed to unmarshal message:", err)
		return
	}
	log.Info("Handling driver message")
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
	d.driverService.UpdateLocation(d.ctx, dto.NewLocation{
		Latitude:        LocationUpdate.Latitude,
		Longitude:       LocationUpdate.Longitude,
		Accuracy_meters: LocationUpdate.AccuracyMeters,
		Speed_kmh:       LocationUpdate.SpeedKmh,
		Heading_Degrees: LocationUpdate.HeadingDegrees,
	}, msg.DriverID)

	if err := d.broker.PublishJSON(context.Background(), "location_fanout", "location", rmMessage); err != nil {
		log.Error("Failed to Publish location_fanout", err)
	}
}

func (d *Distributor) handleRideRequest(requestDelivery amqp.Delivery) {
	log := d.log.Action("handleRideRequest")
	var req dto.RideDetails

	if err := json.Unmarshal(requestDelivery.Body, &req); err != nil {
		log.Error("Error Unmarshalling request:", err)
		requestDelivery.Nack(false, false)
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
		log.Error("Failed to get appropriate drivers from db:", err, "ride-id", req.Ride_id)
		requestDelivery.Nack(false, false)
		return
	}

	var connectedDrivers []dto.DriverInfo
	for _, driver := range allDrivers {
		if d.wsManager.IsDriverConnected(driver.DriverId) {
			connectedDrivers = append(connectedDrivers, driver)
		}
	}
	log.Info(fmt.Sprintf("Found %d connected drivers for ride %s", len(connectedDrivers), req.Ride_id))
	d.sendRideOffers(connectedDrivers, req, requestDelivery)
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
				requestDelivery.Nack(false, false)
				return
			}
			if response.Accepted {
				d.handleDriverAcceptance(response, rideDetails, requestDelivery, driver)
				return
			} else {
				continue
			}
		case <-time.After(30 * time.Second):
			log.Info("No driver accepted the ride within timeout")
			continue
		}
	}
	log.Info("No drivers accepted this ride:", "RideID", rideDetails.Ride_id)
	time.Sleep(7 * time.Second)
	requestDelivery.Nack(false, true)
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
	requestDelivery.Ack(false)
	d.broker.PublishJSON(d.ctx, "driver_topic", fmt.Sprintf("driver.response.%s", driver.DriverId), driverMatch)

	log.Info("Ride accepted by driver", rideDetails.Ride_id, driverMatch.Driver_id)
}

func (d *Distributor) handleRideStatus(statusDelivery amqp.Delivery) {
	log := d.log.Action("handleRideStatus")
	var status messagebrokerdto.RideStatus
	if err := json.Unmarshal(statusDelivery.Body, &status); err != nil {
		log.Error("Failed to unmarshal the ride response message: ", err, "Message", statusDelivery.Body)
		statusDelivery.Nack(false, false)
		return
	}
	log.Info("Received ride status update:", status.RideId, status)
	driverID, err := d.driverService.GetDriverIdByRideId(context.Background(), status.RideId)
	if err != nil {
		log.Error("Failed to get driver ID by ride ID:", err, status.RideId)
		statusDelivery.Nack(false, true)
		return
	}
	switch status.Status {
	case "CANCELLED":
		cancelMessage := websocketdto.CanceledOrderMessage{
			WebSocketMessage: websocketdto.WebSocketMessage{
				Type: "Info",
			},
			RideID:  status.RideId,
			Status:  "canceled",
			Message: "Order was canceled",
		}
		d.wsManager.SendToDriver(d.ctx, driverID, cancelMessage)
		log.Info("Processing ride cancelation:", status.RideId)
		d.driverService.PayDriverMoney(d.ctx, driverID, status.Final_fare)
		d.driverService.UpdateDriverStatus(d.ctx, driverID, "AVAILABLE")
		log.Info("Driver status changed:", driverID)
		statusDelivery.Ack(false)

	case "MATCHED":
		rideDetails, err := d.driverService.GetRideDetailsByRideId(context.Background(), status.RideId)
		if err != nil {
			log.Error("Failed to get ride details by ride ID:", err, status.RideId)
			statusDelivery.Nack(false, true)
			return
		}
		rideDetails.WebSocketMessage = websocketdto.WebSocketMessage{
			Type: websocketdto.MessageTypeRideDetails,
		}
		d.driverService.UpdateDriverStatus(context.Background(), driverID, "EN_ROUTE")
		log.Info("Driver status changed:", driverID)

		driverStatus := messagebrokerdto.DriverStatus{
			DriverID:  driverID,
			RideID:    status.RideId,
			Status:    "EN_ROUTE",
			Timestamp: time.Now().String(),
		}
		d.broker.PublishJSON(d.ctx, "driver_topic", fmt.Sprintf("driver.status.%s", driverID), driverStatus)
		log.Info("Driver status send to rabbitmq", driverID)

		d.wsManager.SendToDriver(context.Background(), driverID, rideDetails)
		log.Info("Processing ride status update:", status.RideId)

		statusDelivery.Ack(false)
	case "COMPLETED":
		log.Info("ride completed", "final_fare", status.Final_fare)
		err := d.driverService.PayDriverMoney(d.ctx, driverID, status.Final_fare)
		if err != nil {
			log.Error("Failed to pay money to driver:", err)
			statusDelivery.Nack(false, false)
		}
	default:
		log.Warn("Ride status message undefined (sending to trash queue)", "status", status.Status)
		statusDelivery.Nack(false, false)
	}
}
