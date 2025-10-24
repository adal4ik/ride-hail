package handle

import (
	"encoding/json"
	"net/http"

	"ride-hail/internal/mylogger"
	"ride-hail/internal/ride-service/core/domain/dto"
	"ride-hail/internal/ride-service/core/ports"
)

type RidesHandler struct {
	ridesService ports.IRidesService
	log          mylogger.Logger
}

func NewRidesHandler(rs ports.IRidesService, log mylogger.Logger) *RidesHandler {
	return &RidesHandler{
		ridesService: rs,
		log:          log,
	}
}

func (rh *RidesHandler) CreateRide() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := dto.RidesRequestDto{}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		res, err := rh.ridesService.CreateRide(req)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}

		jsonResponse(w, http.StatusAccepted, res)
	}
}
