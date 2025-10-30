package messagebrokerdto

import "time"

// Location Update ← location_fanout exchange
type LocationUpdate struct {
	DriverID       string   `json:"driver_id"`
	RideID         string   `json:"ride_id"`
	Location       Location `json:"location"`
	SpeedKmh       float64  `json:"speed_kmh"`
	HeadingDegrees float64  `json:"heading_degrees"`
	Timestamp      string   `json:"timestamp"`
}

type DriverLocationUpdate struct {
	Type               string    `json:"type"`
	RideID             string    `json:"ride_id"`
	DriverLocation     Location  `json:"driver_location"`
	EstimatedArrival   time.Time `json:"estimated_arrival"`
	DistanceToPickupKm float64   `json:"distance_to_pickup_km"`
}
