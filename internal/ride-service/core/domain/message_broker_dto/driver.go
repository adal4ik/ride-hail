package messagebrokerdto

// Driver Match Response ← driver_topic exchange ← driver.response.{ride_id}
type Vehicle struct {
	Make  string `json:"make"`
	Model string `json:"model"`
	Color string `json:"color"`
	Plate string `json:"plate"`
}

type DriverInfo struct {
	Name    string  `json:"name"`
	Rating  float64 `json:"rating"`
	Vehicle Vehicle `json:"vehicle"`
}

type RideAcceptance struct {
	RideID                  string     `json:"ride_id"`
	DriverID                string     `json:"driver_id"`
	Accepted                bool       `json:"accepted"`
	EstimatedArrivalMinutes int        `json:"estimated_arrival_minutes"`
	DriverLocation          Location   `json:"driver_location"`
	DriverInfo              DriverInfo `json:"driver_info"`
}

type DriverStatusUpdate struct {
	DriverId  string `json:"driver_id"`
	Status    string `json:"status"`
	RideId    string `json:"ride_id"`
	Timestamp string `json:"timestamp"`
}
