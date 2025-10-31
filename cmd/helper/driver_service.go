package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	websocketdto "ride-hail/internal/driver-location-service/core/domain/websocket_dto"
)

type DriverService struct {
	driverID   string
	jwtToken   string
	currentLat float64
	currentLng float64
	httpClient *HTTPClient
	wsClient   *WebSocketClient
	logger     *Logger
	ctx        context.Context
}

func NewDriverService(ctx context.Context, driverID, jwtToken string, initialLat, initialLng float64, logger *Logger) *DriverService {
	httpClient := NewHTTPClient(logger)
	wsClient := NewWebSocketClient(ctx, logger)

	return &DriverService{
		driverID:   driverID,
		jwtToken:   jwtToken,
		currentLat: initialLat,
		currentLng: initialLng,
		httpClient: httpClient,
		wsClient:   wsClient,
		logger:     logger,
		ctx:        ctx,
	}
}

func (d *DriverService) Authenticate() error {
	authMsg := websocketdto.AuthMessage{
		WebSocketMessage: websocketdto.WebSocketMessage{
			Type: websocketdto.MessageTypeAuth,
		},
		Token: d.jwtToken,
	}

	time.Sleep(DBWriteDelay)
	return d.wsClient.SendMessage(authMsg)
}

func (d *DriverService) SetOnline() error {
	location := Location{
		Latitude:  d.currentLat,
		Longitude: d.currentLng,
	}

	url := fmt.Sprintf(BaseURL+DriverOnlinePath, d.driverID)
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + d.jwtToken,
	}

	_, err := d.httpClient.DoRequest("POST", url, location, headers)
	if err != nil {
		return fmt.Errorf("setting driver online: %w", err)
	}

	d.logger.HTTP("Driver set online successfully")
	return nil
}

func (d *DriverService) HandleRideOffer(offer websocketdto.RideOfferMessage) error {
	d.logger.WebSocket("🚗 Received ride offer: %+v", offer)

	// Accept the ride offer
	resp := websocketdto.RideResponseMessage{
		WebSocketMessage: websocketdto.WebSocketMessage{
			Type: websocketdto.MessageTypeRideResponse,
		},
		OfferID:  offer.OfferID,
		RideID:   offer.RideID,
		Accepted: true,
		CurrentLocation: websocketdto.Location{
			Latitude:  d.currentLat,
			Longitude: d.currentLng,
		},
	}

	if err := d.wsClient.SendMessage(resp); err != nil {
		return fmt.Errorf("sending ride response: %w", err)
	}

	d.logger.WebSocket("✅ Accepted ride offer %s", offer.OfferID)

	// Start location updates for the ride
	time.Sleep(DBWriteDelay)
	go d.processRide(offer)

	return nil
}

func (d *DriverService) processRide(offer websocketdto.RideOfferMessage) {
	speed := 500.0 // meters per second

	// Go to pickup location
	d.navigateToLocation(offer.PickupLocation, speed, "pickup")

	// Start the ride
	if err := d.startRide(offer.RideID); err != nil {
		d.logger.Error("Failed to start ride: %v", err)
		return
	}

	// Go to destination
	d.navigateToLocation(offer.DestinationLocation, speed, "destination")

	// Complete the ride
	if err := d.completeRide(offer.RideID); err != nil {
		d.logger.Error("Failed to complete ride: %v", err)
		return
	}
}

func (d *DriverService) navigateToLocation(target websocketdto.Location, speed float64, locationType string) {
	current := websocketdto.Location{
		Latitude:  d.currentLat,
		Longitude: d.currentLng,
	}

	d.moveToTarget(current, target, speed)
	d.logger.Info("Arrived at %s location: lat=%.6f, lng=%.6f", locationType, d.currentLat, d.currentLng)

	time.Sleep(DBWriteDelay * 2)
}

func (d *DriverService) startRide(rideID string) error {
	startReq := StartRideRequest{
		RideID: rideID,
		DriverLocation: Location{
			Latitude:  d.currentLat,
			Longitude: d.currentLng,
		},
	}

	url := fmt.Sprintf(BaseURL+DriverStartPath, d.driverID)
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + d.jwtToken,
	}

	data, err := d.httpClient.DoRequest("POST", url, startReq, headers)
	if err != nil {
		return fmt.Errorf("starting ride: %w", err)
	}

	var response StartRideResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return fmt.Errorf("unmarshaling start ride response: %w", err)
	}

	d.logger.HTTP("Ride started: %s", response.Message)
	return nil
}

