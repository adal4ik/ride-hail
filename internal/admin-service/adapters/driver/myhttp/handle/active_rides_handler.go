package handle

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"ride-hail/internal/admin-service/core/service"
	"ride-hail/internal/mylogger"
)

type ActiveDrivesHandler struct {
	activeDrivesService *service.ActiveDrivesService
	mylog               mylogger.Logger
}

func NewActiveDrivesHandler(mylog mylogger.Logger, activeDrivesService *service.ActiveDrivesService) *ActiveDrivesHandler {
	return &ActiveDrivesHandler{
		activeDrivesService: activeDrivesService,
		mylog:               mylog,
	}
}

func (dh *ActiveDrivesHandler) GetActiveRides() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), WaitTime*time.Second)
		defer cancel()

		activeRides, err := dh.activeDrivesService.GetActiveRides(ctx)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, fmt.Errorf("failed to get active rides: %v", err))
			return
		}

		jsonResponse(w, http.StatusOK, activeRides)
	}
}
