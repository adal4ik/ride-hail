package myhttp

import (
	"net/http"
	"ride-hail/internal/config"
	"ride-hail/internal/driver-location-service/adapters/driver/myhttp/handlers"
	"ride-hail/internal/driver-location-service/adapters/driver/myhttp/middleware"
)

func Router(handlers *handlers.Handlers, cfg *config.Config) http.Handler {
	mux := http.NewServeMux()
	// Initializing Handlers

	mdl := middleware.NewAuthMiddleware(cfg.App.PublicJwtSecret)
	handler := mdl.SessionHandler(mux)

	return handler
}
