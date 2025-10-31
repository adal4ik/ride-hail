package main

import "time"

// ANSI color codes
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Cyan   = "\033[36m"
	Gray   = "\033[90m"
)

// Configuration constants for rate limiting
const (
	LocationUpdateInterval = 3 * time.Second
	DBWriteDelay           = 100 * time.Millisecond
	HTTPRequestDelay       = 200 * time.Millisecond
	InitialConnectDelay    = 1 * time.Second
)

// API endpoints
const (
	BaseURL            = "https://localhost:3001"
	RegisterURL        = "https://localhost:3010/driver/register"
	DriverOnlinePath   = "/drivers/%s/online"
	DriverStartPath    = "/drivers/%s/start"
	DriverCompletePath = "/drivers/%s/complete"
	WSDriverPath       = "/ws/drivers/%s"
)

type Config struct {
	InitialLocation   Location
	VehicleSpeed      float64
	DriverCredentials DriverCredentials
}

type DriverCredentials struct {
	Username      string
	Email         string
	Password      string
	LicenseNumber string
	VehicleType   string
}
