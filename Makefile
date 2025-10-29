# Makefile for Ride Hail System

# Binary output directory
BIN_DIR := bin
BIN_NAME := rh

# Service names
SERVICES := auth-service ride-service driver-location-service admin-service

# Default target when no specific target is passed
.PHONY: all
all: build

# Build the Go binary
.PHONY: b
b:
	go build -o $(BIN_DIR)/$(BIN_NAME) ./cmd/app/main.go

# Run Docker Compose
.PHONY: u
u:
	docker-compose up --build

# Stop and remove Docker containers
.PHONY: d
d:
	docker-compose down -v

# Exec into Postgres container
.PHONY: a
a:
	docker exec -it ridehail-postgres psql -U ridehail_user -W ridehail_db

# Run the selected service (single service mode)
.PHONY: run
run:
	@echo "Please specify a service to run."
	@echo "Usage: make run SERVICE=<service>"
	@echo "Example: make run SERVICE=ride-service"

# Run the individual service based on the provided argument
.PHONY: run-service
run-service:
ifeq ($(SERVICE),)
	@echo "Error: No service specified. Use 'SERVICE=<service_name>'"
	@exit 1
endif
	@echo "Starting service: $(SERVICE)..."
	$(BIN_DIR)/$(BIN_NAME) --mode=$(SERVICE)

# Run all services in parallel (example: running all services in the background)
.PHONY: run-all
run-all:
	@echo "Running all services..."
	$(BIN_DIR)/$(BIN_NAME) --mode=auth-service &
	$(BIN_DIR)/$(BIN_NAME) --mode=ride-service &
	$(BIN_DIR)/$(BIN_NAME) --mode=driver-location-service &
	$(BIN_DIR)/$(BIN_NAME) --mode=admin-service &
	wait

# Help target to show usage
.PHONY: help
help:
	@echo "Usage: make [target]"
	@echo "Available targets:"
	@echo "  b           - Build the Go binary"
	@echo "  u           - Start services using docker-compose"
	@echo "  d           - Stop and remove services using docker-compose"
	@echo "  a           - Exec into the Postgres container"
	@echo "  run         - Start a specific service (SERVICE=<service_name>)"
	@echo "  run-all     - Start all services concurrently"
	@echo "  help        - Display this help message"

