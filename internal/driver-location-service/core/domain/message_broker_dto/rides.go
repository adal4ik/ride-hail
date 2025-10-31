package messagebrokerdto

type Location struct {
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	Address string  `json:"addres,omitempty"`
}

// Driver Match Request → ride_topic exchange → ride.request.{ride_type}
type Ride struct {
	RideID              string   `json:"ride_id"`
	RideNumber          string   `json:"ride_number"`
	PickupLocation      Location `json:"pickup_location"`
	DestinationLocation Location `json:"destination_location"`
	RideType            string   `json:"ride_type"`
	EstimatedFare       float64  `json:"estimated_fare"`
	MaxDistanceKm       float64  `json:"max_distance_km"`
	TimeoutSeconds      int      `json:"timeout_seconds"`
	CorrelationID       string   `json:"correlation_id"`
}

// Status Update → ride_topic exchange → ride.status.{status}
type RideStatus struct {
	RideId        string  `json:"ride_id"`
	Status        string  `json:"status"`
	Timestamp     string  `json:"timestamp"`
	Final_fare    float64 `json:"final_fare,omitempty"`
	CorrelationID string  `json:"correlation_id"`
}
