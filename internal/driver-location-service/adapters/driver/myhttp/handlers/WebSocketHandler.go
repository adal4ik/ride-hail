package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ride-hail/internal/driver-location-service/adapters/driven/ws"
	websocketdto "ride-hail/internal/driver-location-service/core/domain/websocket_dto"
	"ride-hail/internal/driver-location-service/core/ports/driver"
	"ride-hail/internal/mylogger"

	"github.com/gorilla/websocket"
)

type WebSocketHandler struct {
	wsManager *ws.WebSocketManager
	upgrader  websocket.Upgrader
	auth      driver.IAuthSerive
	log       mylogger.Logger
}

type AuthService interface {
	ValidateDriverToken(token string) (string, error)
}

func NewWebSocketHandler(wsManager *ws.WebSocketManager, auth driver.IAuthSerive, log mylogger.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		wsManager: wsManager,
		upgrader: websocket.Upgrader{
			CheckOrigin:     func(r *http.Request) bool { return true },
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		auth: auth,
		log:  log,
	}
}

func (h *WebSocketHandler) HandleDriverWebSocket(w http.ResponseWriter, r *http.Request) {
	log := h.log.Action("HandleDriverWebSocket")
	driverID := extractDriverID(r.URL.Path)
	if driverID == "" {
		log.Warn("Driver ID missing in URL")
		http.Error(w, "Driver ID required", http.StatusBadRequest)
		return
	}

	incoming := make(chan []byte, 100)
	outgoing := make(chan []byte, 100)

	if err := h.wsManager.RegisterDriver(r.Context(), driverID, incoming, outgoing); err != nil {
		log.Error("Failed to register driver:", err, driverID)
		http.Error(w, "Failed to register driver", http.StatusInternalServerError)
		return
	}
	defer h.wsManager.UnregisterDriver(r.Context(), driverID)
	log.Info("Driver registered:", driverID)
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	h.wsManager.SetConnection(driverID, conn)

	conn.SetPongHandler(func(string) error {
		h.wsManager.UpdatePing(driverID)
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	fmt.Println("Yes Im here")
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go h.handleIncomingMessages(ctx, driverID, conn, incoming)
	go h.handleOutgoingMessages(ctx, driverID, conn, outgoing)
	go h.handlePing(ctx, conn)
	log.Info("WebSocket connection established for driver:", driverID)
	<-ctx.Done()
}

func (h *WebSocketHandler) handleIncomingMessages(ctx context.Context, driverID string, conn *websocket.Conn, incoming chan<- []byte) {
	log := h.log.Action("handleIncomingMessages")
	defer close(incoming)

	authTimeout := time.NewTimer(5 * time.Second)
	authenticated := false

	go func() {
		<-authTimeout.C
		if !authenticated {
			log.Warn("Authentication timeout for driver:", driverID)
			conn.Close()
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					log.Info("WebSocket closed for driver:", driverID)
				} else {
					log.Error("Error reading WebSocket message for driver:", err, driverID)
				}
				return
			}

			if messageType != websocket.TextMessage {
				continue
			}

			if !authenticated {
				if authenticated = h.handleAuthentication(driverID, message); authenticated {
					h.wsManager.SetAuthenticated(driverID, true)
					h.sendAuthSuccess(conn)
					log.Info("Driver authenticated successfully:", driverID)
				} else {
					log.Warn("Authentication failed for driver:", driverID)
					h.sendAuthError(conn, "Authentication failed")
					conn.Close()
					return
				}
				continue
			}

			if err := h.validateMessage(message); err != nil {
				h.sendError(conn, "invalid_message", err.Error())
				log.Warn("Invalid message from driver:", driverID, err)
				continue
			}

			select {
			case incoming <- message:
				// loginc
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
				h.sendError(conn, "system_busy", "System is busy, please try again")
			}
		}
	}
}

func (h *WebSocketHandler) handleOutgoingMessages(ctx context.Context, driverID string, conn *websocket.Conn, outgoing <-chan []byte) {
	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-outgoing:
			if !ok {
				return
			}

			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		}
	}
}

