package websocketdto

import "time"

// WebSocket message types
const (
	MessageTypeAuth           = "auth"
	MessageTypeRideOffer      = "ride_offer"
	MessageTypeRideResponse   = "ride_response"
	MessageTypeLocationUpdate = "location_update"
	MessageTypeRideDetails    = "ride_details"
	MessageTypePing           = "ping"
	MessageTypePong           = "pong"
	MessageTypeError          = "error"
)

// Base message structure
type WebSocketMessage struct {
	Type string `json:"type"`
}

// Authentication
type AuthMessage struct {
	WebSocketMessage
	Token string `json:"token"`
}

// Ride offer to driver
type RideOfferMessage struct {
	WebSocketMessage
	OfferID                      string    `json:"offer_id"`
	RideID                       string    `json:"ride_id"`
	RideNumber                   string    `json:"ride_number"`
	PickupLocation               Location  `json:"pickup_location"`
	DestinationLocation          Location  `json:"destination_location"`
	EstimatedFare                float64   `json:"estimated_fare"`
	DriverEarnings               float64   `json:"driver_earnings"`
	DistanceToPickupKm           float64   `json:"distance_to_pickup_km"`
	EstimatedRideDurationMinutes int       `json:"estimated_ride_duration_minutes"`
	ExpiresAt                    time.Time `json:"expires_at"`
}

// Driver response to ride offer
type RideResponseMessage struct {
	WebSocketMessage
	OfferID         string   `json:"offer_id"`
	RideID          string   `json:"ride_id"`
	Accepted        bool     `json:"accepted"`
	CurrentLocation Location `json:"current_location,omitempty"`
}

// Location update from driver
type LocationUpdateMessage struct {
	WebSocketMessage
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	AccuracyMeters float64 `json:"accuracy_meters"`
	SpeedKmh       float64 `json:"speed_kmh"`
	HeadingDegrees float64 `json:"heading_degrees"`
}

// Ride details after acceptance
type RideDetailsMessage struct {
	WebSocketMessage
	RideID         string   `json:"ride_id"`
	PassengerName  string   `json:"passenger_name"`
	PassengerPhone string   `json:"passenger_phone"`
	PickupLocation Location `json:"pickup_location"`
}

// Location structure
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address,omitempty"`
	Notes     string  `json:"notes,omitempty"`
}

// Error message
type ErrorMessage struct {
	WebSocketMessage
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

// Connection status
type ConnectionStatus struct {
	DriverID  string    `json:"driver_id"`
	Connected bool      `json:"connected"`
	LastPing  time.Time `json:"last_ping,omitempty"`
	SessionID string    `json:"session_id,omitempty"`
}
