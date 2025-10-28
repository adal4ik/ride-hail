package ws

import (
	"context"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type WebSocketManager struct {
	connections map[string]*DriverConnection
	mu          sync.RWMutex
}

type DriverConnection struct {
	DriverID string
	Conn     *websocket.Conn
	Incoming chan<- []byte
	Outgoing <-chan []byte
	Auth     bool
	mu       sync.Mutex
}

func NewWebSocketManaget() *WebSocketManager {
	return &WebSocketManager{
		connections: make(map[string]*DriverConnection),
	}
}

func (m *WebSocketManager) RegisterDriver(ctx context.Context, driverID string, incoming chan<- []byte, outgoing <-chan []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, exists := m.connections[driverID]; exists {
		existing.Conn.Close()
	}

	m.connections[driverID] = &DriverConnection{
		DriverID: driverID,
		Incoming: incoming,
		Outgoing: outgoing,
		Auth:     false,
	}
	return nil
}

func (m *WebSocketManager) IsDriverConnected(driverID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, exists := m.connections[driverID]
	return exists && conn.Auth
}

func (m *WebSocketManager) SendToDriver(ctx context.Context, driverID string, message []byte) error {
	m.mu.RLock()
	conn, exists := m.connections[driverID]
	m.mu.RUnlock()

	if !exists || !conn.Auth {
		log.Println("Driver not connection or not authenticated: ", driverID)
		return nil
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()
	return conn.Conn.WriteMessage(websocket.TextMessage, message)
}

func (m *WebSocketManager) SetConnection(driverID string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Lock()

	if driverCon, exists := m.connections[driverID]; exists {
		driverCon.Conn = conn
	}
}

func (m *WebSocketManager) UnregisterDriver(ctx context.Context, driverID string) {
	m.mu.Lock()
	defer m.mu.Lock()

	if conn, exists := m.connections[driverID]; exists {
		conn.Conn.Close()
		delete(m.connections, driverID)
	}
}

func (m *WebSocketManager) SetAuthenticated(driverID string, authenticated bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if driverCon, exists := m.connections[driverID]; exists {
		driverCon.Auth = authenticated
	}
}
