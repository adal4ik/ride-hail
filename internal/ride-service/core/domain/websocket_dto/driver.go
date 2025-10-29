package websocketdto

// To Passenger - Match Notification:
type Vehicle struct {
	Make  string `json:"make"`
	Model string `json:"model"`
	Color string `json:"color"`
	Plate string `json:"plate"`
}

type DriverInfo struct {
	DriverID string  `json:"driver_id"`
	Name     string  `json:"name"`
	Rating   float64 `json:"rating"`
	Vehicle  Vehicle `json:"vehicle"`
}

// To Passenger - Match Notification:
type RideStatusUpdateDto struct {
	RideID        string     `json:"ride_id"`
	RideNumber    string     `json:"ride_number"`
	Status        string     `json:"status"`
	DriverInfo    DriverInfo `json:"driver_info"`
	CorrelationID string     `json:"correlation_id"`
}
