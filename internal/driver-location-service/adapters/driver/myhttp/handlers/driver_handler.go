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
	dh.log.Action("ws_conn").Info("start WS handshake")

	// 1) Достаём user_id и роль, которые положил middleware
	userID := r.Header.Get("X-UserId")
	role := r.Header.Get("X-Role")
	if userID == "" || role == "" {
		http.Error(w, "Unauthorized: empty identity", http.StatusUnauthorized)
		return
	}

	// 2) Сверяем, что водитель подключается своим ID
	driverID := r.PathValue("driver_id")
	if role != "DRIVER" || userID != driverID {
		http.Error(w, "Forbidden: driver mismatch", http.StatusForbidden)
		return
	}

	// 3) Дополнительно (по желанию): ограничить Origin
	// dh.upgrader.CheckOrigin = func(r *http.Request) bool {
	//     origin := r.Header.Get("Origin")
	//     return origin == "https://your-frontend.example.com"
	// }

	// 4) Апгрейд только после успешной аутентификации и авторизации
	conn, err := dh.upgrader.Upgrade(w, r, nil)
	if err != nil {
		dh.log.Action("ws_upgrade").Error("failed to upgrade", err)
		return
	}
	defer conn.Close()

	dh.log.Action("ws_conn").Info("WS established", "driver_id", driverID)

	// регистрируем каналы для этого драйвера (как у тебя было)
	dh.inMessages[driverID] = make(chan dto.DriverRideOffer, 100)
	dh.outMessages[driverID] = make(chan dto.DriverResponse, 100)

	// writer
	go func() {
		for {
			select {
			case rideOffer := <-dh.inMessages[driverID]:
				b, err := json.Marshal(rideOffer)
				if err != nil {
					log.Printf("marshal ride offer: %v", err)
					continue
				}
				if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
					log.Printf("write ride offer: %v", err)
					return
				}
			case <-r.Context().Done():
				return
			}
		}
	}()

	// reader (как у тебя было)
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("read msg: %v", err)
			break
		}
		var driverResponse dto.DriverResponse
		if err := json.Unmarshal(message, &driverResponse); err != nil {
			log.Printf("unmarshal driver response: %v", err)
			continue
		}
		dh.outMessages[driverID] <- driverResponse
		log.Printf("recv type=%v: %s", messageType, message)
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
