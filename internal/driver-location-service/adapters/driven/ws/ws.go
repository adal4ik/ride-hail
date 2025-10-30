package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"ride-hail/internal/driver-location-service/core/domain/dto"
	websocketdto "ride-hail/internal/driver-location-service/core/domain/websocket_dto"

	"github.com/gorilla/websocket"
)

type WebSocketManager struct {
	connections map[string]*DriverConnection
	FanIn       chan dto.DriverMessage
	mu          sync.RWMutex
}

type DriverConnection struct {
	DriverID   string
	Conn       *websocket.Conn
	fromDriver <-chan []byte // Сообщения ОТ драйвера К дистрибьютору
	toDriver   chan<- []byte // Сообщения ОТ дистрибьютора К драйверу
	Auth       bool
	LastPing   time.Time
	SessionID  string
	mu         sync.Mutex
}

func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		connections: make(map[string]*DriverConnection),
		FanIn:       make(chan dto.DriverMessage, 1000),
	}
}

func (m *WebSocketManager) RegisterDriver(ctx context.Context, driverID string, incoming <-chan []byte, outgoing chan<- []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, exists := m.connections[driverID]; exists {
		existing.Conn.Close()
	}

	m.connections[driverID] = &DriverConnection{
		DriverID:   driverID,
		fromDriver: incoming,
		toDriver:   outgoing,
		Auth:       false,
		LastPing:   time.Now(),
		SessionID:  fmt.Sprintf("session_%s_%d", driverID, time.Now().Unix()),
	}

	return nil
}

func (m *WebSocketManager) UnregisterDriver(ctx context.Context, driverID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, exists := m.connections[driverID]; exists {
		conn.Conn.Close()
		delete(m.connections, driverID)
	}
}

func (m *WebSocketManager) IsDriverConnected(driverID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conn, exists := m.connections[driverID]
	return exists && conn.Auth && time.Since(conn.LastPing) < 60*time.Second
}

func (m *WebSocketManager) SendToDriver(ctx context.Context, driverID string, message any) error {
	m.mu.RLock()
	conn, exists := m.connections[driverID]
	m.mu.RUnlock()

	if !exists || !conn.Auth {
		return fmt.Errorf("driver not connected or not authenticated: %s", driverID)
	}

	messageBytes, err := json.Marshal(message)
	fmt.Println("Sending to driver:", driverID, "message:", string(messageBytes))
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()
	conn.toDriver <- messageBytes
	// return conn.Conn.WriteMessage(websocket.TextMessage, messageBytes)
	return nil
}

func (m *WebSocketManager) SetConnection(driverID string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if driverConn, exists := m.connections[driverID]; exists {
		driverConn.Conn = conn
	}
}

func (m *WebSocketManager) SetAuthenticated(driverID string, authenticated bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, exists := m.connections[driverID]; exists {
		conn.Auth = authenticated
	}
}

func (m *WebSocketManager) UpdatePing(driverID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, exists := m.connections[driverID]; exists {
		conn.LastPing = time.Now()
	}
}

func (m *WebSocketManager) GetConnectionStatus(driverID string) *websocketdto.ConnectionStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conn, exists := m.connections[driverID]
	if !exists {
		return &websocketdto.ConnectionStatus{
			DriverID:  driverID,
			Connected: false,
		}
	}

	return &websocketdto.ConnectionStatus{
		DriverID:  driverID,
		Connected: conn.Auth && time.Since(conn.LastPing) < 60*time.Second,
		LastPing:  conn.LastPing,
		SessionID: conn.SessionID,
	}
}

func (m *WebSocketManager) GetConnectedDrivers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var drivers []string
	for driverID, conn := range m.connections {
		if conn.Auth && time.Since(conn.LastPing) < 60*time.Second {
			drivers = append(drivers, driverID)
		}
	}
	return drivers
}

func (m *WebSocketManager) GetDriverMessages(driverID string) (<-chan []byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conn, exists := m.connections[driverID]
	if !exists {
		return nil, fmt.Errorf("driver not connected: %s", driverID)
	}

	return conn.fromDriver, nil
}

func (m *WebSocketManager) GetDriversCount(ctx context.Context) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.connections)
}

func (m *WebSocketManager) GetFanIn() <-chan dto.DriverMessage {
	return m.FanIn
}
