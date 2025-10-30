package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand/v2"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	websocketdto "ride-hail/internal/driver-location-service/core/domain/websocket_dto"

	"github.com/gorilla/websocket"
)

// ANSI color codes
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Cyan   = "\033[36m"
	Gray   = "\033[90m"
)

func info(msg string, args ...interface{}) {
	log.Printf(Green+"[INFO] "+Reset+msg, args...)
}

func warn(msg string, args ...interface{}) {
	log.Printf(Yellow+"[WARN] "+Reset+msg, args...)
}

func errLog(msg string, args ...interface{}) {
	log.Printf(Red+"[ERROR] "+Reset+msg, args...)
}

func wsLog(msg string, args ...interface{}) {
	log.Printf(Cyan+"[WS] "+Reset+msg, args...)
}

func httpLog(msg string, args ...interface{}) {
	log.Printf(Gray+"[HTTP] "+Reset+msg, args...)
}

// --- Data structs omitted for brevity ---
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
	UserAttrs struct {
		PhoneNumber string `json:"phone"`
	} `json:"user_attrs"`
}

type HttpResponse struct {
	JWT    string `json:"jwt_access"`
	Msg    string `json:"msg"`
	UserId string `json:"driverId"`
}

type Client struct {
	ctx        context.Context
	conn       *websocket.Conn
	ToDriver   chan []byte
	FromDriver chan []byte
	Tohandle   chan []byte
	DriverId   string
	Jwt        string
	InOffer    bool
	CurrentLat float64
	CurrentLng float64
}

func (c *Client) read() {
	for {
		_, payload, err := c.conn.ReadMessage()
		if err != nil {
			errLog("read error: %v", err)
			return
		}
		data := websocketdto.WebSocketMessage{}
		if err := json.Unmarshal(payload, &data); err != nil {
			warn("cannot unmarshal base ws message: %v", err)
			continue
		}

		switch data.Type {
		case websocketdto.MessageTypeRideOffer:
			var offer websocketdto.RideOfferMessage
			if err := json.Unmarshal(payload, &offer); err != nil {
				errLog("cannot unmarshal ride offer: %v", err)
				continue
			}
			wsLog("🚗 Received ride offer: %+v", offer)

			resp := websocketdto.RideResponseMessage{
				WebSocketMessage: websocketdto.WebSocketMessage{
					Type: websocketdto.MessageTypeRideResponse,
				},
				OfferID:  offer.OfferID,
				RideID:   offer.RideID,
				Accepted: true,
				CurrentLocation: websocketdto.Location{
					Latitude:  c.CurrentLat,
					Longitude: c.CurrentLng,
				},
			}
			dataBytes, _ := json.Marshal(resp)
			c.ToDriver <- dataBytes
			wsLog("✅ Accepted ride offer %s", offer.OfferID)

			go c.LocationUpdate(offer, 30)

		default:
			wsLog("📨 WS message: %+v", data)
		}
	}
}

func (c *Client) write() {
	for {
		select {
		case msg, ok := <-c.ToDriver:
			if !ok {
				errLog("channel closed while writing")
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				errLog("write error: %v", err)
				return
			}
			wsLog("➡️ Sent message from driver %s: %s", c.DriverId, string(msg))
		case <-c.ctx.Done():
			wsLog("context cancelled, stopping writer")
			return
		}
	}
}

func (c *Client) LocationUpdate(offer websocketdto.RideOfferMessage, speed float64) {
	// speed = meters per second, e.g. 10.0 (≈ 36 km/h)
	updateInterval := 5 * time.Second // how often to send updates
	stepDistance := speed * updateInterval.Seconds()

	current := websocketdto.Location{
		Latitude:  c.CurrentLat,
		Longitude: c.CurrentLng,
	}
	target := offer.PickupLocation

	totalDistance := distance(current, target)
	if totalDistance < 1 {
		wsLog("✅ Already at pickup location.")
		return
	}

	steps := int(totalDistance / stepDistance)
	if steps < 1 {
		steps = 1
	}

	dLat := (target.Latitude - current.Latitude) / float64(steps)
	dLng := (target.Longitude - current.Longitude) / float64(steps)

	wsLog("🚗 Moving to pickup: distance=%.2fm, steps=%d, Δlat=%.6f, Δlng=%.6f",
		totalDistance, steps, dLat, dLng)

	ticker := time.NewTicker(updateInterval)
	defer ticker.Stop()

	for i := 0; i <= steps; i++ {
		select {
		case <-ticker.C:
			c.CurrentLat += dLat
			c.CurrentLng += dLng

			locUpdate := websocketdto.LocationUpdateMessage{
				WebSocketMessage: websocketdto.WebSocketMessage{
					Type: websocketdto.MessageTypeLocationUpdate,
				},
				Latitude:  c.CurrentLat,
				Longitude: c.CurrentLng,
			}
			dataBytes, _ := json.Marshal(locUpdate)
			c.ToDriver <- dataBytes

			wsLog("📍 Sent location update (%d/%d): lat=%.6f, lng=%.6f",
				i+1, steps, c.CurrentLat, c.CurrentLng)

			// Stop if we’ve reached (or overshot)
			if i == steps {
				wsLog("🏁 Arrived at pickup location (%.6f, %.6f)", c.CurrentLat, c.CurrentLng)
				return
			}
		case <-c.ctx.Done():
			wsLog("🛑 LocationUpdate stopped (context canceled).")
			return
		}
	}
}

