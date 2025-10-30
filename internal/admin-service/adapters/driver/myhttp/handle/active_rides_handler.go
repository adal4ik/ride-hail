package handle

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
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

		// Get query parameters with defaults
		pageStr := r.PathValue("page")
		pageSizeStr := r.PathValue("page_size")

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
			JsonError(w, http.StatusInternalServerError, fmt.Errorf("failed to get active rides: %v", err))
			return
		}

		jsonResponse(w, http.StatusOK, activeRides)
	}
}
