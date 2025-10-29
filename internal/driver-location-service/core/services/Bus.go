package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	dto "ride-hail/internal/driver-location-service/core/domain/dto"
	messagebrokerdto "ride-hail/internal/driver-location-service/core/domain/message_broker_dto"
	websocketdto "ride-hail/internal/driver-location-service/core/domain/websocket_dto"
	driven "ride-hail/internal/driver-location-service/core/ports/driven"
	"ride-hail/internal/driver-location-service/core/ports/driver"
	"ride-hail/internal/mylogger"

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

	go (*distributor).MessageDistributor()
	return distributor
}

func (d *Distributor) MessageDistributor() error {
	for {
		select {
		case requestDelivery := <-d.rideOffers:
			go d.handleRideRequest(requestDelivery)

		case statusDelivery := <-d.rideStatuses:
			go d.handleRideStatus(statusDelivery)

		case driverMsg := <-d.driverMessages:
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

func (d *Distributor) handleDriverMessage(msg DriverMessage) {
	var baseMsg websocketdto.WebSocketMessage
	if err := json.Unmarshal(msg.Message, &baseMsg); err != nil {
		fmt.Printf("Invalid message format from driver %s: %v\n", msg.DriverID, err)
		return
	}

	switch baseMsg.Type {
	case websocketdto.MessageTypeRideResponse:
		var response websocketdto.RideResponseMessage
		if err := json.Unmarshal(msg.Message, &response); err != nil {
			fmt.Printf("Invalid ride response from driver %s: %v\n", msg.DriverID, err)
			return
		}
		d.handleRideResponse(msg.DriverID, response)

	case websocketdto.MessageTypeLocationUpdate:
		var update websocketdto.LocationUpdateMessage
		if err := json.Unmarshal(msg.Message, &update); err != nil {
			fmt.Printf("Invalid location update from driver %s: %v\n", msg.DriverID, err)
			return
		}
		d.handleLocationUpdate(msg.DriverID, update)
	default:
		fmt.Printf("Unknown message type %s from driver %s\n", baseMsg.Type, msg.DriverID)
	}
}

func (d *Distributor) handleRideRequest(requestDelivery amqp.Delivery) {
	var req dto.RideDetails
	if err := json.Unmarshal(requestDelivery.Body, &req); err != nil {
		fmt.Println("Error unmarshaling ride request:", err)
		requestDelivery.Nack(false, true)
		return
	}

	if d.wsManager.GetDriversCount(d.ctx) == 0 {
		time.Sleep(7 * time.Second)
		requestDelivery.Nack(false, true)
		return
	}
	ctx := context.Background()
	allDrivers, err := d.driverService.FindAppropriateDrivers(ctx,
		req.Pickup_location.Lng,
		req.Destination_location.Lat,
		req.Ride_type)
	if err != nil {
		fmt.Println("Error finding drivers:", err)
		requestDelivery.Nack(false, true)
		return
	}

	var connectedDrivers []dto.DriverInfo
	for _, driver := range allDrivers {
		if d.wsManager.IsDriverConnected(driver.DriverId) {
			connectedDrivers = append(connectedDrivers, driver)
		}
	}

	go d.sendRideOffers(connectedDrivers, req, requestDelivery)
}

func (d *Distributor) sendRideOffers(drivers []dto.DriverInfo, rideDetails dto.RideDetails, requestDelivery amqp.Delivery) {
	responseChan := make(chan websocketdto.RideResponseMessage, len(drivers))
	defer close(responseChan)

	// Отправляем предложения всем драйверам
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

		pendingOffer := PendingOffer{
			RideID:       rideDetails.Ride_id,
			DriverID:     driver.DriverId,
			OfferID:      offer.OfferID,
			ExpiresAt:    offer.ExpiresAt,
			ResponseChan: responseChan,
		}

		d.pendingMu.Lock()
		d.wsManager.SendToDriver(d.ctx, driver.DriverId, pendingOffer)
		d.pendingMu.Unlock()

		select {
		case response := <-responseChan:
			if response.Accepted {
				d.handleDriverAcceptance(response, rideDetails, requestDelivery)
			} else {
				requestDelivery.Nack(false, true)
			}
		case <-time.After(30 * time.Second):
			fmt.Println("No driver accepted the ride within timeout")
			requestDelivery.Nack(false, true)
			break
		}

		message, err := json.Marshal(offer)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		if err := d.wsManager.SendToDriver(context.Background(), driver.DriverId, message); err != nil {
			fmt.Printf("Failed to send offer to driver %s: %v\n", driver.DriverId, err)
			d.pendingMu.Lock()
			delete(d.pendingOffers, offer.OfferID)
			d.pendingMu.Unlock()
		}

		go d.cleanupPendingOffer(offer.OfferID, offer.ExpiresAt)
	}
}

func (d *Distributor) handleRideResponse(driverID string, response websocketdto.RideResponseMessage) {
	d.pendingMu.RLock()
	pendingOffer, exists := d.pendingOffers[response.OfferID]
	d.pendingMu.RUnlock()

	if !exists {
		fmt.Printf("Received response for unknown offer: %s\n", response.OfferID)
		return
	}

	if pendingOffer.DriverID != driverID {
		fmt.Printf("Driver %s attempted to respond to offer for driver %s\n", driverID, pendingOffer.DriverID)
		return
	}

	// Отправляем ответ в канал
	select {
	case pendingOffer.ResponseChan <- response:
		// Ответ отправлен
	default:
		fmt.Printf("Response channel full for offer: %s\n", response.OfferID)
	}

	// Удаляем ожидающее предложение
	d.pendingMu.Lock()
	delete(d.pendingOffers, response.OfferID)
	d.pendingMu.Unlock()
}

func (d *Distributor) cleanupPendingOffer(offerID string, expiresAt time.Time) {
	time.Sleep(time.Until(expiresAt) + time.Second) // +1 секунда на всякий случай

	d.pendingMu.Lock()
	if _, exists := d.pendingOffers[offerID]; exists {
		delete(d.pendingOffers, offerID)
		fmt.Printf("Cleaned up expired offer: %s\n", offerID)
	}
	d.pendingMu.Unlock()
}

func (d *Distributor) handleLocationUpdate(driverID string, update websocketdto.LocationUpdateMessage) {
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
		fmt.Printf("Failed to update driver location: %v\n", err)
		return
	}
	locationUpdate := messagebrokerdto.LocationUpdate{
		DriverID: driverID,
		// Logic for getting Ride ID
		RideID: "asdasda",
		Location: messagebrokerdto.Location{
			Lng: update.Longitude,
			Lat: update.Latitude,
		},
		SpeedKmh:       update.SpeedKmh,
		HeadingDegrees: update.HeadingDegrees,
		Timestamp:      time.Now().String(),
	}

	if err := d.broker.PublishJSON(ctx, "location_fanout", "", locationUpdate); err != nil {
		fmt.Printf("Failed to publish location update: %v\n", err)
	}
}

func (d *Distributor) handleDriverAcceptance(response websocketdto.RideResponseMessage, rideDetails dto.RideDetails, requestDelivery amqp.Delivery) {
	// Создаем ответ для ride-service
	driverMatch := dto.DriverMatchResponse{
		Ride_id:                   rideDetails.Ride_id,
		Driver_id:                 response.RideID, // Здесь должен быть driverID
		Accepted:                  true,
		Estimated_arrival_minutes: 5, // Рассчитать на основе расстояния
		Driver_location: dto.Location{
			Latitude:  response.CurrentLocation.Latitude,
			Longitude: response.CurrentLocation.Longitude,
		},
		// Заполнить driver_info из базы данных
	}
	// Обновляем статус драйвера
	d.driverService.UpdateDriverStatus(context.Background(), driverMatch.Driver_id, "BUSY")

	// Подтверждаем сообщение
	requestDelivery.Ack(false)

	fmt.Printf("Ride %s accepted by driver %s\n", rideDetails.Ride_id, driverMatch.Driver_id)
}

func (d *Distributor) handleRideStatus(statusDelivery amqp.Delivery) {
}
