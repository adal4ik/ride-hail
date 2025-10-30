package handlers

import (
	"context"
	"encoding/json"
	"errors"
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
	log := dh.log.Action("Handling driver WebSocket connection")
	log.Info("Starting websocket handshake")
	// ctx := context.Background()

	// Checking for Driver Existance
	driverID := r.PathValue("driver_id")
	// if ok, err := dh.driverService.CheckDriverById(ctx, driverID); err == nil && !ok {
	// 	log.Info("Driver not found")
	// 	jsonError(w, http.StatusForbidden, errors.New("The driver is not registered or not online"))
	// 	return
	// } else if err != nil {
	// 	log.Error("Failed to check the driver: ", err)
	// 	jsonError(w, http.StatusInternalServerError, err)
	// 	return
	// }

	// // Cheking for duplication connection
	// if _, ok := dh.inMessages[driverID]; ok {
	// 	log.Info("Driver already in connection")
	// 	jsonError(w, http.StatusBadRequest, errors.New("Driver already in webscoket connection"))
	// 	return
	// }

	// Upgrading connection
	conn, err := dh.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("failed to upgrade", err)
		return
	}
	defer conn.Close()
	log.Info("Websocket connection established", "driver_id", driverID)

	// Creating channels
	dh.inMessages[driverID] = make(chan dto.DriverRideOffer, 100)
	dh.outMessages[driverID] = make(chan dto.DriverResponse, 100)

	// Writer
	go func() {
		for {
			select {
			case rideOffer := <-dh.inMessages[driverID]:
				b, err := json.Marshal(rideOffer)
				if err != nil {
					log.Error("marshal ride offer: %v", err)
					continue
				}
				if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
					log.Error("write ride offer: %v", err)
					return
				}
			case <-r.Context().Done():
				return
			}
		}
	}()

	// Reader
	go func() {
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				log.Error("Failed to read message from driver: %v", err)
				break
			}
			var driverResponse dto.DriverResponse
			if err := json.Unmarshal(message, &driverResponse); err != nil {
				log.Error("unmarshal driver response: %v", err)
				continue
			}
			dh.outMessages[driverID] <- driverResponse
			log.Info("recv type=%v: %s", messageType, message)
		}
	}()

	select {
	case <-r.Context().Done():
		delete(dh.inMessages, driverID)
		delete(dh.inMessages, driverID)
	default:
	}
}

func (dh *DriverHandler) GoOnline(w http.ResponseWriter, r *http.Request) {
	log := dh.log.Action("GoOnline")
	ctx := r.Context()

	driverID := r.PathValue("driver_id")

	ok, err := dh.driverService.CheckDriverById(ctx, driverID)
	if err != nil {
		log.Error("check driver failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if !ok {
		log.Info("driver not found")
		http.Error(w, "Forbidden: driver mismatch", http.StatusForbidden)
		return
	}

	current, err := dh.driverService.CheckDriverStatus(ctx, driverID)
	if err != nil {
		log.Error("check status failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	req.Driver_id = driverID

	res, err := dh.driverService.GoOnline(ctx, req)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err)
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
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if !ok {
		log.Info("driver not found")
		http.Error(w, "Forbidden: driver mismatch", http.StatusForbidden)
		return
	}

	current, err := dh.driverService.CheckDriverStatus(ctx, driverID)
	if err != nil {
		log.Error("check status failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
		jsonError(w, http.StatusInternalServerError, err)
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
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	} else if !ok {
		http.Error(w, "Forbidden: driver mismatch", http.StatusForbidden)
		return
	}

	// ✅ единственная новая строка логики:
	if err := dh.driverService.RequireActiveRide(ctx, driverID); err != nil {
		if errors.Is(err, model.ErrNoActiveRide) {
			http.Error(w, "Bad Request: no active ride to update location", http.StatusBadRequest)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var req dto.NewLocation
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	res, err := dh.driverService.UpdateLocation(ctx, req, driverID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err)
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
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if !ok {
		log.Warn("driver not found", "driver_id", driverID)
		http.Error(w, "Forbidden: driver mismatch", http.StatusForbidden)
		return
	}

	// Декодим JSON
	var req dto.StartRide
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("decode body failed", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	log.Info("decoded request", "ride_id", req.Ride_id)

	// Запуск
	res, err := dh.driverService.StartRide(ctx, req)
	if err != nil {
		log.Error("start ride failed", err, "ride_id", req.Ride_id, "driver_id", driverID)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Info("ride started", "ride_id", res.Ride_id, "driver_id", driverID, "started_at", res.Started_at)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(res)
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
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	res, err := dh.driverService.CompleteRide(ctx, req)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err)
		return
	}

	jsonResponse(w, http.StatusAccepted, res)
}
