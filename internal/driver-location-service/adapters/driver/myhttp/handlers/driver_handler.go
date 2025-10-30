package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"ride-hail/internal/driver-location-service/core/domain/dto"
	"ride-hail/internal/driver-location-service/core/domain/model"
	"ride-hail/internal/driver-location-service/core/ports/driver"
	"ride-hail/internal/mylogger"

	"github.com/gorilla/websocket"
)

const (
	DriverStatusOnline  = "AVAILABLE"
	DriverStatusOffline = "OFFLINE"
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
	log := dh.log.Action("GoOnline")
	ctx := r.Context()

	driverID := r.PathValue("driver_id")

	ok, err := dh.driverService.CheckDriverById(ctx, driverID)
	if err != nil {
		log.Error("check driver failed", err)
		JsonError(w, http.StatusInternalServerError, fmt.Errorf("Internal Server Error"))
		return
	}
	if !ok {
		log.Info("driver not found")
		JsonError(w, http.StatusForbidden, fmt.Errorf("Forbidden: driver mismatch"))
		return
	}

	current, err := dh.driverService.CheckDriverStatus(ctx, driverID)
	if err != nil {
		log.Error("check status failed", err)
		JsonError(w, http.StatusInternalServerError, fmt.Errorf("Internal Server Error"))
		return
	}
	if current == DriverStatusOnline {
		jsonResponse(w, http.StatusOK, map[string]any{
			"driver_id": driverID,
			"status":    current,
			"message":   "already online",
		})
		return
	}

	defer r.Body.Close()
	var req dto.DriverCoordinatesDTO
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
	log := dh.log.Action("GoOffline")
	ctx := r.Context()

	driverID := r.PathValue("driver_id")

	ok, err := dh.driverService.CheckDriverById(ctx, driverID)
	if err != nil {
		log.Error("check driver failed", err)
		JsonError(w, http.StatusInternalServerError, fmt.Errorf("Internal Server Error"))
		return
	}
	if !ok {
		log.Info("driver not found")
		JsonError(w, http.StatusForbidden, fmt.Errorf("Forbidden: driver mismatch"))
		return
	}

	current, err := dh.driverService.CheckDriverStatus(ctx, driverID)
	if err != nil {
		log.Error("check status failed", err)
		JsonError(w, http.StatusInternalServerError, fmt.Errorf("Internal Server Error"))
		return
	}
	if current == DriverStatusOffline {
		jsonResponse(w, http.StatusOK, map[string]any{
			"driver_id": driverID,
			"status":    current,
			"message":   "already offline",
		})
		return
	}

	res, err := dh.driverService.GoOffline(ctx, driverID)
	if err != nil {
		JsonError(w, http.StatusInternalServerError, err)
		return
	}
	jsonResponse(w, http.StatusAccepted, res)
}

func (dh *DriverHandler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	log := dh.log.Action("Update Location")
	ctx := r.Context()

	driverID := r.PathValue("driver_id")
	if ok, err := dh.driverService.CheckDriverById(ctx, driverID); err != nil {
		log.Error("Failed to check the driver: ", err)
		JsonError(w, http.StatusInternalServerError, fmt.Errorf("Internal Server Error"))

		return
	} else if !ok {
		JsonError(w, http.StatusForbidden, fmt.Errorf("Forbidden: driver mismatch"))
		return
	}

	// ✅ единственная новая строка логики:
	if err := dh.driverService.RequireActiveRide(ctx, driverID); err != nil {
		if errors.Is(err, model.ErrNoActiveRide) {
			JsonError(w, http.StatusBadRequest, fmt.Errorf("Bad Request: no active ride to update location"))
			return
		}
		JsonError(w, http.StatusInternalServerError, fmt.Errorf("Internal Server Error"))
		return
	}

	var req dto.NewLocation
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JsonError(w, http.StatusBadRequest, err)
		return
	}

	res, err := dh.driverService.UpdateLocation(ctx, req, driverID)
	if err != nil {
		JsonError(w, http.StatusInternalServerError, err)
		return
	}
	jsonResponse(w, http.StatusAccepted, res)
}

func (dh *DriverHandler) StartRide(w http.ResponseWriter, r *http.Request) {
	log := dh.log.Action("driver.start_ride")
	ctx := context.Background()

	driverID := r.PathValue("driver_id")
	log.Info("request received", "driver_id", driverID)

	// Проверяем, существует ли водитель
	ok, err := dh.driverService.CheckDriverById(ctx, driverID)
	if err != nil {
		log.Error("check driver failed", err, "driver_id", driverID)
		JsonError(w, http.StatusInternalServerError, fmt.Errorf("Internal Server Error"))
		return
	}
	if !ok {
		log.Warn("driver not found", "driver_id", driverID)
		JsonError(w, http.StatusForbidden, fmt.Errorf("Forbidden: driver mismatch"))
		return
	}

	// Декодим JSON
	var req dto.StartRide
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JsonError(w, http.StatusBadRequest, err)
		return
	}
	log.Info("decoded request", "ride_id", req.Ride_id)

	// Запуск
	req.Driver_location.Driver_id = driverID
	res, err := dh.driverService.StartRide(ctx, req)
	if err != nil {
		JsonError(w, http.StatusInternalServerError, err)
		log.Error("start ride failed", err, "ride_id", req.Ride_id, "driver_id", driverID)
		return
	}

	log.Info("ride started", "ride_id", res.Ride_id, "driver_id", driverID, "started_at", res.Started_at)
	jsonResponse(w, http.StatusAccepted, res)
}

func (dh *DriverHandler) CompleteRide(w http.ResponseWriter, r *http.Request) {
	log := dh.log.Action("Go Online")
	ctx := context.Background()

	// Checking Driver For Existance
	driverID := r.PathValue("driver_id")
	if ok, err := dh.driverService.CheckDriverById(ctx, driverID); err == nil && !ok {
		log.Info("Driver not found")
		JsonError(w, http.StatusForbidden, fmt.Errorf("Forbidden: driver mismatch"))
		return
	} else if err != nil {
		log.Error("Failed to check the driver: ", err)
		JsonError(w, http.StatusForbidden, fmt.Errorf("Forbidden: driver mismatch"))
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
