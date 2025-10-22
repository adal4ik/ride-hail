package ws

import (
	"context"
	"encoding/json"

	"ride-hail/internal/mylogger"
	websocketdto "ride-hail/internal/ride-service/core/domain/websocket_dto"

	"github.com/gorilla/websocket"
)

// TODO: add logging, add main function to sent event for the client, ping pong
type Client struct {
	log         mylogger.Logger
	ctx         context.Context
	conn        *websocket.Conn
	dispatcher  *Dispatcher
	egress      chan websocketdto.Event
	passengerId string
	cancelAuth  context.CancelFunc
}

func NewClient(ctx context.Context, log mylogger.Logger, conn *websocket.Conn, dis *Dispatcher, passengerId string, cancelAuth context.CancelFunc) *Client {
	return &Client{
		log:         log,
		ctx:         ctx,
		conn:        conn,
		dispatcher:  dis,
		egress:      make(chan websocketdto.Event),
		passengerId: passengerId,
		cancelAuth:  cancelAuth,
	}
}

func (c *Client) ReadMessage() {
	defer func() {
		c.dispatcher.RemoveClient(c)
	}()
	log := c.log.Action("ReadMessage").With("passenger-id", c.passengerId)
	c.conn.SetReadLimit(1024)

	// loop forever
	for {
		_, payload, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error("cannot read message", err)
			}
			break
		}

		var req websocketdto.Event
		if err := json.Unmarshal(payload, &req); err != nil {
			log.Warn("cannot unmarshal json", "err", err)

			continue
		}

		if err := c.dispatcher.EventHandle(c, req); err != nil {
			log.Error("cannot handle event", err)
		}

	}
}

func (c *Client) WriteMessage() {
	defer func() {
		c.dispatcher.RemoveClient(c)
	}()
	log := c.log.Action("WriteMessage").With("passenger-id", c.passengerId)

	for {
		select {
		case <-c.ctx.Done():
			c.conn.Close()
			return
		case msg, ok := <-c.egress:
			if !ok {
				log.Info("egress is closed")
				// dispathcer has closed this connection channel, so communicate that to frontend
				if err := c.conn.WriteMessage(websocket.CloseMessage, nil); err != nil {
					// Log that the connection is closed and the reason
					log.Error("connection closed: ", err)
				}
				return
			}

			data, err := json.Marshal(msg)
			if err != nil {
				log.Error("cannot marshal message", err)

				return // closes the connection, should we really
			}
			// Write a Regular text message to the connection
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Error("cannot write message", err)
			}
		}
	}
}
