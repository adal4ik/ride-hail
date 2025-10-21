package ws

import (
	"context"
	"net/http"
	"sync"
	"time"

	"ride-hail/internal/mylogger"

	"github.com/gorilla/websocket"
)

// ================================================================================================== //
// websocketUpgrader is used to upgrade incomming HTTP requests into a persitent websocket connection //
// ================================================================================================== //
var websocketUpgrader = websocket.Upgrader{
	// TODO: add checkOrigin
	// // Apply the Origin Checker
	// CheckOrigin:     checkOrigin,
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// ClientList is a map used to help manage a map of clients
type ClientList map[string]*Client

type Dispatcher struct {
	clients ClientList
	sync.RWMutex
	log mylogger.Logger
}

func NewDispathcer(log mylogger.Logger) *Dispatcher {
	return &Dispatcher{
		clients: make(ClientList),
		log:     log,
	}
}

func (d *Dispatcher) WsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := d.log.Action("loginHandler")
		passengerId := r.PathValue("passenger_id")

		if passengerId == "" {
			w.WriteHeader(http.StatusUnauthorized)
			log.Warn("how it even possible?")
			return
		}

		conn, err := websocketUpgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error("cannot upgrade", err)
			return
		}
		ctx, cancel := context.WithCancel(context.Background())
		ctxAuth, cancelAuth := context.WithCancel(context.Background())

		client := NewClient(ctx, conn, d, passengerId, cancelAuth)
		d.AddClient(client)
		go client.ReadMessage()
		go client.WriteMessage()
		go d.StartTimerAuth(client, cancel, ctxAuth)
		// select {
		// case <-time.After(time.Second * 5):
		// 	// close connection
		// 	conn.Close()
		// case msg := <-client.egress:
		// 	if msg.Type != "auth" {
		// 	}
		// }
	}
}

// TODO: write to client, via map, without any channel

func (d *Dispatcher) AddClient(client *Client) {
	d.Lock()
	defer d.Unlock()

	d.clients[client.passengerId] = client
}

func (d *Dispatcher) StartTimerAuth(client *Client, cancel context.CancelFunc, ctxAuth context.Context) {
	select {
	case <-time.After(time.Second * 5):
		cancel()
	case <-ctxAuth.Done():
		return
	}
}
