package dto

// API Transfer data

type RidesRequestDto struct {
	PassengerId string `json:"passenger_id"`

	PickUpLatitude  float64 `json:"pickup_latitude"`
	PickUpLongitude float64 `json:"pickup_longitude"`
	PickUpAddress   string  `json:"pickup_address"`

	DestinationLatitude  float64 `json:"destination_latitude"`
	DestinationLongitude float64 `json:"destination_longitude"`
	DestinationAddress   string  `json:"destination_address"`

	RideType string `json:"ride_type"`
}

type RidesResponseDto struct {
	RideId                   string  `json:"ride_id"`
	RideNumber               string  `json:"ride_number"`
	Status                   string  `json:"status"`
	EstimatedFare            float64 `json:"estimated_fare"`
	EstimatedDurationMinutes float64 `json:"estimated_duration_minutes"`
	EstimatedDistanceKm      float64 `json:"estimated_distance_km"`
}

type RideStatusUpdate struct {
	ClientId   string
	RideNumber string
	Status     string
}
