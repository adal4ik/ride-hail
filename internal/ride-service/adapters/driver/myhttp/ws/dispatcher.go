package ws

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"ride-hail/internal/mylogger"
	websocketdto "ride-hail/internal/ride-service/core/domain/websocket_dto"
	"ride-hail/internal/ride-service/core/ports"

	"github.com/gorilla/websocket"
)

var ErrEventNotSupported = errors.New("this event type is not supported")

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
	PassengerService ports.IPassengerService
	eventHandler     EventHandler
	hander           map[string]EventHandle
	clients          ClientList
	sync.RWMutex
	log mylogger.Logger
}

func NewDispathcer(log mylogger.Logger, passengerRepo ports.IPassengerService, eventHader EventHandler) *Dispatcher {
	return &Dispatcher{
		clients:          make(ClientList),
		PassengerService: passengerRepo,
		log:              log,
		eventHandler:     eventHader,
	}
}

func (d *Dispatcher) InitHandler() {
	d.hander["auth"] = d.eventHandler.AuthHandler
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

		ok, err := d.PassengerService.FindPassenger(passengerId)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		conn, err := websocketUpgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error("cannot upgrade", err)
			return
		}
		ctx, cancel := context.WithCancel(context.Background())
		ctxAuth, cancelAuth := context.WithCancel(context.Background())

		client := NewClient(ctx, d.log, conn, d, passengerId, cancelAuth)
		d.AddClient(client)
		go client.ReadMessage()
		go client.WriteMessage()
		go d.StartTimerAuth(client, cancel, ctxAuth)

		// Code from temu
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
	d.Lock()
	defer d.Unlock()

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

func (d *Dispatcher) EventHandle(client *Client, event websocketdto.Event) error {
	if handler, ok := d.hander[event.Type]; ok {
		// Execute the handler and return any err
		if err := handler(client, event); err != nil {
			return err
		}
		return nil
	} else {
		return ErrEventNotSupported
	}
}
