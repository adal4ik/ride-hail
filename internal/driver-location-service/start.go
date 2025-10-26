package rideservice

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"

	"ride-hail/internal/config"
	"ride-hail/internal/driver-location-service/adapters/driven/bm"
	"ride-hail/internal/driver-location-service/adapters/driven/db"
	"ride-hail/internal/driver-location-service/adapters/driver/myhttp"
	"ride-hail/internal/driver-location-service/adapters/driver/myhttp/handlers"
	"ride-hail/internal/driver-location-service/core/domain/dto"
	"ride-hail/internal/driver-location-service/core/services"
	"ride-hail/internal/mylogger"
)

func Execute(ctx context.Context, mylog mylogger.Logger, cfg *config.Config) error {
	newCtx, close := signal.NotifyContext(ctx, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer close()

	mylog.Action("Starting Driver Location Service").Info("Initializing components")
	mylog.Action("Connecting to Database").Info("Connecting to the database")
	database, err := db.ConnectDB(newCtx, cfg.DB, mylog)
	if err != nil {
		return err
	}
	mylog.Action("Database connected").Info("Database connection established")
	defer database.Close()
	mylog.Action("Setting up components").Info("Setting up repository, broker, service, and handlers")
	repository := db.New(database)
	broker, err := bm.New(ctx, *cfg.RabbitMq, mylog)
	if err != nil {
		return err
	}

	// Declaring channels for ride offers and driver responses
	rideOffers := make(chan dto.RideDetails, 100)
	driverResponses := make(map[string]chan dto.DriverResponse)
	messageDriver := make(map[string]chan dto.DriverRideOffer)

	// Creating the distributor
	distributor := services.NewDistributor(newCtx, messageDriver, &rideOffers, driverResponses, broker)

	// Start the message distributor in a separate goroutine
	go func() {
		if err := distributor.MessageDistributor(); err != nil {
			mylog.Error("Message distributor encountered an error", err)
		}
	}()

	mylog.Action("Broker connected").Info("Message broker connection established")
	service := services.New(repository, mylog, broker)
	handler := handlers.New(service, mylog, messageDriver, driverResponses)
	mux := myhttp.Router(handler, cfg)
	mylog.Action("Components set up").Info("All components are set up successfully")
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%v", cfg.Srv.DriverLocationServicePort),
		Handler: mux,
	}

	mylog.Action("HTTP server configured").Info("HTTP server is configured and ready to start")

	// RabbitMq consumer setup
	consumer := bm.NewConsumer(newCtx, broker, mylog, rideOffers)
	if err := consumer.SubscribeForMessages(); err != nil {
		mylog.Error("Failed to subscribe for messages", err)
		return err
	}
	// Running server
	mylog.Action("Starting server").Info("Starting HTTP server")
	runErrCh := make(chan error, 1)
	go func() {
		mylog.Action("Server initialized").Info("server is starting on port :" + cfg.Srv.DriverLocationServicePort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			runErrCh <- err
		}
	}()

	mylog.Action("Server started").Info("HTTP server started successfully and is running")
	// Wait for signal or server crash
	select {
	case <-newCtx.Done():
		mylog.Info("Shutdown signal received")
		// return GracefullShutDown(context.Background())
	case err := <-runErrCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			mylog.Error("Server failed unexpectedly", err)
			return err
		}
		mylog.Info("Server exited normally")
		return nil
	}
	return nil
}
