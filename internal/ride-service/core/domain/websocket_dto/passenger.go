package websocketdto

import "time"

type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}
// To Passenger - Location Update:
type DriverLocationUpdate struct {
	Type               string    `json:"type"`
	RideID             string    `json:"ride_id"`
	DriverLocation     Location  `json:"driver_location"`
	EstimatedArrival   time.Time `json:"estimated_arrival"`
	DistanceToPickupKm float64   `json:"distance_to_pickup_km"`
}

// To Passenger - Status Updates:
type RideStatusUpdate struct {
	Type    string `json:"type"`
	RideID  string `json:"ride_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}
