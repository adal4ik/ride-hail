package handlers

import (
	"ride-hail/internal/driver-location-service/core/services"
	"ride-hail/internal/mylogger"
)

type Handlers struct {
	DriverHandler *DriverHandler
}

func New(service *services.Service, log mylogger.Logger) *Handlers {
	return &Handlers{
		DriverHandler: NewDriverHandler(service.DriverService, log),
	}
}
