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
	"sync"
	"syscall"

	"github.com/gorilla/websocket"
)

type LocationDetail struct {
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	Address string  `json:"address"`
}

type DriverRideOffer struct {
	Type            string         `json:"type"`
	Ride_id         string         `json:"ride_id"`
	Passenger_name  string         `json:"passenger_name"`
	Passenger_phone string         `json:"passenger_phone"`
	Pickup_location LocationDetail `json:"pickup_location"`
}

type DriverResponse struct {
	Type             string               `json:"type"`
	Offer_id         string               `json:"offer_id"`
	Ride_id          string               `json:"ride_id"`
	Accepted         bool                 `json:"accepted"`
	Current_location DriverCoordinatesDTO `json:"current_location"`
}
type DriverCoordinatesDTO struct {
	Driver_id string  `json:"driver_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
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
	Incoming  chan DriverRideOffer
	Outcoming chan DriverResponse
	Tohandle  chan DriverRideOffer

	DriverId   string
	Jwt        string
	CurrentLat float64
	CurrentLng float64

	PickupLocationLng float64
	PickupLocationLat float64
}

func (c *Client) read() {
	for {
		_, payload, err := c.conn.ReadMessage()
		if err != nil {
			log.Fatal("err to read")
		}
		data := DriverRideOffer{}
		err = json.Unmarshal(payload, &data)
		if err != nil {
			log.Printf("err to unmarshal")
		}
		fmt.Printf("gay info: %+v", data)
		c.Tohandle <- data
	}
}

func (c *Client) write() {
	for {
		select {
		case msg, ok := <-c.Outcoming:
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

func (c *Client) handle() {
	for {
		select {
		case msg, ok := <-c.Tohandle:
			if !ok {
				log.Fatal("lox")
				return
			}

			c.PickupLocationLat = msg.Pickup_location.Lat
			c.PickupLocationLng = msg.Pickup_location.Lng
			res := DriverResponse{
				Type:     "",
				Offer_id: "lox",
				Ride_id:  msg.Ride_id,
				Accepted: true,
				Current_location: DriverCoordinatesDTO{
					Driver_id: c.DriverId,
					Latitude:  c.CurrentLat,
					Longitude: c.CurrentLng,
				},
			}
			c.Outcoming <- res
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
	ch := make(chan DriverRideOffer)
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
		Incoming: ch,
		conn:     c,
		ctx:      ctx,
		DriverId: response.UserId,
		Jwt:      response.JWT,
	}

	wg.Add(3)
	go client.read()
	go client.write()
	go client.handle()

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
