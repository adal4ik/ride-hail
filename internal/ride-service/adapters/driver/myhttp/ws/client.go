package ws

import (
	"context"
	"encoding/json"

	websocketdto "ride-hail/internal/ride-service/core/domain/websocket_dto"

	"github.com/gorilla/websocket"
)
// TODO: add logging, add message approve, add main function to sent event for the client
type Client struct {
	ctx         context.Context
	conn        *websocket.Conn
	dis         *Dispatcher
	egress      chan websocketdto.Event
	passengerId string
	cancelAuth  context.CancelFunc
}

func NewClient(ctx context.Context, conn *websocket.Conn, dis *Dispatcher, passengerId string, cancelAuth context.CancelFunc) *Client {
	return &Client{
		ctx:         ctx,
		conn:        conn,
		dis:         dis,
		egress:      make(chan websocketdto.Event),
		passengerId: passengerId,
		cancelAuth:  cancelAuth,
	}
}

func (c *Client) ReadMessage() {
	c.conn.SetReadLimit(1024)

	// loop forever
	for {
		_, payload, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			}

			break
		}

		var req websocketdto.Event
		if err := json.Unmarshal(payload, &req); err != nil {
			continue
		}
		if req.Type == "auth"{
			c.cancelAuth()
		}
	}
}

func (c *Client) WriteMessage() {
	for {
		select {
		case <-c.ctx.Done():
			c.conn.Close()
			return
		case msg, ok := <-c.egress:
			if !ok {
				return
			}

			data, err := json.Marshal(msg)
			if err != nil {
				return // closes the connection, should we really
			}
			// Write a Regular text message to the connection
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			}
		}
	}
}
