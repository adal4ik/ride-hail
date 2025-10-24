package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"ride-hail/internal/driver-location-service/core/domain/dto"
	"ride-hail/internal/driver-location-service/core/ports/driver"
	"ride-hail/internal/mylogger"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

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

func (h *DriverHandler) HandleDriverConnection(w http.ResponseWriter, r *http.Request) {
	h.log.Action("Handling driver WebSocket connection").Info("Starting WebSocket upgrade")
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	log.Println("WebSocket connection established")

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}
		log.Printf("Received message: %s", message)

		err = conn.WriteMessage(messageType, message)
		if err != nil {
			log.Printf("Error writing message: %v", err)
			break
		}
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
