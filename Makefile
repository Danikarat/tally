.PHONY: up down db-logs db-shell migrate-up migrate-down migrate-version build run test fmt clean

# Load .env if present
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# Default target
all: test build

# Start local services with Docker Compose
up:
	docker compose up -d

# Stop local services
down:
	docker compose down

# Stream PostgreSQL logs
db-logs:
	docker compose logs -f postgres

# Open interactive psql shell inside PostgreSQL container
db-shell:
	docker compose exec -it postgres psql -U $(or $(DB_USER),tally) -d $(or $(DB_NAME),tally_dev)

# Apply all pending database migrations
migrate-up:
	go run ./cmd/migrate -up

# Rollback all database migrations
migrate-down:
	go run ./cmd/migrate -down

# Print current database migration version
migrate-version:
	go run ./cmd/migrate -version

# Build Go API binary
build:
	go build -v -o bin/api ./cmd/api

# Run Go API locally
run:
	go run ./cmd/api

# Run unit and integration tests with data race detection
test:
	go test -v -race ./...

# Format all Go source files
fmt:
	gofmt -s -w .

# Clean build artifacts
clean:
	rm -rf bin/
