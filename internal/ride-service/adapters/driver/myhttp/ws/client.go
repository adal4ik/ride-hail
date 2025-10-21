package ws

import (
	"context"
	"encoding/json"

	websocketdto "ride-hail/internal/ride-service/core/domain/websocket_dto"

	"github.com/gorilla/websocket"
)

type Client struct {
	ctx         context.Context
	conn        *websocket.Conn
	dis         *Dispatcher
	egress      chan websocketdto.Event
	passengerId string
}

func NewClient(ctx context.Context, conn *websocket.Conn, dis *Dispatcher, passengerId string) *Client {
	return &Client{
		ctx:         ctx,
		conn:        conn,
		dis:         dis,
		egress:      make(chan websocketdto.Event),
		passengerId: passengerId,
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
		}
	}
}

func (c *Client) WriteMessage() {
	for {
		select {
		case <-c.ctx.Done():
			c.conn.Close()
			return
		case event, ok := <-c.egress:
			if !ok {
				return
			}

			

		}
	}
}
