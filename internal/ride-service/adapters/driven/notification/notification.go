package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"ride-hail/internal/mylogger"
	"ride-hail/internal/ride-service/core/ports"
	"sync"

	messagebrokerdto "ride-hail/internal/ride-service/core/domain/message_broker_dto"
	websocketdto "ride-hail/internal/ride-service/core/domain/websocket_dto"

	"github.com/rabbitmq/amqp091-go"
)

const (
	// routing key
	driverResponse  = "driver_responses"
	driverStatus    = "driver_status"
	locationUpdates = "location_updates"

	// websocket type
	rideStatusUpdate     = "ride_status_update"
	driverLocationUpdate = "driver_location_update"
)

type Notification struct {
	ctx              context.Context
	wg               *sync.WaitGroup
	log              mylogger.Logger
	dispatcher       ports.INotifyWebsocket
	consumer         ports.IRidesBroker
	rideService      ports.IRidesService
	passengerService ports.IPassengerService
}

func New(
	ctx context.Context,
	wg *sync.WaitGroup,
	log mylogger.Logger,
	dispatcher ports.INotifyWebsocket,
	consumer ports.IRidesBroker,
	passengerService ports.IPassengerService,
	rideService ports.IRidesService,
) *Notification {
	return &Notification{
		ctx:              ctx,
		wg:               wg,
		log:              log,
		dispatcher:       dispatcher,
		consumer:         consumer,
		rideService:      rideService,
		passengerService: passengerService,
	}
}

func (n *Notification) Run() error {
	chDriverResponse, err := n.consumer.ConsumeMessageFromDrivers(n.ctx, driverResponse, "")
	if err != nil {
		return err
	}

	chDriverStatus, err := n.consumer.ConsumeMessageFromDrivers(n.ctx, driverStatus, "")
	if err != nil {
		return err
	}

	chLocation, err := n.consumer.ConsumeMessageFromDrivers(n.ctx, locationUpdates, "")
	if err != nil {
		return err
	}

	n.wg.Add(3)
	go n.work(n.ctx, chDriverResponse, n.DriverResponse)
	go n.work(n.ctx, chDriverStatus, n.DriverStatusUpdate)
	go n.work(n.ctx, chLocation, n.LocationUpdate)

	return nil
}

func (n *Notification) work(
	ctx context.Context,
	ch <-chan amqp091.Delivery,
	Do func(msg amqp091.Delivery) error,
) {
	log := n.log.Action("work")
	defer func() {
		log.Info("one worker is done")
		n.wg.Done()
	}()
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}

			err := Do(msg)
			if err != nil {
				continue
			}
		case <-ctx.Done():
			return
		}
	}
}

func (n *Notification) DriverResponse(msg amqp091.Delivery) error {
	log := n.log.Action("DriverReponse")
	m := messagebrokerdto.RideAcceptance{}
	err := json.Unmarshal(msg.Body, &m)
	if err != nil {
		log.Error("cannot unmarshal", err)
		msg.Nack(false, false)
		return err
	}
	fmt.Printf("Driver Response Message: %+v\n", string(msg.Body))
	passengerId, rideNumber, err := n.rideService.SetStatusMatch(m.RideID, m.DriverID)
	if err != nil {
		log.Error("cannot set status to match", err)
		msg.Nack(false, false)
		return err
	}
	log.Info("ride status set to match", "ride-id", m.RideID, "driver-id", m.DriverID)
	m1 := websocketdto.RideStatusUpdateDto{
		RideID:     m.RideID,
		RideNumber: rideNumber,
		Status:     "MATCHED",
		DriverInfo: websocketdto.DriverInfo{
			DriverID: m.DriverID,
			Name:     m.DriverInfo.Name,
			Rating:   m.DriverInfo.Rating,
			Vehicle:  websocketdto.Vehicle(m.DriverInfo.Vehicle),
		},
		CorrelationID: msg.CorrelationId,
	}

	payload, err := json.Marshal(m1)
	if err != nil {
		log.Error("cannot marshal", err)
		msg.Nack(false, false)
		return err
	}

	eventMsg := websocketdto.Event{
		Type: rideStatusUpdate,
		Data: payload,
	}

	n.dispatcher.WriteToUser(passengerId, eventMsg)

	return msg.Ack(false)
}

func (n *Notification) LocationUpdate(msg amqp091.Delivery) error {
	log := n.log.Action("LocationUpdate")
	m2 := messagebrokerdto.LocationUpdate{}
	log.Info("nigga what did i get?", "body", string(msg.Body))
	err := json.Unmarshal(msg.Body, &m2)
	if err != nil {
		log.Error("cannot unmarshal", err)
		msg.Nack(false, false)
		return err
	}
	passengerId, estimatedTime, distance, err := n.rideService.EstimateDistance(m2.RideID, m2.Location.Lng, m2.Location.Lat, m2.SpeedKmh)
	if err != nil {
		log.Error("cannot estimate distance", err)
		msg.Nack(false, false)
		return err
	}
	m1 := websocketdto.DriverLocationUpdate{
		RideID: m2.RideID,
		DriverLocation: websocketdto.Location{
			Lat: m2.Location.Lat,
			Lng: m2.Location.Lng,
		},
		EstimatedArrival:   estimatedTime,
		DistanceToPickupKm: distance,
	}

	payload, err := json.Marshal(m1)
	if err != nil {
		log.Error("cannot marshal", err)
		msg.Nack(false, false)
		return err
	}
	m := websocketdto.Event{
		Type: driverLocationUpdate,
		Data: payload,
	}
	log.Debug("get locationUpdate")
	n.dispatcher.WriteToUser(passengerId, m)

	msg.Ack(false)
	return nil
}

func (n *Notification) DriverStatusUpdate(msg amqp091.Delivery) error {
	log := n.log.Action("DriverStatusUpdate")
	driverStatusUpdateMessage := messagebrokerdto.DriverStatusUpdate{}

	err := json.Unmarshal(msg.Body, &driverStatusUpdateMessage)
	if err != nil {
		log.Error("cannot unmarshal", err)
		msg.Nack(false, false)
		return err
	}

	passengerId, data, err := n.rideService.UpdateRideStatus(driverStatusUpdateMessage)
	if err != nil {
		log.Error("cannot update ride status", err)
		msg.Nack(false, false)
		return err
	}

	n.dispatcher.WriteToUser(passengerId, data)

	msg.Ack(false)
	return nil
}

// }
