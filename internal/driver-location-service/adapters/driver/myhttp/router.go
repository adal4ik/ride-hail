package myhttp

import (
	"net/http"

	"ride-hail/internal/config"
	"ride-hail/internal/driver-location-service/adapters/driver/myhttp/handlers"
	"ride-hail/internal/driver-location-service/adapters/driver/myhttp/middleware"
)

func Router(handlers *handlers.Handlers, cfg *config.Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/drivers/{driver_id}/online", handlers.DriverHandler.GoOnline)
	mux.HandleFunc("/drivers/{driver_id}/offline", handlers.DriverHandler.GoOffline)
	mux.HandleFunc("/drivers/{driver_id}/location", handlers.DriverHandler.UpdateLocation)
	mux.HandleFunc("/drivers/{driver_id}/start", handlers.DriverHandler.StartRide)
	mux.HandleFunc("/drivers/{driver_id}/complete", handlers.DriverHandler.CompleteRide)
	mdl := middleware.NewAuthMiddleware(cfg.App.PublicJwtSecret)
	handler := mdl.SessionHandler(mux)

	return handler
}