func (h *WebSocketHandler) handlePing(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *WebSocketHandler) handleAuthentication(driverID string, message []byte) bool {
	log := h.log.Action("handleAuthentication")
	var baseMsg websocketdto.WebSocketMessage
	if err := json.Unmarshal(message, &baseMsg); err != nil {
		log.Error("Failed to unmarshal authentication message:", err, driverID)
		return false
	}

	if baseMsg.Type != websocketdto.MessageTypeAuth {
		return false
	}

	var authMsg websocketdto.AuthMessage
	if err := json.Unmarshal(message, &authMsg); err != nil {
		log.Error("Failed to unmarshal auth message:", err, driverID)
		return false
	}

	tokenDriverID, err := h.auth.ValidateDriverToken(authMsg.Token)
	if err != nil {
		log.Error("Token validation failed:", err, driverID)
		return false
	}
	if tokenDriverID != driverID {
		log.Warn("Driver ID mismatch in token:", driverID)
		return false
	}
	return true
}

func (h *WebSocketHandler) validateMessage(message []byte) error {
	var baseMsg websocketdto.WebSocketMessage
	if err := json.Unmarshal(message, &baseMsg); err != nil {
		return err
	}

	switch baseMsg.Type {
	case websocketdto.MessageTypeRideResponse:
		var rideResp websocketdto.RideResponseMessage
		if err := json.Unmarshal(message, &rideResp); err != nil {
			return err
		}
		return h.validateRideResponse(rideResp)

	case websocketdto.MessageTypeLocationUpdate:
		var locUpdate websocketdto.LocationUpdateMessage
		if err := json.Unmarshal(message, &locUpdate); err != nil {
			return err
		}
		return h.validateLocationUpdate(locUpdate)
	default:
		return fmt.Errorf("unknown message type: %s", baseMsg.Type)
	}
}

func (h *WebSocketHandler) validateRideResponse(resp websocketdto.RideResponseMessage) error {
	if resp.OfferID == "" {
		return fmt.Errorf("offer_id is required")
	}
	if resp.RideID == "" {
		return fmt.Errorf("ride_id is required")
	}
	return nil
}

func (h *WebSocketHandler) validateLocationUpdate(update websocketdto.LocationUpdateMessage) error {
	if update.Latitude < -90 || update.Latitude > 90 {
		return fmt.Errorf("invalid latitude: %f", update.Latitude)
	}
	if update.Longitude < -180 || update.Longitude > 180 {
		return fmt.Errorf("invalid longitude: %f", update.Longitude)
	}
	return nil
}

func (h *WebSocketHandler) sendAuthSuccess(conn *websocket.Conn) {
	successMsg := websocketdto.WebSocketMessage{
		Type: "auth_success",
	}
	messageBytes, _ := json.Marshal(successMsg)
	conn.WriteMessage(websocket.TextMessage, messageBytes)
}

func (h *WebSocketHandler) sendAuthError(conn *websocket.Conn, message string) {
	errorMsg := websocketdto.ErrorMessage{
		WebSocketMessage: websocketdto.WebSocketMessage{
			Type: websocketdto.MessageTypeError,
		},
		ErrorCode:    "auth_failed",
		ErrorMessage: message,
	}
	messageBytes, _ := json.Marshal(errorMsg)
	conn.WriteMessage(websocket.TextMessage, messageBytes)
}

func (h *WebSocketHandler) sendError(conn *websocket.Conn, code, message string) {
	errorMsg := websocketdto.ErrorMessage{
		WebSocketMessage: websocketdto.WebSocketMessage{
			Type: websocketdto.MessageTypeError,
		},
		ErrorCode:    code,
		ErrorMessage: message,
	}
	messageBytes, _ := json.Marshal(errorMsg)
	conn.WriteMessage(websocket.TextMessage, messageBytes)
}

func extractDriverID(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-1]
}

// Admin endpoint для проверки статуса соединений
func (h *WebSocketHandler) GetConnectionStatus(w http.ResponseWriter, r *http.Request) {
	driverID := r.URL.Query().Get("driver_id")
	if driverID != "" {
		status := h.wsManager.GetConnectionStatus(driverID)
		jsonResponse(w, http.StatusOK, status)
		return
	}

	// Возвращаем список всех подключенных драйверов
	connectedDrivers := h.wsManager.GetConnectedDrivers()
	response := map[string]interface{}{
		"connected_drivers": connectedDrivers,
		"total_connected":   len(connectedDrivers),
	}
	jsonResponse(w, http.StatusOK, response)
}
