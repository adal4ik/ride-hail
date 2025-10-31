package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HTTPClient struct {
	client *http.Client
	logger *Logger
}

func NewHTTPClient(logger *Logger) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

func (h *HTTPClient) DoRequest(method, url string, body interface{}, headers map[string]string) ([]byte, error) {
	time.Sleep(HTTPRequestDelay)

	var bodyBytes []byte
	var err error

	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return data, nil
}

// Request/Response models
type DriverRegistrationRequest struct {
	Username      string    `json:"username"`
	Email         string    `json:"email"`
	Password      string    `json:"password"`
	LicenseNumber string    `json:"license_number"`
	VehicleType   string    `json:"vehicle_type"`
	VehicleAttrs  Vehicle   `json:"vehicle_attrs"`
	UserAttrs     UserAttrs `json:"user_attrs"`
}

type Vehicle struct {
	Make  string `json:"make"`
	Model string `json:"model"`
	Color string `json:"color"`
	Plate string `json:"plate"`
	Year  int    `json:"year"`
}

type UserAttrs struct {
	PhoneNumber string `json:"phone"`
}

type LocationRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type StartRideRequest struct {
	RideID         string   `json:"ride_id"`
	DriverLocation Location `json:"driver_location"`
}

type CompleteRideRequest struct {
	RideId        string   `json:"ride_id"`
	FinalLocation Location `json:"final_location"`
	ActualDist    float64  `json:"actual_distance_km"`
	ActualDur     int      `json:"actual_duration_minutes"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	DriverID  string  `json:"driver_id,omitempty"`
}

type RegistrationResponse struct {
	JWT    string `json:"jwt_access"`
	Msg    string `json:"msg"`
	UserID string `json:"driverId"`
}

type OnlineResponse struct {
	Status    string `json:"status"`
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

type StartRideResponse struct {
	RideID    string `json:"ride_id"`
	Status    string `json:"status"`
	StartedAt string `json:"started_at"`
	Message   string `json:"message"`
}

type CompleteRideResponse struct {
	Message        string  `json:"message"`
	RideID         string  `json:"ride_id"`
	Status         string  `json:"status"`
	CompletedAt    string  `json:"completed_at"`
	DriverEarnings float64 `json:"driver_earnings"`
}
