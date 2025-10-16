package domain

import "time"

type Rides struct {
	ID                      string // uuid
	CreatedAt               time.Time
	UpdateAt                time.Time
	RideNumber              string
	PassengerId             string // uuid
	DriverId                string // uuid
	VehicleType             string
	Status                  string
	Priority                string
	RequestedAt             time.Time
	MatchedAt               time.Time
	ArrivedAt               time.Time
	StartedAt               time.Time
	CompletedAt             time.Time
	CancelledAt             time.Time
	CancellationReason      string
	EstimatedFare           float64
	FinalFare               float64
	PickupCoordinateId      string // uuid references coordinates(id)
	DestinationCoordinateId string // uuid references coordinates(id)
}
