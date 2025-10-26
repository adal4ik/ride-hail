package handlers

import (
	"ride-hail/internal/driver-location-service/core/domain/dto"
	"ride-hail/internal/driver-location-service/core/services"
	"ride-hail/internal/mylogger"
)

type Handlers struct {
	DriverHandler *DriverHandler
}

func New(service *services.Service, log mylogger.Logger, inMessages map[string]chan dto.DriverRideOffer, outMessages map[string]chan dto.DriverResponse) *Handlers {
	return &Handlers{
		DriverHandler: NewDriverHandler(service.DriverService, log, inMessages, outMessages),
	}
}
