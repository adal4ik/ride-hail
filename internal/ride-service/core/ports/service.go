package ports

import "ride-hail/internal/ride-service/core/domain/dto"

type IRidesService interface {
	CreateRide(dto.RidesRequestDto) (dto.RidesResponseDto, error)
	// input: rideId, driverId, output: passengerId, error
	StatusMatch(string, string) (string, string, error)
}

type IPassengerService interface {
	FindPassenger(passengerId string) (bool, error) 
}