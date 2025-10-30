package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"ride-hail/internal/driver-location-service/core/domain/dto"
	"ride-hail/internal/driver-location-service/core/ports/driver"
	"ride-hail/internal/mylogger"

	"github.com/gorilla/websocket"
)

type DriverHandler struct {
	driverService driver.IDriverService
	log           mylogger.Logger
	upgrader      websocket.Upgrader
}

func NewDriverHandler(driverService driver.IDriverService, log mylogger.Logger) *DriverHandler {
	return &DriverHandler{
		driverService: driverService,
		log:           log,
		upgrader: websocket.Upgrader{
			CheckOrigin:     func(r *http.Request) bool { return true },
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

func (dh *DriverHandler) GoOnline(w http.ResponseWriter, r *http.Request) {
	// log := dh.log.Action("Go Online")
	ctx := context.Background()

	// Checking Driver For Existance
	driverID := r.PathValue("driver_id")
	// if ok, err := dh.driverService.CheckDriverById(ctx, driverID); err == nil && !ok {
	// 	log.Info("Driver not found")
	// 	http.Error(w, "Forbidden: driver mismatch", http.StatusForbidden)
	// 	return
	// } else if err != nil {
	// 	log.Error("Failed to check the driver: ", err)
	// 	http.Error(w, "Forbidden: driver mismatch", http.StatusForbidden)
	// 	return
	// }

	// Preparing
	req := dto.DriverCoordinatesDTO{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JsonError(w, http.StatusBadRequest, err)
		return
	}
	req.Driver_id = driverID
	res, err := dh.driverService.GoOnline(ctx, req)
	if err != nil {
		JsonError(w, http.StatusInternalServerError, err)
		return
	}

	jsonResponse(w, http.StatusAccepted, res)
}

func (dh *DriverHandler) GoOffline(w http.ResponseWriter, r *http.Request) {
	log := dh.log.Action("Go Online")
	ctx := context.Background()

	// Checking Driver For Existance
	driverID := r.PathValue("driver_id")
	if ok, err := dh.driverService.CheckDriverById(ctx, driverID); err == nil && !ok {
		log.Info("Driver not found")
		http.Error(w, "Forbidden: driver mismatch", http.StatusForbidden)
		return
	} else if err != nil {
		log.Error("Failed to check the driver: ", err)
		http.Error(w, "Forbidden: driver mismatch", http.StatusForbidden)
		return
	}

	driver_id := driverID
	res, err := dh.driverService.GoOffline(ctx, driver_id)
	if err != nil {
		JsonError(w, http.StatusInternalServerError, err)
		return
	}
	jsonResponse(w, http.StatusAccepted, res)
}

func (dh *DriverHandler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	log := dh.log.Action("Go Online")
	ctx := context.Background()

	// Checking Driver For Existance
	driverID := r.PathValue("driver_id")
	if ok, err := dh.driverService.CheckDriverById(ctx, driverID); err == nil && !ok {
		log.Info("Driver not found")
		http.Error(w, "Forbidden: driver mismatch", http.StatusForbidden)
		return
	} else if err != nil {
		log.Error("Failed to check the driver: ", err)
		http.Error(w, "Forbidden: driver mismatch", http.StatusForbidden)
		return
	}

	req := dto.NewLocation{}
	driver_id := r.PathValue("driver_id")
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JsonError(w, http.StatusBadRequest, err)
		return
	}
	res, err := dh.driverService.UpdateLocation(ctx, req, driver_id)
	if err != nil {
		JsonError(w, http.StatusInternalServerError, err)
		return
	}

	jsonResponse(w, http.StatusAccepted, res)
}

func (dh *DriverHandler) StartRide(w http.ResponseWriter, r *http.Request) {
	log := dh.log.Action("Go Online")
	ctx := context.Background()

	// Checking Driver For Existance
	driverID := r.PathValue("driver_id")
	if ok, err := dh.driverService.CheckDriverById(ctx, driverID); err == nil && !ok {
		log.Info("Driver not found")
		http.Error(w, "Forbidden: driver mismatch", http.StatusForbidden)
		return
	} else if err != nil {
		log.Error("Failed to check the driver: ", err)
		http.Error(w, "Forbidden: driver mismatch", http.StatusForbidden)
		return
	}

	req := dto.StartRide{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JsonError(w, http.StatusBadRequest, err)
		return
	}
	res, err := dh.driverService.StartRide(ctx, req)
	if err != nil {
		JsonError(w, http.StatusInternalServerError, err)
		return
	}

	jsonResponse(w, http.StatusAccepted, res)
}

func (dh *DriverHandler) CompleteRide(w http.ResponseWriter, r *http.Request) {
	log := dh.log.Action("Go Online")
	ctx := context.Background()

	// Checking Driver For Existance
	driverID := r.PathValue("driver_id")
	if ok, err := dh.driverService.CheckDriverById(ctx, driverID); err == nil && !ok {
		log.Info("Driver not found")
		http.Error(w, "Forbidden: driver mismatch", http.StatusForbidden)
		return
	} else if err != nil {
		log.Error("Failed to check the driver: ", err)
		http.Error(w, "Forbidden: driver mismatch", http.StatusForbidden)
		return
	}

	req := dto.RideCompleteForm{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JsonError(w, http.StatusBadRequest, err)
		return
	}
	res, err := dh.driverService.CompleteRide(ctx, req)
	if err != nil {
		JsonError(w, http.StatusInternalServerError, err)
		return
	}

	jsonResponse(w, http.StatusAccepted, res)
}