func (d *DriverService) completeRide(rideID string) error {
	completeReq := CompleteRideRequest{
		RideId: rideID,
		FinalLocation: Location{
			Latitude:  d.currentLat,
			Longitude: d.currentLng,
		},
		ActualDist: 5.5,
		ActualDur:  16,
	}

	url := fmt.Sprintf(BaseURL+DriverCompletePath, d.driverID)
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + d.jwtToken,
	}

	data, err := d.httpClient.DoRequest("POST", url, completeReq, headers)
	if err != nil {
		return fmt.Errorf("completing ride: %w", err)
	}

	var response CompleteRideResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return fmt.Errorf("unmarshaling complete ride response: %w", err)
	}

	d.logger.Info("🏁 Ride completed: ID=%s, Earnings=%.2f", response.RideID, response.DriverEarnings)
	return nil
}

func (d *DriverService) moveToTarget(current, target websocketdto.Location, speed float64) {
	stepDistance := speed * LocationUpdateInterval.Seconds()
	totalDistance := distance(current, target)

	if totalDistance < 1 {
		d.logger.WebSocket("✅ Already at target location")
		return
	}

	steps := int(totalDistance / stepDistance)
	if steps < 1 {
		steps = 1
	}

	dLat := (target.Latitude - current.Latitude) / float64(steps)
	dLng := (target.Longitude - current.Longitude) / float64(steps)

	d.logger.WebSocket("🚗 Moving to target: distance=%.2fm, steps=%d", totalDistance, steps)

	ticker := time.NewTicker(LocationUpdateInterval)
	defer ticker.Stop()

	for i := 0; i < steps; i++ {
		select {
		case <-ticker.C:
			d.currentLat += dLat
			d.currentLng += dLng

			locUpdate := websocketdto.LocationUpdateMessage{
				WebSocketMessage: websocketdto.WebSocketMessage{
					Type: websocketdto.MessageTypeLocationUpdate,
				},
				Latitude:  d.currentLat,
				Longitude: d.currentLng,
			}

			if err := d.wsClient.SendMessage(locUpdate); err != nil {
				d.logger.Error("Failed to send location update: %v", err)
				return
			}

			d.logger.WebSocket("📍 Location update (%d/%d): lat=%.6f, lng=%.6f",
				i+1, steps, d.currentLat, d.currentLng)

			time.Sleep(DBWriteDelay)

		case <-d.ctx.Done():
			d.logger.WebSocket("🛑 Movement stopped (context canceled)")
			return
		}
	}

	// Final position adjustment
	d.currentLat = target.Latitude
	d.currentLng = target.Longitude

	// Send final location update
	finalUpdate := websocketdto.LocationUpdateMessage{
		WebSocketMessage: websocketdto.WebSocketMessage{
			Type: websocketdto.MessageTypeLocationUpdate,
		},
		Latitude:       d.currentLat,
		Longitude:      d.currentLng,
		AccuracyMeters: 5.0,
		SpeedKmh:       45.0,
		HeadingDegrees: 90.0,
	}

	if err := d.wsClient.SendMessage(finalUpdate); err != nil {
		d.logger.Error("Failed to send final location update: %v", err)
	}

	time.Sleep(DBWriteDelay)
	d.logger.WebSocket("🏁 Arrived at target location (%.6f, %.6f)", d.currentLat, d.currentLng)
}

// distance calculates the haversine distance between two coordinates in meters
func distance(a, b websocketdto.Location) float64 {
	const R = 6371000 // Earth radius in meters
	dLat := (b.Latitude - a.Latitude) * math.Pi / 180
	dLng := (b.Longitude - a.Longitude) * math.Pi / 180
	lat1 := a.Latitude * math.Pi / 180
	lat2 := b.Latitude * math.Pi / 180

	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLng/2)*math.Sin(dLng/2)*math.Cos(lat1)*math.Cos(lat2)
	return 2 * R * math.Asin(math.Sqrt(h))
}
