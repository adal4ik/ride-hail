package ports

import "ride-hail/internal/ride-service/core/domain/dto"

type IRidesService interface {
	CreateRide(dto.RidesRequestDto) (dto.RidesResponseDto, error)
}

type IPassengerService interface {
	FindPassenger(passengerId string) (bool, error) 
}