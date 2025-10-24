package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	authservice "ride-hail/internal/auth-service"
	"ride-hail/internal/config"
	"ride-hail/internal/mylogger"

	adminservice "ride-hail/internal/admin-service"

	driverlocationservice "ride-hail/internal/driver-location-service"

	rideservice "ride-hail/internal/ride-service"
)

var (
	ErrModeFlag       = errors.New("ErrModeFlag")
	ErrUnknownService = errors.New("ErrUnknownService")
)

func main() {
	cfg, err := config.New()
	// Initialize structured JSON logger
	appLogger, err := mylogger.New(cfg.Log.Level)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	appLogger.Action("ride_hail_system_started").Info("Ride Hail System starting up")

	// Global flags for selecting the service mode
	fs := flag.NewFlagSet("main", flag.ExitOnError)
	mode := fs.String("mode", "", "service to run: ride-service | driver-service | admin-service")

	// Only parse the first few args for `--mode`, the rest go to the service
	args := os.Args[1:]
	modeArgs := []string{}
	for i, arg := range args {
		if strings.HasPrefix(arg, "--mode") || strings.HasPrefix(arg, "-mode") {
			modeArgs = args[:i+1]
			break
		}
	}

	// parse mode
	if err := fs.Parse(modeArgs); err != nil {
		appLogger.Action("ride_hail_system_failed").Error("Failed to parse flags", err)
		help(fs)
		return
	}

	if *mode == "" {
		appLogger.Action("ride_hail_system_failed").Error("Failed to start ride hail system", ErrModeFlag)
		help(fs)
		return
	}

	// Remaining args after parsing --mode
	// remainingArgs := args[len(modeArgs):]

	ctx := context.Background()
	switch *mode {

	case "admin-service", "as":
		l := appLogger.With("service", "admin-service")
		l.Action("admin_service_started").Info("Admin Service starting up")
		if err := adminservice.Execute(ctx, l, cfg); err != nil {
			l.Action("admin_service_failed").Error("Error in admin-service", err)
		}
		l.Action("admin_service_completed").Info("Admin Service shut down successfully")
	case "driver-location-service", "dls":
		driverServiceLogger := appLogger.With("service", "driver-location-service")
		driverServiceLogger.Action("driver_location_service_started").Info("Driver and Location service starting up")
		if err := driverlocationservice.Execute(ctx, driverServiceLogger, cfg); err != nil {
			driverServiceLogger.Action("driver_location_service_failed").Error("Error in driver-location-service", err)
		}
		driverServiceLogger.Action("driver_location_service_completed").Info("Driver and Location service shut down successfully")
	case "ride-service", "rs":
		l := appLogger.With("service", "ride-service")
		l.Info("Ride Service starting up")
		if err := rideservice.Execute(ctx, l, cfg); err != nil {
			l.Error("Error in ride-service", err)
		}
		l.Info("Ride Service shut down successfully")
	case "auth-service", "au":
		l := appLogger.With("service", "auth-service")
		l.Info("Auth Service starting up")
		if err := authservice.Execute(ctx, l, cfg); err != nil {
			l.Error("Error in auth-service", err)
		}
		l.Info("Auth Service shut down successfully")
	default:
		appLogger.Action("ride_hail_system_failed").Error("Failed to start ride hail system", ErrUnknownService)
		help(fs)
	}
}

func help(fs *flag.FlagSet) {
	fmt.Println("\nRide Hail System - Usage:")
	fs.PrintDefaults()
	fmt.Println("\nAvailable Services:")
	fmt.Println("  ride-service (rs)     - Orchestrates ride lifecycle and passenger interactions")
	fmt.Println("  driver-service (ds)   - Handles driver operations, matching, and location tracking")
	fmt.Println("  admin-service (as)    - Provides monitoring, analytics, and system oversight")
	fmt.Println("  auth-service (au)     - User logic")
	fmt.Println("\nExamples:")
	fmt.Println("  bin/rh --mode=ride-service --port=3000")
	fmt.Println("  bin/rh --mode=driver-service --port=3001")
	fmt.Println("  bin/rh --mode=admin-service --port=3004")
	fmt.Println("  bin/rh --mode=auth-service --port=3010")
	fmt.Println("\nConfiguration:")
	fmt.Println("  Use environment variables or config files for database, RabbitMQ, and service settings")
}
