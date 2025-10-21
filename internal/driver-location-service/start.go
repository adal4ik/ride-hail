package rideservice

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"ride-hail/internal/config"
	"ride-hail/internal/driver-location-service/adapters/driven/bm"
	"ride-hail/internal/driver-location-service/adapters/driven/db"
	"ride-hail/internal/driver-location-service/adapters/driver/myhttp"
	"ride-hail/internal/driver-location-service/adapters/driver/myhttp/handlers"
	"ride-hail/internal/driver-location-service/core/services"
	"ride-hail/internal/mylogger"
	"syscall"
)

func Execute(ctx context.Context, mylog mylogger.Logger, cfg *config.Config) error {
	newCtx, close := signal.NotifyContext(ctx, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer close()
	database, err := db.ConnectDB(newCtx, cfg.DB, mylog)
	if err != nil {
		return err
	}
	defer database.Close()
	repository := db.New(database)
	broker, err := bm.New(ctx, *cfg.RabbitMq, mylog)
	if err != nil {
		return nil
	}
	service := services.New(repository, &mylog, broker)
	handler := handlers.New(service, mylog)
	mux := myhttp.Router(handler, cfg)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%v", cfg.Srv.DriverLocationServicePort),
		Handler: mux,
	}

	// Running server
	runErrCh := make(chan error, 1)
	go func() {
		mylog.Action("Server initialized").Info("server is starting on port :" + cfg.Srv.DriverLocationServicePort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			runErrCh <- err
		}
	}()

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
