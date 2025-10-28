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

type DriverHandler struct {
	driverService driver.IDriverService
	log           mylogger.Logger
	upgrader      websocket.Upgrader
	inMessages    map[string]chan dto.DriverRideOffer
	outMessages   map[string]chan dto.DriverResponse
}

func NewDriverHandler(driverService driver.IDriverService, log mylogger.Logger, inMessages map[string]chan dto.DriverRideOffer, outMessages map[string]chan dto.DriverResponse) *DriverHandler {
	return &DriverHandler{
		driverService: driverService,
		log:           log,
		upgrader: websocket.Upgrader{
			CheckOrigin:     func(r *http.Request) bool { return true },
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		inMessages:  inMessages,
		outMessages: outMessages,
	}
}

func (dh *DriverHandler) HandleDriverConnection(w http.ResponseWriter, r *http.Request) {
	dh.log.Action("Handling driver WebSocket connection").Info("Starting WebSocket upgrade")
	conn, err := dh.upgrader.Upgrade(w, r, nil)
	if err != nil {
		dh.log.Action("WebSocket connection establishing").Error("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	dh.log.Action("WebSocket connection establishing").Info("WebSocket connection established successfully")
	driver_id := r.PathValue("driver_id")
	dh.inMessages[driver_id] = make(chan dto.DriverRideOffer, 100)
	dh.outMessages[driver_id] = make(chan dto.DriverResponse, 100)
	go func() {
		for {
			select {
			case rideOffer := <-dh.inMessages[driver_id]:
				offerBytes, err := json.Marshal(rideOffer)
				if err != nil {
					log.Printf("Error marshalling ride offer: %v", err)
					break
				}
				err = conn.WriteMessage(websocket.TextMessage, offerBytes)
				if err != nil {
					log.Printf("Error writing ride offer message: %v", err)
					break
				}
			default:
				continue
			}
		}
	}()
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}
		// IMPLEMENT LOGIC TO PROCESS INCOMING DRIVER RESPONSES BY TYPES
		var driverResponse dto.DriverResponse
		if err := json.Unmarshal(message, &driverResponse); err != nil {
			log.Printf("Error unmarshalling driver response: %v", err)
			continue
		}
		dh.outMessages[driver_id] <- driverResponse
		log.Printf("Received message type %v: %s", messageType, message)
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
