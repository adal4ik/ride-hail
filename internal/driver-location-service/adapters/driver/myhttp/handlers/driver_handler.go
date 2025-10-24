package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"ride-hail/internal/driver-location-service/core/domain/dto"
	"ride-hail/internal/driver-location-service/core/ports/driver"
	"ride-hail/internal/mylogger"
)

type DriverHandler struct {
	driverService driver.IDriverService
	log           mylogger.Logger
}

func NewDriverHandler(driverService driver.IDriverService, log mylogger.Logger) *DriverHandler {
	return &DriverHandler{
		driverService: driverService,
		log:           log,
	}
}

func (dh *DriverHandler) GoOnline(w http.ResponseWriter, r *http.Request) {
	req := dto.DriverCoordinatesDTO{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	req.Driver_id = r.PathValue("driver_id")
	ctx := context.Background()
	res, err := dh.driverService.GoOnline(ctx, req)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err)
		return
	}

	jsonResponse(w, http.StatusAccepted, res)
}

func (dh *DriverHandler) GoOffline(w http.ResponseWriter, r *http.Request) {
	driver_id := r.PathValue("driver_id")
	ctx := context.Background()
	res, err := dh.driverService.GoOffline(ctx, driver_id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err)
		return
	}
	jsonResponse(w, http.StatusAccepted, res)
}

func (dh *DriverHandler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	req := dto.NewLocation{}
	driver_id := r.PathValue("driver_id")
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	ctx := context.Background()
	res, err := dh.driverService.UpdateLocation(ctx, req, driver_id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err)
		return
	}

	jsonResponse(w, http.StatusAccepted, res)
}

func (dh *DriverHandler) StartRide(w http.ResponseWriter, r *http.Request) {
	req := dto.StartRide{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	ctx := context.Background()
	res, err := dh.driverService.StartRide(ctx, req)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err)
		return
	}

	jsonResponse(w, http.StatusAccepted, res)
}

func (dh *DriverHandler) CompleteRide(w http.ResponseWriter, r *http.Request) {
	req := dto.RideCompleteForm{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	ctx := context.Background()
	res, err := dh.driverService.CompleteRide(ctx, req)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err)
		return
	}

	jsonResponse(w, http.StatusAccepted, res)
}
