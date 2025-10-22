package handle

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"ride-hail/internal/admin-service/core/service"
	"ride-hail/internal/mylogger"
	"time"
)

type SystemOverviewHandler struct {
	systemOverviewService *service.SystemOverviewService
	mylog                 mylogger.Logger
}

func NewSystemOverviewHandler(mylog mylogger.Logger, systemOverviewService *service.SystemOverviewService) *SystemOverviewHandler {
	return &SystemOverviewHandler{
		systemOverviewService: systemOverviewService,
		mylog:                 mylog,
	}
}

func (dh *SystemOverviewHandler) GetSystemOverview() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userRole := r.Header.Get("X-Role")
		expirationDate, err := time.Parse("2006-01-02T15:04:05.000Z", r.Header.Get("X-Exp"))
		if err != nil {
			jsonError(w, http.StatusInternalServerError, errors.New("failed to parse expiration date"))
			return
		}

		// Convert both to UTC for consistent comparison
		now := time.Now().UTC()
		expirationDate = expirationDate.UTC()

		if expirationDate.Before(now) {
			// Token has expired
			jsonError(w, http.StatusUnauthorized, errors.New("token expired"))
			return
		}

		if userRole != "ADMIN" {
			jsonError(w, http.StatusForbidden, errors.New("only admins allowed to use this service"))
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), WaitTime*time.Second)
		defer cancel()

		systemOverview, err := dh.systemOverviewService.GetSystemOverview(ctx)
		if err != nil {
			jsonError(w, http.StatusBadRequest, fmt.Errorf("failed to get system overview: %v", err))
			return
		}

		jsonResponse(w, http.StatusOK, systemOverview)
	}
}
