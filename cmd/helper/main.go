package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os/signal"
	"syscall"
	"time"

	websocketdto "ride-hail/internal/driver-location-service/core/domain/websocket_dto"
)

type Application struct {
	logger *Logger
	config *Config
}

func NewApplication() *Application {
	return &Application{
		logger: &Logger{},
		config: &Config{
			InitialLocation: Location{
				Latitude:  43.2220,
				Longitude: 76.8512,
			},
			VehicleSpeed: 500.0,
			DriverCredentials: DriverCredentials{
				Username:    "demo-driver",
				Password:    "driver123",
				VehicleType: "ECONOMY",
			},
		},
	}
}

func (app *Application) registerDriver(httpClient *HTTPClient) (*RegistrationResponse, error) {
	credentials := app.config.DriverCredentials
	credentials.Email = fmt.Sprintf("%s@mail.com", app.generateRandomString())
	credentials.LicenseNumber = app.generateRandomString()

	regReq := DriverRegistrationRequest{
		Username:      credentials.Username,
		Email:         credentials.Email,
		Password:      credentials.Password,
		LicenseNumber: credentials.LicenseNumber,
		VehicleType:   credentials.VehicleType,
		VehicleAttrs: Vehicle{
			Make:  "Toyota",
			Model: "Camry",
			Color: "White",
			Plate: "KZ 123 ABC",
			Year:  2020,
		},
		UserAttrs: UserAttrs{
			PhoneNumber: "+7-123-456-78-90",
		},
	}

	app.logger.HTTP("Registering driver...")
	time.Sleep(InitialConnectDelay)

	data, err := httpClient.DoRequest("POST", RegisterURL, regReq, map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil {
		return nil, fmt.Errorf("registering driver: %w", err)
	}

	var response RegistrationResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("unmarshaling registration response: %w", err)
	}

	app.logger.HTTP("Driver created: %s (JWT: %s)", response.UserID, response.JWT[:15]+"...")
	return &response, nil
}

func (app *Application) setupDriverOnline(httpClient *HTTPClient, driverID, jwtToken string) error {
	onlineReq := LocationRequest{
		Latitude:  43.283859,
		Longitude: 76.999909,
	}

	app.logger.HTTP("Setting driver online...")

	url := fmt.Sprintf(BaseURL+DriverOnlinePath, driverID)
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + jwtToken,
	}

	data, err := httpClient.DoRequest("POST", url, onlineReq, headers)
	if err != nil {
		return fmt.Errorf("setting driver online: %w", err)
	}

	var response OnlineResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return fmt.Errorf("unmarshaling online response: %w", err)
	}

	app.logger.HTTP("Driver online: %s", response.Message)
	return nil
}

func (app *Application) messageHandler(driverService *DriverService) func(int, []byte) error {
	return func(messageType int, payload []byte) error {
		var baseMsg websocketdto.WebSocketMessage
		if err := json.Unmarshal(payload, &baseMsg); err != nil {
			app.logger.Warn("Cannot unmarshal base WS message: %v", err)
			return nil
		}

		switch baseMsg.Type {
		case websocketdto.MessageTypeRideOffer:
			var offer websocketdto.RideOfferMessage
			if err := json.Unmarshal(payload, &offer); err != nil {
				app.logger.Error("Cannot unmarshal ride offer: %v", err)
				return nil
			}
			return driverService.HandleRideOffer(offer)
		default:
			app.logger.WebSocket("📨 WS message: %+v", baseMsg)
		}
		return nil
	}
}

func (app *Application) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	httpClient := NewHTTPClient(app.logger)

	// Step 1: Register driver
	registration, err := app.registerDriver(httpClient)
	if err != nil {
		return fmt.Errorf("failed to register driver: %w", err)
	}

	// Step 2: Set driver online
	if err := app.setupDriverOnline(httpClient, registration.UserID, registration.JWT); err != nil {
		return fmt.Errorf("failed to set driver online: %w", err)
	}

	// Step 3: Setup WebSocket connection
	time.Sleep(InitialConnectDelay)
	wsURL := fmt.Sprintf("wss://localhost:3001"+WSDriverPath, registration.UserID)

	driverService := NewDriverService(
		ctx,
		registration.UserID,
		registration.JWT,
		app.config.InitialLocation.Latitude,
		app.config.InitialLocation.Longitude,
		app.logger,
	)

	if err := driverService.wsClient.Connect(wsURL); err != nil {
		return fmt.Errorf("failed to connect to websocket: %w", err)
	}
	defer driverService.wsClient.Close()

	// Step 4: Authenticate and set online via WebSocket
	if err := driverService.Authenticate(); err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	time.Sleep(HTTPRequestDelay)
	if err := driverService.SetOnline(); err != nil {
		return fmt.Errorf("failed to set online via HTTP: %w", err)
	}

	app.logger.Info("🟢 Driver is online and waiting for ride offers...")

	// Step 5: Start reading WebSocket messages
	if err := driverService.wsClient.ReadMessages(app.messageHandler(driverService)); err != nil {
		return fmt.Errorf("websocket read error: %w", err)
	}

	<-ctx.Done()
	app.logger.Info("🛑 Shutting down gracefully")
	return nil
}

func (app *Application) generateRandomString() string {
	chars := []string{"A", "B", "C", "D", "E", "F", "H", "T"}
	result := ""
	for range 10 {
		result += chars[rand.IntN(len(chars))]
	}
	return result
}

func main() {
	app := NewApplication()
	if err := app.Run(); err != nil {
		app.logger.Error("Application failed: %v", err)
	}
}
