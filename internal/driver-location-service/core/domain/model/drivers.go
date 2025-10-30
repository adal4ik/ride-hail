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

// START RIDE

type StartRide struct {
	Ride_id         string
	Driver_location DriverCoordinates
}

type StartRideResponse struct {
	Ride_id    string
	Status     string
	Started_at string
	Message    string
}

// New location for LOCATION UPDATE
type NewLocation struct {
	Latitude        float64
	Longitude       float64
	Accuracy_meters float64
	Speed_kmh       float64
	Heading_Degrees float64
}

type NewLocationResponse struct {
	Coordinate_id string
	Updated_at    string
}

// Complete Ride
type RideCompleteForm struct {
	Ride_id          string
	FinalLocation    Location
	ActualDistancekm float64
	ActualDurationm  float64
}

type RideCompleteResponse struct {
	Ride_id       string
	Status        string
	CompletedAt   string
	DriverEarning float64
	Message       string
}

// DriverInfo
type DriverInfo struct {
	DriverId  string
	Name      string
	Email     string
	Vehicle   []byte
	Rating    float64
	Latitude  float64
	Longitude float64
	Distance  float64
}

// RideDetails for WebSocket
type RideDetails struct {
	Ride_id        string
	PassengerName  string
	PassengerAttrs []byte
	PickupLocation Location
}

type DriverLocation struct {
	Driver_id string  `json:"driver_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address,omitempty"`
	Driver_id string  `json:"driver_id,omitempty"`
}

// type Location2 struct {
// 	Latitude  float64
// 	Longitude float64
// 	Address   string
// 	Notes     string
// }
