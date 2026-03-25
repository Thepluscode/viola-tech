.PHONY: help build run test migrate-up migrate-down migrate-status migrate-create dev clean

help:
	@echo "Gateway API - Available targets:"
	@echo ""
	@echo "  build          - Build the gateway-api binary"
	@echo "  run            - Run the gateway-api server"
	@echo "  test           - Run tests"
	@echo "  migrate-up     - Apply all pending migrations"
	@echo "  migrate-down   - Rollback the last migration"
	@echo "  migrate-status - Show migration status"
	@echo "  migrate-create - Create a new migration (usage: make migrate-create NAME=add_users)"
	@echo "  dev            - Run in development mode (auto-reload)"
	@echo "  clean          - Clean build artifacts"

build:
	@echo "Building gateway-api..."
	@go build -o bin/gateway-api ./cmd/gateway-api
	@echo "✓ Built: bin/gateway-api"

run: build
	@echo "Starting gateway-api..."
	@./bin/gateway-api

test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out

migrate-up:
	@./scripts/migrate.sh up

migrate-down:
	@./scripts/migrate.sh down

migrate-status:
	@./scripts/migrate.sh status

migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-create NAME=add_users"; \
		exit 1; \
	fi
	@./scripts/migrate.sh create $(NAME)

dev:
	@echo "Running in development mode..."
	@echo "Ensure PostgreSQL and Kafka are running (make dev in root)"
	@go run ./cmd/gateway-api/main.go

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f coverage.out
	@echo "✓ Cleaned"
