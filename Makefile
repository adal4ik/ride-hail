BIN_DIR := bin
BIN_NAME := rh
SERVICES := auth-service ride-service driver-location-service admin-service

.PHONY: all
all: b

.PHONY: b
b: 
	go build -o $(BIN_DIR)/$(BIN_NAME) ./cmd/app/main.go

.PHONY: u
u:
	docker-compose up --build

.PHONY: d
d:
	docker-compose down -v

.PHONY: a
a:
	docker exec -it ridehail-postgres psql -U ridehail_user -W ridehail_db

.PHONY: run
run:
ifeq ($(s),)
	@echo "Error: No service specified"
	@exit 1
endif
	$(BIN_DIR)/$(BIN_NAME) --mode=$(s)

.PHONY: run-all
run-all:
	$(foreach service, $(SERVICES), gnome-terminal --tab -- bash -c "./bin/rh --mode=$(service); exec bash" &)
	wait

.PHONY: kill-rh
kill-rh:
	pkill rh

.PHONY: help
help:
	@echo "Targets: b, u, d, a, run, run-all, run-all-tmux, help"

.PHONY: helper
helper:
	go run ./cmd/helper/main.go