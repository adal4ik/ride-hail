package model

// Online Mode
type DriverCoordinates struct {
	Driver_id string
	Latitude  float64
	Longitude float64
}

// Offline Mode

type DriverOfflineResponse struct {
	Session_id      string
	Session_summary Summary
}

type Summary struct {
	Duration_hours  float64
	Rides_completed int
	Earnings        float64
}
