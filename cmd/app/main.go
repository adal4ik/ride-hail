package main

import (
	"flag"
	"os"
	rideservice "ride-hail/internal/ride-service"
)

func main() {

	rideCmd := flag.NewFlagSet("ride-service", flag.ExitOnError)
	port := rideCmd.Int("port", 8080, "port")

	if len(os.Args) < 2 {
		os.Exit(1)
	}

	switch os.Args[1] {
	case "ride-service":
		rideService := rideservice.New(port)
		rideService.Run()
	}
}
