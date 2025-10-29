package ports

import "ride-hail/internal/ride-service/core/domain/dto"

type IRidesService interface {
	CreateRide(dto.RidesRequestDto) (dto.RidesResponseDto, error)

	// input: rideId, driverId, output: passengerId, rideNumber, error
	// set to status match, and also send to the exchange
	SetStatusMatch(string, string) (passengerId string, rideNumber string, err error)
	EstimateDistance(rideId string, longitude, latitude, speed float64) (passengerId, estimatedTime string, distance float64, err error)
	CancelEveryPossibleRides() error
}

type IPassengerService interface {
	IsPassengerExists(passengerId string) (bool, error)
	// output passengerId
}
