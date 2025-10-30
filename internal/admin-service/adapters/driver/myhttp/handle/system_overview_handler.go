package handle

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"ride-hail/internal/admin-service/core/service"
	"ride-hail/internal/mylogger"
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
		ctx, cancel := context.WithTimeout(context.Background(), WaitTime*time.Second)
		defer cancel()

		systemOverview, err := dh.systemOverviewService.GetSystemOverview(ctx)
		if err != nil {
			JsonError(w, http.StatusBadRequest, fmt.Errorf("failed to get system overview: %v", err))
			return
		}

		jsonResponse(w, http.StatusOK, systemOverview)
	}
}
