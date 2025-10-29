package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"os/signal"
	websocketdto "ride-hail/internal/driver-location-service/core/domain/websocket_dto"
	"sync"
	"syscall"

	"github.com/gorilla/websocket"
)

type HttpRequest struct {
	Username      string `json:"username"`
	Email         string `json:"email"`
	Password      string `json:"password"`
	LicenseNumber string `json:"license_number"`
	VehicleType   string `json:"vehicle_type"`
	VehicleAttrs  struct {
		Make  string `json:"make"`
		Model string `json:"model"`
		Color string `json:"color"`
		Plate string `json:"plate"`
		Year  int    `json:"year"`
	} `json:"vehicle_attrs"`
}

type HttpResponse struct {
	JWT    string `json:"jwt"`
	Msg    string `json:"msg"`
	UserId string `json:"userId"`
}
type Client struct {
	ctx       context.Context
	conn      *websocket.Conn
	ToDriver  chan []byte
	FromDriver chan []byte

	Tohandle  chan []byte

	DriverId   string
	Jwt        string

	InOffer bool

	CurrentLat float64
	CurrentLng float64

	PickupLocationLng float64
	PickupLocationLat float64

	DestLocationLat float64
	DestpLocationLat float64

}

func (c *Client) read() {
	for {
		_, payload, err := c.conn.ReadMessage()
		if err != nil {
			log.Fatal("err to read")
		}
		data := websocketdto.WebSocketMessage{}
		err = json.Unmarshal(payload, &data)
		if err != nil {
			log.Printf("err to unmarshal")
		}
		switch data.Type {
		case websocketdto.MessageTypeRideOffer:
			data := websocketdto.RideOfferMessage{}
			err = json.Unmarshal(payload, &data)
			if err != nil {
				log.Printf("err to unmarshal ride offer: %v", err)
			}
			fmt.Printf("Ride offer received, gay: %+v\n", data)
			
			newData := websocketdto.RideResponseMessage{
				WebSocketMessage: websocketdto.WebSocketMessage{
					Type: websocketdto.MessageTypeRideResponse,
				},
				OfferID:  data.OfferID,
				RideID:   data.RideID,
				Accepted: true,
				CurrentLocation: websocketdto.Location{
					Latitude:  c.CurrentLat,
					Longitude: c.CurrentLng,
				},
			}
			responseData, err := json.Marshal(newData)
			if err != nil {
				log.Printf("err to marshal ride response: %v", err)
				continue
			}
			c.ToDriver <- responseData
		}

		fmt.Printf("get info: %+v", data)
	}
}

func (c *Client) write() {
	for {
		select {
		case msg, ok := <-c.ToDriver:
			if !ok {
				log.Fatal("err to read msg")
			}
			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("erro gay: %v", err)
				continue
			}
			err = c.conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				log.Fatal("err to write msg")
			}
			fmt.Printf("write info: %+v", data)

		case <-c.ctx.Done():
			return
		}
	}
}


func main() {

	HttpRequest := HttpRequest{
		Username:      "gay",
		Email:         fmt.Sprintf("%s@mail.com", randGenerate()),
		Password:      "gay123",
		LicenseNumber: randGenerate(),
		VehicleType:   "ECONOMY",
		VehicleAttrs: struct {
			Make  string "json:\"make\""
			Model string "json:\"model\""
			Color string "json:\"color\""
			Plate string "json:\"plate\""
			Year  int    "json:\"year\""
		}{
			Make:  "TOyota",
			Model: "camry",
			Color: "white",
			Plate: "KZ 123 ABC",
			Year:  2020,
		},
	}

	requestBOdy, err := json.Marshal(&HttpRequest)
	if err != nil {
		log.Fatalf("jr")
	}
	// create driver via http request automatcilcaspiuhpuaei
	responseHttp, err := http.Post("http://localhost:3010/driver/register", "application/json", bytes.NewBuffer(requestBOdy))
	if err != nil {
		log.Fatalf("cannot make http request: %v", err)
	}
	fmt.Print(responseHttp.StatusCode)
	bodyResponse, err := io.ReadAll(responseHttp.Body)
	if err != nil {
		log.Fatal("cannot make http request")

	}
	response := HttpResponse{}
	err = json.Unmarshal(bodyResponse, &response)
	if err != nil {
		log.Fatal("cannot unmarshal http request")
	}
	fmt.Printf("created driver res: %+v\n, %s", response, string(bodyResponse))
	// establish connection via ws
	fmt.Printf("sex: %v, lox: %s\n", response.UserId, fmt.Sprintf("ws://localhost:3001/ws/drivers/%s", response.UserId))

	c, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://localhost:3001/ws/drivers/%s", response.UserId), nil)
	if err != nil {
		log.Fatalf("gay error: %v", err)
	}
	defer c.Close()
	fmt.Printf("websocket connection\n")

	ctx, close := signal.NotifyContext(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer close()
	wg := sync.WaitGroup{}
	client := &Client{
		ToDriver: make(chan []byte),
		FromDriver: make(chan []byte),
		conn:     c,
		ctx:      ctx,
		DriverId: response.UserId,
		Jwt:      response.JWT,
	}

	wg.Add(2)
	go client.read()
	go client.write()

	<-ctx.Done()
	wg.Wait()
}

func randGenerate() string {
	a := []string{"A", "B", "C", "D", "E", "F", "H", "T"}
	res := ""
	for range 10 {
		res += a[rand.IntN(len(a))]
	}
	return res
}
