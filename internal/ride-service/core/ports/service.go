package ports

import "ride-hail/internal/ride-service/core/domain/dto"

type IRidesService interface {
	CreateRide(dto.RidesRequestDto) (dto.RidesResponseDto, error)

	// input: rideId, driverId, output: passengerId, rideNumber, error
	// set to status match, and also send to the exchange
	SetStatusMatch(string, string) (string, string, error)
	FindPassengerByRideId(rideId string) (passengerId string, err error)
}

type IPassengerService interface {
	IsPassengerExists(passengerId string) (bool, error)
	// output passengerId
}
