package main

import (
	"bytes"
	"context"
	"crypto/tls"
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

// Configuration constants for rate limiting
const (
	LocationUpdateInterval = 3 * time.Second        // How often to send location updates
	DBWriteDelay           = 100 * time.Millisecond // Delay after DB operations
	HTTPRequestDelay       = 200 * time.Millisecond // Delay between HTTP requests
	InitialConnectDelay    = 1 * time.Second        // Delay after initial connection
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
type HttpRequest2 struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type HttpRequest3 struct {
	RideId        string   `json:"ride_id"`
	FinalLocation FinalLoc `json:"final_location"`
	ActualDist    float64  `json:"actual_distance_km"`
	ActualDur     int      `json:"actual_duration_minutes"`
}

type FinalLoc struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type HttpResponse struct {
	JWT    string `json:"jwt_access"`
	Msg    string `json:"msg"`
	UserId string `json:"driverId"`
}
type HttpResponse2 struct {
	Status    string `json:"status"`
	SessionId string `json:"session_id"`
	Message   string `json:"message"`
}
type HttpResponse3 struct {
	Message        string  `json:"message"`
	RideID         string  `json:"ride_id"`
	Status         string  `json:"status"`
	CompletedAt    string  `json:"completed_at"`
	DriverEarnings float64 `json:"driver_earnings"`
}
type HttpRequest4 struct {
	RideID         string   `json:"ride_id"`
	DriverLocation Location `json:"driver_location"`
}
type HttpResponse4 struct {
	RideID    string `json:"ride_id"`
	Status    string `json:"status"`
	StartedAt string `json:"started_at"`
	Message   string `json:"message"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Driver_id string  `json:"driver_id,omitempty"`
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
	defer func() {
		close(c.ToDriver) // tell writer to stop
		wsLog("read loop ended, closing writer")
	}()
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

			// Add small delay before starting location updates
			time.Sleep(DBWriteDelay)
			go c.LocationUpdate(offer, 500)

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
			// Small delay after sending message to prevent overwhelming
			time.Sleep(50 * time.Millisecond)
		case <-c.ctx.Done():
			wsLog("context cancelled, stopping writer")
			return
		}
	}
}

func (c *Client) LocationUpdate(offer websocketdto.RideOfferMessage, speed float64) {
	current := websocketdto.Location{
		Latitude:  c.CurrentLat,
		Longitude: c.CurrentLng,
	}
	target := offer.PickupLocation

	c.goToTarget(current, target, speed)

	info("arrived to pickup location", "lat", c.CurrentLat, "lng", c.CurrentLng)

	// Add delay before starting ride
	time.Sleep(DBWriteDelay * 2)

	// START RIDE - Fixed implementation
	startReq := HttpRequest4{
		RideID: offer.RideID,
		DriverLocation: Location{
			Latitude:  c.CurrentLat,
			Longitude: c.CurrentLng,
		},
	}

	body0, err := json.Marshal(&startReq)
	if err != nil {
		errLog("Error marshaling start ride request: %v", err)
		return
	}

	httpLog("Starting ride %s...", offer.RideID)
	// Create HTTP request with proper headers
	startURL := fmt.Sprintf("https://localhost:3001/drivers/%s/start", c.DriverId)
	req0, err := http.NewRequest("POST", startURL, bytes.NewBuffer(body0))
	if err != nil {
		errLog("Error creating start ride request: %v", err)
		return
	} // Add authorization header
	req0.Header.Set("Authorization", "Bearer "+c.Jwt)
	req0.Header.Set("Content-Type", "application/json")

	// Add delay before HTTP request
	time.Sleep(HTTPRequestDelay)

	client0 := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 10 * time.Second, // Add timeout
	}

	resp0, err := client0.Do(req0)
	if err != nil {
		errLog("Cannot start ride: %v", err)
		return
	}
	defer resp0.Body.Close()

	data0, _ := io.ReadAll(resp0.Body)

	response0 := HttpResponse4{}
	json.Unmarshal(data0, &response0)

	httpLog(fmt.Sprint("Ride started: %s", response0.Message))

	// Delay between HTTP requests
	time.Sleep(HTTPRequestDelay)

	// Add delay before starting next leg
	time.Sleep(DBWriteDelay)

	current2 := websocketdto.Location{
		Latitude:  c.CurrentLat,
		Longitude: c.CurrentLng,
	}
	target2 := offer.DestinationLocation

	c.goToTarget(current2, target2, speed)

	info("arrived to destination location", "lat", c.CurrentLat, "lng", c.CurrentLng)

	// Add delay before completing ride
	time.Sleep(DBWriteDelay)

	HttpRequest3 := HttpRequest3{
		RideId: offer.RideID,
		FinalLocation: FinalLoc{
			Latitude:  c.CurrentLat,
			Longitude: c.CurrentLng,
		},
		ActualDist: 5.5,
		ActualDur:  16,
	}
	httpLog("Completing ride...")

	body3, _ := json.Marshal(&HttpRequest3)

	url := fmt.Sprintf("https://localhost:3001/drivers/%s/complete", c.DriverId)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body3))
	if err != nil {
		errLog("Error creating request", "err", err)
		return
	}

	jwtToken := c.Jwt
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Content-Type", "application/json")

	// Add delay before HTTP request
	time.Sleep(HTTPRequestDelay)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		errLog("cannot complete the drive", "err", err)
		return
	}
	defer resp.Body.Close()

	// Add delay after HTTP request
	time.Sleep(DBWriteDelay)

	data, _ := io.ReadAll(resp.Body)
	response := HttpResponse3{}
	json.Unmarshal(data, &response)

	httpLog("Completion info", "msg", response.Message, "s", response.Status)
	fmt.Printf("🏁 Ride completed: ID=%s, Earnings=%.2f\n", response.RideID, response.DriverEarnings)
}

func (c *Client) goToTarget(current, target websocketdto.Location, speed float64) {
	// speed = meters per second, e.g. 10.0 (≈ 36 km/h)
	stepDistance := speed * LocationUpdateInterval.Seconds()

	// Calculate the total distance from current to target location
	totalDistance := distance(current, target)
	if totalDistance < 1 {
		wsLog("✅ Already at pickup location.")
		return
	}

	// Calculate the number of steps to take based on speed and distance
	steps := int(totalDistance / stepDistance)
	if steps < 1 {
		steps = 1
	}

	// Calculate the differences in latitude and longitude between target and current location
	dLat := (target.Latitude - current.Latitude) / float64(steps)
	dLng := (target.Longitude - current.Longitude) / float64(steps)

	wsLog("🚗 Moving to pickup: distance=%.2fm, steps=%d, Δlat=%.6f, Δlng=%.6f",
		totalDistance, steps, dLat, dLng)

	// Use the configured interval for location updates
	ticker := time.NewTicker(LocationUpdateInterval)
	defer ticker.Stop()

	for i := 0; i < steps; i++ {
		select {
		case <-ticker.C:
			// Update current position by adding dLat and dLng to it
			c.CurrentLat += dLat
			c.CurrentLng += dLng

			// Send the location update
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

			// Small delay after sending update to prevent overwhelming the system
			time.Sleep(DBWriteDelay)

		case <-c.ctx.Done():
			wsLog("🛑 LocationUpdate stopped (context canceled).")
			return
		}
	}

	// Final step to arrive at the target (precisely)
	c.CurrentLat = target.Latitude
	c.CurrentLng = target.Longitude

	// Send the final location update
	locUpdate := websocketdto.LocationUpdateMessage{
		WebSocketMessage: websocketdto.WebSocketMessage{
			Type: websocketdto.MessageTypeLocationUpdate,
		},
		Latitude:       c.CurrentLat,
		Longitude:      c.CurrentLng,
		AccuracyMeters: 5.0,
		SpeedKmh:       45.0,
		HeadingDegrees: 90.0,
	}
	dataBytes, _ := json.Marshal(locUpdate)
	c.ToDriver <- dataBytes

	// Delay after final update
	time.Sleep(DBWriteDelay)

	wsLog("🏁 Arrived at pickup location (%.6f, %.6f)", c.CurrentLat, c.CurrentLng)
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

	// Add initial delay
	time.Sleep(InitialConnectDelay)

	client0 := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client0.Post("https://localhost:3010/driver/register", "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Fatalf("Cannot register driver: %v", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	response := HttpResponse{}
	json.Unmarshal(data, &response)
	httpLog("Driver created: %s (JWT: %s)", response.UserId, response.JWT[:15]+"...")

	// Delay between HTTP requests
	time.Sleep(HTTPRequestDelay)

	// second request ==========================================================================
	HttpRequest2 := HttpRequest2{
		Latitude:  43.283859,
		Longitude: 76.999909,
	}
	httpLog("Making him online...")

	body2, _ := json.Marshal(&HttpRequest2)

	url := fmt.Sprintf("https://localhost:3001/drivers/%s/online", response.UserId)
	req0, err := http.NewRequest("POST", url, bytes.NewBuffer(body2))
	if err != nil {
		errLog("Error creating request: %v", err)
		return
	}

	jwtToken := response.JWT
	req0.Header.Set("Authorization", "Bearer "+jwtToken)
	req0.Header.Set("Content-Type", "application/json")

	// Delay before HTTP request
	time.Sleep(HTTPRequestDelay)

	client2 := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp2, err := client2.Do(req0)
	if err != nil {
		errLog("cannot make online driver: %v", err)
		return
	}
	defer resp2.Body.Close()

	data2, _ := io.ReadAll(resp2.Body)
	response2 := HttpResponse2{}
	json.Unmarshal(data2, &response2)

	httpLog("Online info", "msg", response2.Message)

	// Delay before WebSocket connection
	time.Sleep(InitialConnectDelay)

	// WS logic
	wsURL := fmt.Sprintf("wss://localhost:3001/ws/drivers/%s", response.UserId)
	wsLog("Connecting to WSs: %s", wsURL)

	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		log.Printf("cannot connect to WebSocket: %v", err)
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

	// Delay before auth
	time.Sleep(DBWriteDelay)

	authMsg, _ := json.Marshal(websocketdto.AuthMessage{
		WebSocketMessage: websocketdto.WebSocketMessage{
			Type: websocketdto.MessageTypeAuth,
		},
		Token: client.Jwt,
	})
	client.ToDriver <- authMsg
	wsLog("Auth message sent")

	// Delay before setting online
	time.Sleep(HTTPRequestDelay)

	// Set online
	locData, _ := json.Marshal(websocketdto.Location{
		Latitude:  client.CurrentLat,
		Longitude: client.CurrentLng,
	})

	req, _ := http.NewRequest("POST",
		fmt.Sprintf("https://localhost:3001/drivers/%s/online", client.DriverId),
		bytes.NewBuffer(locData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.Jwt))

	cl := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
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
