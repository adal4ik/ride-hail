package services

import (
	"ride-hail/internal/driver-location-service/adapters/driven/db"
	ports "ride-hail/internal/driver-location-service/core/ports/driven"
	"ride-hail/internal/mylogger"
)

type Service struct {
	DriverService *DriverService
	AuthService   *AuthService
}

// Must properly implement Auth Service
func New(repositories *db.Repository, log mylogger.Logger, broker ports.IDriverBroker, secretKey string) *Service {
	return &Service{
		DriverService: NewDriverService(repositories.DriverRepository, log, broker),
		AuthService:   NewAuthService(secretKey),
	}
}
