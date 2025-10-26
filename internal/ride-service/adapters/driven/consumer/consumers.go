package consumer

import (
	"context"
	"encoding/json"
	messagebrokerdto "ride-hail/internal/ride-service/core/domain/message_broker_dto"
	websocketdto "ride-hail/internal/ride-service/core/domain/websocket_dto"
	"ride-hail/internal/ride-service/core/ports"

	"github.com/rabbitmq/amqp091-go"
)

const (
	// routing key
	driverResponse  = "driver.response.*"
	driverStatus    = "driver.status.*"
	locationUpdates = "location"

	// websocket type
	rideStatusUpdate     = "ride_status_update"
	driverLocationUpdate = "driver_location_update"
)

type Notification struct {
	ctx         context.Context
	dispatcher  ports.INotifyWebsocket
	consumer    ports.IBrokerConsumer
	rideService ports.IRidesService
	passengerService ports.IPassengerService
}

func New(
	ctx context.Context,
	dispatcher ports.INotifyWebsocket,
	consumer ports.IBrokerConsumer,
	passengerService ports.IPassengerService,
	rideService ports.IRidesService,
) *Notification {
	return &Notification{
		ctx:         ctx,
		dispatcher:  dispatcher,
		consumer:    consumer,
		rideService: rideService,
		passengerService: passengerService,
	}
}

func (n *Notification) Run() error {
	chDriverResponse, err := n.consumer.Consume(n.ctx, driverResponse)
	if err != nil {
		return err
	}

	// chDriverStatus, err := n.consumer.Consume(n.ctx, driverStatus)
	// if err != nil {
	// 	return err
	// }

	chLocation, err := n.consumer.Consume(n.ctx, locationUpdates)
	if err != nil {
		return err
	}

	go n.work(n.ctx, chDriverResponse, n.DriverResponse)
	go n.work(n.ctx, chLocation, n.LocationUpdate)

	return nil
}

func (n *Notification) work(
	ctx context.Context,
	ch <-chan amqp091.Delivery,
	Do func(msg amqp091.Delivery) error,
) {
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
	m := messagebrokerdto.RideAcceptance{}

	err := json.Unmarshal(msg.Body, &m)
	if err != nil {
		return err
	}
	passengerId, rideNumber, err := n.rideService.SetStatusMatch(m.RideID, m.DriverID)
	if err != nil {
		return err
	}

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
		return err
	}

	eventMsg := websocketdto.Event{
		Type: rideStatusUpdate,
		Data: payload,
	}

	n.dispatcher.WriteToUser(passengerId, eventMsg)
	return nil
}

func (n *Notification) LocationUpdate(msg amqp091.Delivery) error {

	m2 := messagebrokerdto.LocationUpdate{}

	err := json.Unmarshal(msg.Body, &m2)
	if err != nil {
		return err
	}
	// TODO: add calculation with time and distance calculation
	m1 := websocketdto.DriverLocationUpdate{
		RideID: m2.RideID,
		DriverLocation: websocketdto.Location{
			Lat: m2.Location.Lat,
			Lng: m2.Location.Lng,
		},
	}
	passengerId, err := n.rideService.FindPassengerByRideId(m2.RideID)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(m1)
	if err != nil {
		return err
	}
	m := websocketdto.Event{
		Type: driverLocationUpdate,
		Data: payload,
	}
	// TODO: define passengerId
	n.dispatcher.WriteToUser(passengerId, m)
	return nil
}
