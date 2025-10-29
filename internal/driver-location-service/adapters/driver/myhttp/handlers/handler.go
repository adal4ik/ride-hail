package handlers

import (
	"ride-hail/internal/driver-location-service/adapters/driven/ws"
	"ride-hail/internal/driver-location-service/core/services"
	"ride-hail/internal/mylogger"
)

type Handlers struct {
	DriverHandler    *DriverHandler
	WebSocketHandler *WebSocketHandler
}

func New(service *services.Service, log mylogger.Logger, wsManager *ws.WebSocketManager) *Handlers {
	return &Handlers{
		DriverHandler:    NewDriverHandler(service.DriverService, log),
		WebSocketHandler: NewWebSocketHandler(wsManager, service.AuthService, log),
	}
}
