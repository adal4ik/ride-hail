package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"ride-hail/internal/config"
	"ride-hail/internal/mylogger"
	"time"

	"github.com/gorilla/websocket"
)

// Message structure for authentication
type AuthMessage struct {
	Type string `json:"type"`
	Data struct {
		Token string `json:"token"`
	} `json:"data"`
}

// WebSocket URL for driver connection
func main() {
	// Initialize config and logger
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	appLogger, err := mylogger.New(cfg.Log.Level)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	appLogger.Action("ride_hail_system_started").Info("Ride Hail System starting up")

	// Get driver_id and token from CLI flags
	driverID := flag.String("driver_id", "", "Driver ID to connect to WebSocket")
	driverToken := flag.String("token", "", "Driver token for authentication")
	flag.Parse()

	if *driverID == "" || *driverToken == "" {
		log.Fatal("Driver ID and token are required")
	}

	// WebSocket URL
	wsURL := fmt.Sprintf("ws://localhost:3001/ws/drivers/%s", *driverID)

	// Connect to WebSocket server
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Fatalf("Failed to connect to WebSocket server: %v", err)
	}
	defer conn.Close()
	appLogger.Action("WebSocket_connected").Info("Connected to WebSocket server", "driver_id", *driverID)

	// Prepare the authentication message
	authMessage := AuthMessage{
		Type: "auth",
	}
	authMessage.Data.Token = *driverToken // Set the provided token

	// Convert the authentication message to JSON
	authBytes, err := json.Marshal(authMessage)
	if err != nil {
		appLogger.Error("Error marshalling auth message", err)
		return
	}

	// Send the authentication message
	err = conn.WriteMessage(websocket.TextMessage, authBytes)
	if err != nil {
		appLogger.Error("Error sending authentication message", err)
		return
	}

	appLogger.Info("Sent authentication message", "message", string(authBytes))

	// Function to send JSON messages
	sendJSON := func(msg interface{}) {
		bytes, _ := json.Marshal(msg)
		if err := conn.WriteMessage(websocket.TextMessage, bytes); err != nil {
			appLogger.Error("Error sending message", err)
		} else {
			appLogger.Info("Sent message", "message", string(bytes))
		}
	}

	// Simulate driver location
	driverLocation := struct {
		Latitude  float64
		Longitude float64
	}{
		Latitude:  43.236,
		Longitude: 76.886,
	}

	// Read messages from server
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				appLogger.Error("Error reading WebSocket message", err)
				break
			}

			var msg map[string]interface{}
			json.Unmarshal(message, &msg)

			msgType := msg["type"].(string)
			appLogger.Info("Received message", "type", msgType, "message", string(message))

			switch msgType {
			case "ride_offer":
				// Automatically accept ride offers
				rideResp := struct {
					Type            string `json:"type"`
					OfferID         string `json:"offer_id"`
					RideID          string `json:"ride_id"`
					Accepted        bool   `json:"accepted"`
					CurrentLocation struct {
						Latitude  float64 `json:"latitude"`
						Longitude float64 `json:"longitude"`
					} `json:"current_location"`
				}{
					Type:     "ride_response",
					OfferID:  msg["offer_id"].(string),
					RideID:   msg["ride_id"].(string),
					Accepted: true,
					CurrentLocation: struct {
						Latitude  float64 `json:"latitude"`
						Longitude float64 `json:"longitude"`
					}{
						Latitude:  driverLocation.Latitude,
						Longitude: driverLocation.Longitude,
					},
				}
				sendJSON(rideResp)
			}
		}
	}()

	// Periodically send driver location updates
	ticker := time.NewTicker(2 * time.Second)
	for range ticker.C {
		// Simulate small movement
		driverLocation.Latitude += (rand.Float64() - 0.5) / 1000
		driverLocation.Longitude += (rand.Float64() - 0.5) / 1000

		locationUpdate := struct {
			Type           string  `json:"type"`
			Latitude       float64 `json:"latitude"`
			Longitude      float64 `json:"longitude"`
			AccuracyMeters float64 `json:"accuracy_meters"`
			SpeedKmh       float64 `json:"speed_kmh"`
			HeadingDegrees float64 `json:"heading_degrees"`
		}{
			Type:           "location_update",
			Latitude:       driverLocation.Latitude,
			Longitude:      driverLocation.Longitude,
			AccuracyMeters: 5.0,
			SpeedKmh:       40 + rand.Float64()*10,
			HeadingDegrees: rand.Float64() * 360,
		}

		sendJSON(locationUpdate)
	}
}
