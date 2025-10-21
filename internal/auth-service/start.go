package authservice

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"ride-hail/internal/auth-service/adapters/driver/myhttp"
	"ride-hail/internal/config"
	"ride-hail/internal/mylogger"
	"syscall"
)

func Execute(ctx context.Context, mylog mylogger.Logger, cfg *config.Config) error {
	newCtx, close := signal.NotifyContext(ctx, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer close()

	server := myhttp.NewServer(newCtx, ctx, mylog, cfg)

	// Run server in goroutine
	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- server.Run()
	}()

	// Wait for signal or server crash
	select {
	case <-newCtx.Done():
		mylog.Info("Shutdown signal received")
		return server.Stop(context.Background())
	case err := <-runErrCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			mylog.Error("Server failed unexpectedly", err)
			return err
		}
		mylog.Info("Server exited normally")
		return nil
	}
}
