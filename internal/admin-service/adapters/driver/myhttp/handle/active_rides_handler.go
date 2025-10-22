package handle

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"ride-hail/internal/admin-service/core/service"
	"ride-hail/internal/mylogger"
	"strconv"
	"time"
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
		userRole := r.Header.Get("X-Role")

		if userRole != "ADMIN" {
			jsonError(w, http.StatusForbidden, errors.New("only admins allowed to use this service"))
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), WaitTime*time.Second)
		defer cancel()

		// Get query parameters with defaults
		pageStr := r.URL.Query().Get("page")
		pageSizeStr := r.URL.Query().Get("page_size")

		// Set defaults if not provided
		if pageStr == "" {
			pageStr = "1"
		}
		if pageSizeStr == "" {
			pageSizeStr = "20"
		}

		// Convert to integers
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			http.Error(w, "Invalid page parameter", http.StatusBadRequest)
			return
		}

		pageSize, err := strconv.Atoi(pageSizeStr)
		if err != nil || pageSize < 1 || pageSize > 100 {
			http.Error(w, "Invalid page_size parameter", http.StatusBadRequest)
			return
		}

		activeRides, err := dh.activeDrivesService.GetActiveRides(ctx, page, pageSize)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, fmt.Errorf("failed to get active rides: %v", err))
			return
		}

		jsonResponse(w, http.StatusOK, activeRides)
	}
}
