package dto

import "time"

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

// New location for LOCATION UPDATE
type NewLocation struct {
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	Accuracy_meters float64 `json:"accuracy_meters"`
	Speed_kmh       float64 `json:"speed_kmh"`
	Heading_Degrees float64 `json:"heading_degrees"`
}

type NewLocationResponse struct {
	Coordinate_id string `json:"coordinate_id"`
	Updated_at    string `json:"updated_at"`
}

// Complete Ride
type RideCompleteForm struct {
	Ride_id          string   `json:"ride_id"`
	FinalLocation    Location `json:"final_location"`
	ActualDistancekm float64  `json:"actual_distance_km"`
	ActualDurationm  float64  `json:"actual_duration_minutes"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type RideCompleteResponse struct {
	Ride_id       string  `json:"ride_id"`
	Status        string  `json:"status"`
	CompletedAt   string  `json:"completed_at"`
	DriverEarning float64 `json:"driver_earnings"`
	Message       string  `json:"message"`
}

// Ride Details
type RideDetails struct {
	Ride_id              string         `json:"ride_id"`
	Ride_number          string         `json:"ride_number"`
	Pickup_location      LocationDetail `json:"pickup_location"`
	Destination_location LocationDetail `json:"destination_location"`
	Ride_type            string         `json:"ride_type"`
	Estimated_fare       float64        `json:"estimated_fare"`
	Max_distance_km      float64        `json:"max_distance_km"`
	Timeout_seconds      int            `json:"timeout_seconds"`
	Correlation_id       string         `json:"correlation_id"`
}
type LocationDetail struct {
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	Address string  `json:"address"`
}

// Driver Response
type DriverResponse struct {
	Type             string               `json:"type"`
	Offer_id         string               `json:"offer_id"`
	Ride_id          string               `json:"ride_id"`
	Accepted         bool                 `json:"accepted"`
	Current_location DriverCoordinatesDTO `json:"current_location"`
}

// Driver Ride Offer
type DriverRideOffer struct {
	Type                 string         `json:"type"`
	Offer_id             string         `json:"offer_id"`
	Ride_id              string         `json:"ride_id"`
	Ride_number          string         `json:"ride_number"`
	Pickup_location      LocationDetail `json:"pickup_location"`
	Destination_location LocationDetail `json:"destination_location"`
	Estimated_fare       float64        `json:"estimated_fare"`
	Driver_earnings      float64        `json:"driver_earnings"`
	DistanceToPickUp     float64        `json:"distance_to_pickup_km"`
	EstimatedDuration    int            `json:"estimated_ride_duration_minutes"`
	ExpiredAt            string         `json:"expires_at"`
}

type LocationDetailsForOffer struct {
	Lat     float64 `json:"latitude"`
	Lng     float64 `json:"longitude"`
	Address string  `json:"address"`
	Notes   string  `json:"notes"`
}

// Ride Status Update
type RideStatusUpdate struct {
	RideID        string    `json:"ride_id"`                  // обязательный
	RideNumber    string    `json:"ride_number,omitempty"`    // опционально
	Status        string    `json:"status"`                   // см. константы выше
	DriverID      string    `json:"driver_id,omitempty"`      // если статус связан с водителем
	Reason        string    `json:"reason,omitempty"`         // причина отмены/нет водителя и т.п.
	Timestamp     time.Time `json:"timestamp"`                // когда изменился статус
	CorrelationID string    `json:"correlation_id,omitempty"` // если прокидываете трассировку
}

// Driver Info
type DriverInfo struct {
	DriverId  string
	Name      string `json:"name"`
	Email     string
	Vehicle   VehicleDetail `json:"vehicle"`
	Rating    float64       `json:"rating"`
	Latitude  float64
	Longitude float64
	Distance  float64
}
type VehicleDetail struct {
	Make  string `json:"make"`
	Model string `json:"model"`
	Color string `json:"color"`
	Plate string `json:"plate"`
}

// Driver Match Response

type DriverMatchResponse struct {
	Ride_id                   string                `json:"ride_id"`
	Driver_id                 string                `json:"driver_id"`
	Accepted                  bool                  `json:"accepted"`
	Estimated_arrival_minutes int                   `json:"estimated_arrival_minutes"`
	Driver_location           Location              `json:"driver_location"`
	Driver_info               DriverInfoForResponse `json:"driver_info"`
}

type DriverInfoForResponse struct {
	Name    string        `json:"name"`
	Vehicle VehicleDetail `json:"vehicle"`
	Rating  float64       `json:"rating"`
}

// Driver Message
type DriverMessage struct {
	DriverID string
	Message  []byte
}
