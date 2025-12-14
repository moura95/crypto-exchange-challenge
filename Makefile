.PHONY: help build run test lint lint-fix docker-build docker-run docker-stop docker-clean

# Default target
help:
	@echo "Available targets:"
	@echo "  make build            - Build the application"
	@echo "  make run              - Run the application"
	@echo "  make test             - Run unit tests"
	@echo "  make lint             - Run linter"
	@echo "  make lint-fix         - Run linter with auto-fix"
	@echo ""
	@echo "Docker commands:"
	@echo "  make docker-build     - Build Docker image"
	@echo "  make docker-run       - Run with docker-compose"
	@echo "  make docker-stop      - Stop docker-compose"
	@echo "  make docker-clean     - Remove Docker image and containers"

# Build the application
build:
	@echo "Building..."
	go build -o bin/server cmd/main.go

# Run the application
run:
	@echo "Running server..."
	go run cmd/main.go

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	@echo "\nCoverage:"
	go tool cover -func=coverage.out

# Run linter (requires golangci-lint to be installed)
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Error: golangci-lint not installed. Run: brew install golangci-lint" && exit 1)
	golangci-lint run

# Run linter with auto-fix (requires golangci-lint to be installed)
lint-fix:
	@echo "Running linter with auto-fix..."
	@which golangci-lint > /dev/null || (echo "Error: golangci-lint not installed. Run: brew install golangci-lint" && exit 1)
	golangci-lint run --fix

# Docker: Build image
docker-build:
	@echo "Building Docker image..."
	docker build --load -t crypto-exchange:latest .
	@echo "✅ Image built successfully!"
	@docker images crypto-exchange-api:latest

# Docker: Run with docker-compose
docker-run:
	@echo "Starting containers with docker-compose..."
	docker compose up -d
	@echo "✅ Server running at http://localhost:8080"
	@echo "   Health check: http://localhost:8080/health"
	@echo "   Swagger UI: http://localhost:8080/swagger/index.html"

# Docker: Stop containers
docker-stop:
	@echo "Stopping containers..."
	docker compose down

# Docker: Clean up
docker-clean:
	@echo "Removing containers and images..."
	docker compose down --rmi all --volumes --remove-orphans
	@echo "Cleanup complete!"