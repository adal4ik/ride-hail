package dto

// ONLINE MODE
type DriverCoordinatesDTO struct {
	Driver_id string  `json:"driver_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type DriverOnlineResponse struct {
	Status     string `json:"status"`
	Session_id string `json:"session_id"`
	Message    string `json:"message"`
}

// OFFLINE MODE

type DriverOfflineRespones struct {
	Status          string  `json:"status"`
	Session_id      string  `json:"session_id"`
	Session_summary Summary `json:"session_summary"`
	Message         string  `json:"message"`
}

type Summary struct {
	Duration_hours  float64 `json:"duration_hours"`
	Rides_completed int     `json:"rides_completed"`
	Earnings        float64 `json:"earnings"`
}

// START RIDE
type StartRide struct {
	Ride_id         string               `json:"ride_id"`
	Driver_location DriverCoordinatesDTO `json:"driver_location"`
}

type StartRideResponse struct {
	Ride_id    string `json:"ride_id"`
	Status     string `json:"status"`
	Started_at string `json:"started_at"`
	Message    string `json:"message"`
}
