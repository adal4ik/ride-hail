package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"ride-hail/internal/driver-location-service/adapters/driven/ws"

	"github.com/gorilla/websocket"
)

type WebSocketHandler struct {
	wsManager *ws.WebSocketManager
	upgrader  websocket.Upgrader
	auth      AuthService
}

type AuthService interface {
	ValidateDriverToken(token string) (string, error)
}

func NewWebSocketHandler(wsManager *ws.WebSocketManager, auth AuthService) *WebSocketHandler {
	return &WebSocketHandler{
		wsManager: wsManager,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		auth: auth,
	}
}

func (h *WebSocketHandler) HandlerDriverWebsocket(w http.ResponseWriter, r *http.Request) {
	driverID := r.PathValue("driver_id")

	incoming := make(chan []byte, 100)
	outgoing := make(chan []byte, 100)

	if err := h.wsManager.RegisterDriver(r.Context(), driverID, incoming, outgoing); err != nil {
		jsonError(w, http.StatusInternalServerError, errors.New("Failed to register driver"))
		return
	}
	defer h.wsManager.UnregisterDriver(r.Context(), driverID)

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	h.wsManager.SetConnection(driverID, conn)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// go h.handleIncomingMessages(ctx, driverID, conn, incoming)
	// go h.handleOutgoingMessages(ctx, driverID, conn, outgoing)
	// go h.handlePingPong(ctx, conn)

	<-ctx.Done()
}

func (h *WebSocketHandler) handleIncomingMessages(ctx context.Context, driverID string, conn *websocket.Conn, incoming chan<- []byte) {
	defer close(incoming)

	authTimeout := time.After(5 * time.Second)
	authenticated := false

	for {
		select {
		case <-ctx.Done():
			return
		default:
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if messageType != websocket.TextMessage {
				continue
			}

			if !authenticated {
				if h.handleAuthentication(driverID, message, authTimeout) {
					authenticated = true
					h.wsManager.SetAuthenticated(driverID, true)
				} else {
					conn.Close()
					return
				}
				continue
			}

			select {
			case incoming <- message:
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
				// drop message if not read in time
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

func (h *WebSocketHandler) handlePingPong(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

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

func (h *WebSocketHandler) handleAuthentication(driverID string, message []byte, authTimeout <-chan time.Time) bool {
	select {
	case <-authTimeout:
		return false
	default:
		var authMsg struct {
			Token string `json:"token"`
		}
		if err := json.Unmarshal(message, &authMsg); err != nil {
			return false
		}
		validatedDriverID, err := h.auth.ValidateDriverToken(authMsg.Token)
		if err != nil || validatedDriverID != driverID {
			return false
		}
		response := struct {
			Success bool   `json:"success"`
			Msg     string `json:"message"`
		}{
			Success: true,
			Msg:     "Authentication successful",
		}
		responseData, _ := json.Marshal(response)
		return h.wsManager.SendToDriver(context.Background(), driverID, responseData) == nil
	}
}
