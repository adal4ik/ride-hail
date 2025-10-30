package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"ride-hail/internal/driver-location-service/adapters/driven/ws"
	"ride-hail/internal/driver-location-service/core/domain/dto"
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
	driverID := r.PathValue("driver_id")
	if driverID == "" {
		log.Warn("Driver ID missing in URL")
		http.Error(w, "Driver ID required", http.StatusBadRequest)
		return
	}

	fromDriver := make(chan []byte, 100)
	toDriver := make(chan []byte, 100)
	if err := h.wsManager.RegisterDriver(r.Context(), driverID, fromDriver, toDriver); err != nil {
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
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go h.handleIncomingMessages(ctx, driverID, conn, fromDriver)
	go h.handleOutgoingMessages(ctx, driverID, conn, toDriver)
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
			var userMessageType string
			if userMessageType, err = h.validateMessage(message); err != nil {
				h.sendError(conn, "invalid_message", err.Error())
				log.Warn("Invalid message from driver:", driverID, err)
				continue
			}

			switch userMessageType {
			case websocketdto.MessageTypeRideResponse:
				log.Info("Received ride response from driver:", driverID)
				incoming <- message
			case websocketdto.MessageTypeLocationUpdate:
				log.Info("Received location update:", driverID)
				var driverMessage dto.DriverMessage
				driverMessage.DriverID = driverID
				driverMessage.Message = message
				h.wsManager.FanIn <- driverMessage
			default:
				log.Warn("Unhandled message type from driver:", driverID, messageType)
			}

		}
	}
}

func (h *WebSocketHandler) handleOutgoingMessages(ctx context.Context, driverID string, conn *websocket.Conn, outgoing <-chan []byte) {
	log := h.log.Action("handleOutgoingMessages")
	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-outgoing:
			log.Info("Sending message to driver:", driverID)
			if !ok {
				return
			}

			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Error("Error sending message to driver:", err, driverID)
				return
			}
			log.Info("Message sent to driver successfully:", driverID)
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

func (h *WebSocketHandler) validateMessage(message []byte) (string, error) {
	var baseMsg websocketdto.WebSocketMessage
	if err := json.Unmarshal(message, &baseMsg); err != nil {
		return "", err
	}

	switch baseMsg.Type {
	case websocketdto.MessageTypeRideResponse:
		var rideResp websocketdto.RideResponseMessage
		if err := json.Unmarshal(message, &rideResp); err != nil {
			return "", err
		}
		return baseMsg.Type, h.validateRideResponse(rideResp)

	case websocketdto.MessageTypeLocationUpdate:
		var locUpdate websocketdto.LocationUpdateMessage
		if err := json.Unmarshal(message, &locUpdate); err != nil {
			return "", err
		}
		return baseMsg.Type, h.validateLocationUpdate(locUpdate)
	default:
		return "", fmt.Errorf("unknown message type: %s", baseMsg.Type)
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
