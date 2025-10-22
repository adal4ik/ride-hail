package ws

import (
	"context"
	"encoding/json"
	"ride-hail/internal/mylogger"
	"time"

	websocketdto "ride-hail/internal/ride-service/core/domain/websocket_dto"

	"github.com/gorilla/websocket"
)

var (
	// pongWait is how long we will await a pong response from client
	pongWait = 10 * time.Second
	// pingInterval has to be less than pongWait, We cant multiply by 0.9 to get 90% of time
	// Because that can make decimals, so instead *9 / 10 to get 90%
	// The reason why it has to be less than PingRequency is becuase otherwise it will send a new Ping before getting response
	pingInterval = (pongWait * 9) / 10
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

	c.conn.SetPongHandler(c.PingHandler)

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

	ticker := time.NewTicker(pingInterval)

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
		case <-ticker.C:
			log.Info("ping")

			if err := c.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.Error("write to ping", err)
				return // return to break this goroutine triggeing cleanup
			}
		}
	}
}

func (c *Client) PingHandler(pongMessage string) error {
	log := c.log.Action("PingHandler").With("passenger-id", c.passengerId)
	log.Info("pong")
	return c.conn.SetReadDeadline(time.Now().Add(pongWait))
}
