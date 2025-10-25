package consumer

import (
	"context"
	"encoding/json"
	messagebrokerdto "ride-hail/internal/ride-service/core/domain/message_broker_dto"
	websocketdto "ride-hail/internal/ride-service/core/domain/websocket_dto"
	"ride-hail/internal/ride-service/core/ports"
	"time"

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
	publisher   ports.IRidesBroker
	rideService ports.IRidesService
}

func New(
	ctx context.Context,
	dispatcher ports.INotifyWebsocket,
	consumer ports.IBrokerConsumer,
	publisher ports.IRidesBroker,
	rideService ports.IRidesService,
) *Notification {
	return &Notification{
		ctx:         ctx,
		dispatcher:  dispatcher,
		consumer:    consumer,
		rideService: rideService,
		publisher:   publisher,
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

		}
	}
}

func (n *Notification) DriverResponse(msg amqp091.Delivery) error {
	m := messagebrokerdto.RideAcceptance{}

	err := json.Unmarshal(msg.Body, &m)
	if err != nil {
		return err
	}
	passengerId, rideNumber, err := n.rideService.StatusMatch(m.RideID, m.DriverID)

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
		Type: "ride_status_update",
		Data: payload,
	}

	n.dispatcher.WriteToUser(passengerId, eventMsg)

	m2 := messagebrokerdto.RideStatus{
		RideId: m.RideID,
		Status: "IN_PROGRESS",
		// TODO: add nice format 2024-12-16T10:34:00Z
		Timestamp:     time.Now().Format("2006-01-02T15:04:05"),
		DriverID:      m.DriverID,
		CorrelationID: msg.CorrelationId,
	}
	ctx, cancel := context.WithTimeout(n.ctx, time.Second*15)
	defer cancel()

	err = n.publisher.PushMessageToStatus(ctx, m2)
	if err != nil {
		return err
	}
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

	payload, err := json.Marshal(m1)
	if err != nil {
		return err
	}
	m := websocketdto.Event{
		Type: locationUpdates,
		Data: payload,
	}

	// TODO: define passengerId
	n.dispatcher.WriteToUser("", m)
	return nil
}
