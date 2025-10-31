package rideservice

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"sync"
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
	var wg sync.WaitGroup
	// Context Declaration
	signalCtx, close := signal.NotifyContext(ctx, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer close()
	// Connecting to Database
	database, err := db.ConnectDB(signalCtx, cfg.DB, mylog)
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
	defer broker.Close()
	log.Info("Successfully connected to message broker")

	// Declaring Consumer
	consumer := bm.NewConsumer(signalCtx, broker, mylog)
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
	wg.Add(1)
	distributor := services.NewDistributor(signalCtx, req, statusMsgs, wbManager, broker, service.DriverService, mylog)
	go func() {
		defer wg.Done()
		if err := distributor.MessageDistributor(); err != nil {
			mylog.Error("Message distributor encountered an error", err)
		}
	}()
	log.Info("Distribur successfully setted up and ready to work")

	cert, err := tls.LoadX509KeyPair(cfg.App.CertPath, cfg.App.CertKeyPath)
	if err != nil {
		return fmt.Errorf("failed to load TLS cert/key: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	// Defining the rounter
	mux := myhttp.Router(handler, cfg)
	httpServer := &http.Server{
		Addr:      fmt.Sprintf(":%v", cfg.Srv.DriverLocationServicePort),
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	// Running server
	wg.Add(1)
	runErrCh := make(chan error, 1)
	go func() {
		defer wg.Done()
		log.Info("Server is starting on port :" + cfg.Srv.DriverLocationServicePort)
		if err := httpServer.ListenAndServeTLS(cfg.App.CertPath, cfg.App.CertKeyPath); err != nil && err != http.ErrServerClosed {
			runErrCh <- err
		}
	}()
	log.Info("Server is started successfully")

	// Listening for channels
	select {
	case <-signalCtx.Done():
		log.Info("Shutdown signal received")
		log.Info("Shutting down gracefully......")
		if err := httpServer.Shutdown(context.Background()); err != nil {
			log.Error("HTTP server shutdown failed", err)
		}
		// Waiting
		log.Info("waiting for workers......")
		wg.Wait()
		log.Info("All workers are done")
		// Drivers
		err := service.DriverService.GracefullShutdown(context.Background())
		if err != nil {
			log.Error("Failed to shutdown drivers", err)
		} else {
			log.Info("All drivers are offline now and all sessions are completed")
		}
	case err = <-runErrCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("Server failed unexpectedly", err)
		}
	}

	return err
}
