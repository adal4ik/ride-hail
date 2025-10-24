package model

import "time"

type Rides struct {
	ID                    string // uuid
	CreatedAt             time.Time
	UpdateAt              time.Time
	RideNumber            string
	PassengerId           string // uuid
	DriverId              string // uuid
	VehicleType           string
	Status                string
	Priority              int
	RequestedAt           time.Time
	MatchedAt             time.Time
	ArrivedAt             time.Time
	StartedAt             time.Time
	CompletedAt           time.Time
	CancelledAt           time.Time
	CancellationReason    string
	EstimatedFare         float64
	FinalFare             float64
	PickupCoordinate      Coordinates 
	DestinationCoordinate Coordinates 
}

type Coordinates struct {
	Id              string // uuid
	CreatedAt       time.Time
	UpdatedAt       time.Time
	EntityId        string
	EntityType      string
	Address         string
	Latitude        float64
	Longitude       float64
	FareAmount      float64
	DistanceKm      float64
	DurationMinutes float64
	IsCurrent       bool
}
