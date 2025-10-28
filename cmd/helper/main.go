package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/gorilla/websocket"
)

type AuthMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

type DriverResponse struct {
	Type            string `json:"type"`
	OfferID         string `json:"offer_id"`
	RideID          string `json:"ride_id"`
	Accepted        bool   `json:"accepted"`
	CurrentLocation struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"current_location"`
}

type RideOffer struct {
	Type           string `json:"type"`
	OfferID        string `json:"offer_id"`
	RideID         string `json:"ride_id"`
	RideNumber     string `json:"ride_number"`
	PickupLocation struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Address   string  `json:"address"`
	} `json:"pickup_location"`
	DestinationLocation struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Address   string  `json:"address"`
	} `json:"destination_location"`
	EstimatedFare float64 `json:"estimated_fare"`
	DriverEarning float64 `json:"driver_earnings"`
}

type RideStatusUpdate struct {
	Type      string `json:"type"`
	RideID    string `json:"ride_id"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	Timestamp string `json:"timestamp"`
}

type LocationUpdate struct {
	Type           string  `json:"type"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	AccuracyMeters float64 `json:"accuracy_meters"`
	SpeedKmh       float64 `json:"speed_kmh"`
	HeadingDegrees float64 `json:"heading_degrees"`
}

// Linear interpolation helper
func interpolate(a, b, t float64) float64 {
	return a + (b-a)*t
}

// Haversine distance (in km)
func distanceKm(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * R * math.Asin(math.Sqrt(a))
}

// Simulate smooth movement from A to B
func simulateMovement(conn *websocket.Conn, rideID string, startLat, startLng, endLat, endLng float64, speedKmh float64) {
	distKm := distanceKm(startLat, startLng, endLat, endLng)
	durationHours := distKm / speedKmh
	duration := time.Duration(durationHours * float64(time.Hour))
	steps := int(duration.Seconds() / 2) // every 2 seconds

	fmt.Printf("Simulating movement for ride %s (%.2f km, %d steps)\n", rideID, distKm, steps)

	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		lat := interpolate(startLat, endLat, t)
		lng := interpolate(startLng, endLng, t)
		heading := rand.Float64() * 360

		update := LocationUpdate{
			Type:           "location_update",
			Latitude:       lat,
			Longitude:      lng,
			AccuracyMeters: 5.0,
			SpeedKmh:       speedKmh,
			HeadingDegrees: heading,
		}
		data, _ := json.Marshal(update)
		conn.WriteMessage(websocket.TextMessage, data)
		time.Sleep(2 * time.Second)
	}
}

func main() {
	driverID := flag.String("driver_id", "", "Driver ID to connect to WebSocket")
	driverToken := flag.String("token", "", "Driver token for authentication")
	flag.Parse()

	if *driverID == "" || *driverToken == "" {
		log.Fatal("Driver ID and token are required")
	}

	wsURL := fmt.Sprintf("ws://localhost:3001/ws/drivers/%s", *driverID)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Fatalf("Failed to connect to WebSocket server: %v", err)
	}
	defer conn.Close()
	fmt.Println("Connected to WebSocket")

	// Authenticate
	authMsg := AuthMessage{
		Type:  "auth",
		Token: fmt.Sprintf("Bearer %s", *driverToken),
	}
	authBytes, _ := json.Marshal(authMsg)
	conn.WriteMessage(websocket.TextMessage, authBytes)
	fmt.Println("Sent authentication")

	// Read messages
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}

		var msg map[string]interface{}
		json.Unmarshal(message, &msg)
		msgType := msg["type"].(string)
		fmt.Println("Received:", msgType)

		switch msgType {
		case "ride_offer":
			// Parse ride offer
			var offer RideOffer
			json.Unmarshal(message, &offer)
			fmt.Printf("Received ride offer: %s → %s\n", offer.PickupLocation.Address, offer.DestinationLocation.Address)

			// Accept ride
			resp := DriverResponse{
				Type:     "ride_response",
				OfferID:  offer.OfferID,
				RideID:   offer.RideID,
				Accepted: true,
			}
			resp.CurrentLocation.Latitude = 43.236
			resp.CurrentLocation.Longitude = 76.886
			respBytes, _ := json.Marshal(resp)
			conn.WriteMessage(websocket.TextMessage, respBytes)
			fmt.Println("Accepted ride")

			// Simulate movement: to pickup
			status := RideStatusUpdate{
				Type:      "ride_status_update",
				RideID:    offer.RideID,
				Status:    "EN_ROUTE_TO_PICKUP",
				Timestamp: time.Now().Format(time.RFC3339),
			}
			data, _ := json.Marshal(status)
			conn.WriteMessage(websocket.TextMessage, data)

			simulateMovement(conn, offer.RideID, 43.236, 76.886, offer.PickupLocation.Latitude, offer.PickupLocation.Longitude, 40)

			// Arrived at pickup
			arrived := RideStatusUpdate{
				Type:      "ride_status_update",
				RideID:    offer.RideID,
				Status:    "ARRIVED_PICKUP",
				Message:   "Driver has arrived at pickup",
				Timestamp: time.Now().Format(time.RFC3339),
			}
			arrBytes, _ := json.Marshal(arrived)
			conn.WriteMessage(websocket.TextMessage, arrBytes)
			fmt.Println("Arrived at pickup")

			time.Sleep(5 * time.Second)

			// Start ride
			started := RideStatusUpdate{
				Type:      "ride_status_update",
				RideID:    offer.RideID,
				Status:    "RIDE_STARTED",
				Timestamp: time.Now().Format(time.RFC3339),
			}
			startBytes, _ := json.Marshal(started)
			conn.WriteMessage(websocket.TextMessage, startBytes)
			fmt.Println("Ride started")

			// Move to destination
			simulateMovement(conn, offer.RideID, offer.PickupLocation.Latitude, offer.PickupLocation.Longitude, offer.DestinationLocation.Latitude, offer.DestinationLocation.Longitude, 45)

			// Complete ride
			completed := RideStatusUpdate{
				Type:      "ride_status_update",
				RideID:    offer.RideID,
				Status:    "RIDE_COMPLETED",
				Message:   "Ride completed successfully",
				Timestamp: time.Now().Format(time.RFC3339),
			}
			compBytes, _ := json.Marshal(completed)
			conn.WriteMessage(websocket.TextMessage, compBytes)
			fmt.Println("Ride completed")
		}
	}
}
