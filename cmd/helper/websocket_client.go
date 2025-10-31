package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

type WebSocketClient struct {
	conn     *websocket.Conn
	toDriver chan []byte
	ctx      context.Context
	logger   *Logger
}

func NewWebSocketClient(ctx context.Context, logger *Logger) *WebSocketClient {
	return &WebSocketClient{
		toDriver: make(chan []byte, 100),
		ctx:      ctx,
		logger:   logger,
	}
}

func (w *WebSocketClient) Connect(url string) error {
	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("connecting to websocket: %w", err)
	}

	w.conn = conn
	w.logger.WebSocket("✅ WebSocket connected to %s", url)
	return nil
}

func (w *WebSocketClient) Close() error {
	if w.conn != nil {
		return w.conn.Close()
	}
	return nil
}

func (w *WebSocketClient) SendMessage(message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshaling message: %w", err)
	}

	if err := w.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("writing message: %w", err)
	}

	time.Sleep(50 * time.Millisecond) // Prevent overwhelming
	return nil
}

func (w *WebSocketClient) ReadMessages(handler func(messageType int, payload []byte) error) error {
	for {
		select {
		case <-w.ctx.Done():
			w.logger.WebSocket("Read loop stopped: context cancelled")
			return nil
		default:
			messageType, payload, err := w.conn.ReadMessage()
			if err != nil {
				return fmt.Errorf("reading message: %w", err)
			}

			if err := handler(messageType, payload); err != nil {
				w.logger.Error("Error handling message: %v", err)
			}
		}
	}
}
