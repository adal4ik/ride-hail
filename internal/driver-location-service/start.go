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
	"ride-hail/internal/driver-location-service/adapters/driven/ws"
	"ride-hail/internal/driver-location-service/adapters/driver/myhttp"
	"ride-hail/internal/driver-location-service/adapters/driver/myhttp/handlers"
	"ride-hail/internal/driver-location-service/core/services"
	"ride-hail/internal/mylogger"
)

func Execute(ctx context.Context, mylog mylogger.Logger, cfg *config.Config) error {
	log := mylog.Action("Execute")

	// Context Declaration
	newCtx, close := signal.NotifyContext(ctx, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer close()

	// Connecting to Database
	database, err := db.ConnectDB(newCtx, cfg.DB, mylog)
	if err != nil {
		log.Error("Database connection failed: ", err)
		return err
	}
	defer database.Close()
	log.Info("Database connection established successufuly")

	// Declaring Broker
	broker, err := bm.New(ctx, *cfg.RabbitMq, mylog)
	if err != nil {
		log.Error("Broker connection failed: ", err)
		return err
	}

	// Declaring channels for ride offers and driver responses
	// driverResponses := make(map[string]chan dto.DriverResponse)
	// messageDriver := make(map[string]chan dto.DriverRideOffer)

	// Declaring Consumer
	consumer := bm.NewConsumer(newCtx, broker, mylog)
	req, statusMsgs, err := consumer.ListenAll()
	if err != nil {
		log.Error("Failed to subscribe for messages", err)
		return err
	}
	log.Info("Consumer is listenning for the messages")

	// Declaring service components
	repository := db.New(database)
	wbManager := ws.NewWebSocketManager()
	service := services.New(repository, mylog, broker, cfg.App.PublicJwtSecret)
	handler := handlers.New(service, mylog, wbManager)
	log.Info("All driver-location components are declared")

	// Creating the distributor
	distributor := services.NewDistributor(newCtx, req, statusMsgs, wbManager, broker, service.DriverService, mylog)
	go func() {
		if err := distributor.MessageDistributor(); err != nil {
			mylog.Error("Message distributor encountered an error", err)
		}
	}()
	log.Info("Distribur successfully setted up and ready to work")

	// Defining the rounter
	mux := myhttp.Router(handler, cfg)
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%v", cfg.Srv.DriverLocationServicePort),
		Handler: mux,
	}

	// Running server
	runErrCh := make(chan error, 1)
	go func() {
		log.Info("Server is starting on port :" + cfg.Srv.DriverLocationServicePort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			runErrCh <- err
		}
	}()
	mylog.Info("Server is started successfully")

	// Gracefull Shutdown
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
