package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"ride-hail/internal/mylogger"
	websocketdto "ride-hail/internal/ride-service/core/domain/websocket_dto"

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
	log := d.log.Action("AddClient")
	d.Lock()
	defer d.Unlock()

	d.clients[client.passengerId] = client
	log.Info("passenger successfully added", "passengerId", client.passengerId)
}

func (d *Dispatcher) RemoveClient(client *Client) {
	log := d.log.Action("RemoveClient")
	d.Lock()
	defer d.Unlock()

	if _, ok := d.clients[client.passengerId]; ok {
		client.conn.Close()
		delete(d.clients, client.passengerId)
		log.Info("passenger successfully deleted", "passengerId", client.passengerId)
	} else {
		log.Warn("passenger doesnt exist in map", "passengerId", client.passengerId)
	}
}

func (d *Dispatcher) WriteToUser(passengerId string, event websocketdto.Event) {
	if client, ok := d.clients[passengerId]; ok {
		client.egress <- event
	}
}

func (d *Dispatcher) StartTimerAuth(client *Client, cancel context.CancelFunc, ctxAuth context.Context) {
	type msg struct {
		Text string `json:"text"`
	}
	select {
	case <-time.After(time.Second * 5):
		msg := msg{
			Text: "time out",
		}
		data, _ := json.Marshal(msg)
		event := websocketdto.Event{
			Type: "auth",
			Data: data,
		}

		d.WriteToUser(client.passengerId, event)
		cancel()
	case <-ctxAuth.Done():
		msg := msg{
			Text: "auth success",
		}
		data, _ := json.Marshal(msg)
		event := websocketdto.Event{
			Type: "auth",
			Data: data,
		}
		d.WriteToUser(client.passengerId, event)
		return
	}
}
