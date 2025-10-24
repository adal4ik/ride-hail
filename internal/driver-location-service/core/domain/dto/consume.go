package dto

type RideRequest struct {
	RideID          string   `json:"ride_id"`
	PassengerID     string   `json:"passenger_id"`
	RideType        string   `json:"ride_type"`
	Pickup          GeoPoint `json:"pickup"`
	Destination     GeoPoint `json:"destination"`
	MatchTimeoutSec int      `json:"match_timeout_sec"`
	EstimatedFare   float64  `json:"estimated_fare"`
}

type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type DriverAccepted struct {
	RideID   string `json:"ride_id"`
	DriverID string `json:"driver_id"`
	ETA      int    `json:"eta_sec"`
}

type RideNoDriver struct {
	RideID string `json:"ride_id"`
	Reason string `json:"reason"`
}