func moveLinearly(current, target websocketdto.Location, steps int, delay time.Duration) {
	dLat := (target.Latitude - current.Latitude) / float64(steps)
	dLng := (target.Longitude - current.Longitude) / float64(steps)

	for i := 0; i <= steps; i++ {
		newLat := current.Latitude + dLat*float64(i)
		newLng := current.Longitude + dLng*float64(i)
		fmt.Printf("Step %d: Lat: %.6f, Lng: %.6f\n", i, newLat, newLng)
		time.Sleep(delay)
	}
}

// distance calculates the haversine distance between two coordinates in meters
func distance(a, b websocketdto.Location) float64 {
	const R = 6371000 // Earth radius in meters
	dLat := (b.Latitude - a.Latitude) * math.Pi / 180
	dLng := (b.Longitude - a.Longitude) * math.Pi / 180
	lat1 := a.Latitude * math.Pi / 180
	lat2 := b.Latitude * math.Pi / 180

	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLng/2)*math.Sin(dLng/2)*math.Cos(lat1)*math.Cos(lat2)
	return 2 * R * math.Asin(math.Sqrt(h))
}

func main() {
	HttpRequest := HttpRequest{
		Username:      "demo-driver",
		Email:         fmt.Sprintf("%s@mail.com", randGenerate()),
		Password:      "driver123",
		LicenseNumber: randGenerate(),
		VehicleType:   "ECONOMY",
	}
	HttpRequest.VehicleAttrs = struct {
		Make  string "json:\"make\""
		Model string "json:\"model\""
		Color string "json:\"color\""
		Plate string "json:\"plate\""
		Year  int    "json:\"year\""
	}{
		Make:  "Toyota",
		Model: "Camry",
		Color: "White",
		Plate: "KZ 123 ABC",
		Year:  2020,
	}
	HttpRequest.UserAttrs = struct {
		PhoneNumber string "json:\"phone\""
	}{PhoneNumber: "+7-123-456-78-90"}

	body, _ := json.Marshal(&HttpRequest)
	httpLog("Registering driver...")

	resp, err := http.Post("http://localhost:3010/driver/register", "application/json", bytes.NewBuffer(body))
	if err != nil {
		errLog("cannot register driver: %v", err)
		return
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)

	response := HttpResponse{}
	json.Unmarshal(data, &response)
	httpLog("Driver created: %s (JWT: %s)", response.UserId, response.JWT[:15]+"...")

	wsURL := fmt.Sprintf("ws://localhost:3001/ws/drivers/%s", response.UserId)
	wsLog("Connecting to WS: %s", wsURL)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		errLog("cannot connect WS: %v", err)
		return
	}
	defer conn.Close()
	wsLog("✅ WebSocket connected")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	client := &Client{
		ToDriver:   make(chan []byte),
		FromDriver: make(chan []byte),
		conn:       conn,
		ctx:        ctx,
		DriverId:   response.UserId,
		Jwt:        response.JWT,
		CurrentLat: 43.2220,
		CurrentLng: 76.8512,
	}

	go client.read()
	go client.write()

	authMsg, _ := json.Marshal(websocketdto.AuthMessage{
		WebSocketMessage: websocketdto.WebSocketMessage{
			Type: websocketdto.MessageTypeAuth,
		},
		Token: client.Jwt,
	})
	client.ToDriver <- authMsg
	wsLog("🪪 Auth message sent")

	// Set online
	locData, _ := json.Marshal(websocketdto.Location{
		Latitude:  client.CurrentLat,
		Longitude: client.CurrentLng,
	})

	req, _ := http.NewRequest("POST",
		fmt.Sprintf("http://localhost:3001/drivers/%s/online", client.DriverId),
		bytes.NewBuffer(locData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.Jwt))

	cl := &http.Client{}
	res, err := cl.Do(req)
	if err != nil {
		errLog("cannot set online: %v", err)
		return
	}
	defer res.Body.Close()
	httpLog("Driver set online: HTTP %d", res.StatusCode)

	info("🟢 Driver is online and waiting for ride offers...")

	<-ctx.Done()
	info("🛑 Shutting down gracefully")
}

func randGenerate() string {
	a := []string{"A", "B", "C", "D", "E", "F", "H", "T"}
	res := ""
	for range 10 {
		res += a[rand.IntN(len(a))]
	}
	return res
}
