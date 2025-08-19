SHELL := /bin/bash

APP?=api-gateway
TAG?=dev

.PHONY: help
help:
	@echo "Targets:"
	@echo "  make run             - Start dev stack with Docker Compose"
	@echo "  make stop            - Stop dev stack"
	@echo "  make clean           - Stop dev stack and clean the volume"
	@echo "  make logs APP=name   - Tail logs for a service"
	@echo "  make test            - Run all Go tests"
	@echo "  make build APP=name  - Build one service locally"
	@echo "  make tidy            - go mod tidy"

.PHONY: run
run:
	docker compose -f docker-compose.dev.yml up --build

.PHONY: stop
stop:
	docker compose -f docker-compose.dev.yml down

.PHONY: clean
clean:
	docker compose -f docker-compose.dev.yml down -v


.PHONY: logs
logs:
	docker compose -f docker-compose.dev.yml logs -f $(APP)

.PHONY: test
test:
	go test ./... -count=1

.PHONY: build
build:
	cd services/$(APP) && go build -o ../../bin/$(APP)

.PHONY: tidy
tidy:
	go mod tidy
