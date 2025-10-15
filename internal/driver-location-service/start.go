package driverlocationservice

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"ride-hail/internal/config"
	"ride-hail/internal/mylogger"
)

func Run(ctx context.Context, l mylogger.Logger, cfg *config.Config) error {
	shutdown, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	<-shutdown.Done()
	fmt.Println("Gracefully shutting down...")
	return nil
}
