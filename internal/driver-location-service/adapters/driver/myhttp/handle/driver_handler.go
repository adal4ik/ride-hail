package handle

import (
	"encoding/json"
	"net/http"
	"ride-hail/internal/driver-location-service/core/domain/dto"
	"ride-hail/internal/driver-location-service/core/ports"
	"ride-hail/internal/mylogger"
)

type DriverHandler struct {
	driverService ports.IDriverService
	log           mylogger.Logger
}

func NewDriverHandler(driverService ports.IDriverService, log mylogger.Logger) *DriverHandler {
	return &DriverHandler{
		driverService: driverService,
		log:           log,
	}
}

func (rh *DriverHandler) GoOnline() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := dto.RidesRequestDto{}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		res, err := rh.driverService.GoOnline(req)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}

		jsonResponse(w, http.StatusAccepted, res)
	}
}

func (rh *DriverHandler) GoOffline() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := dto.RidesRequestDto{}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		res, err := rh.driverService.GoOffline(req)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}

		jsonResponse(w, http.StatusAccepted, res)
	}
}

func (rh *DriverHandler) UpdateLocation() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := dto.RidesRequestDto{}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		res, err := rh.driverService.UpdateLocation(req)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}

		jsonResponse(w, http.StatusAccepted, res)
	}
}

func (rh *DriverHandler) StartRide() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := dto.RidesRequestDto{}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		res, err := rh.driverService.StartRide(req)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}

		jsonResponse(w, http.StatusAccepted, res)
	}
}

func (rh *DriverHandler) CompleteRide() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := dto.RidesRequestDto{}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		res, err := rh.driverService.CompleteRide(req)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err)
			return
		}

		jsonResponse(w, http.StatusAccepted, res)
	}
}
