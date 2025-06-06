.PHONY: all build run test clean swagger-gen \
        dev dev-up dev-down dev-build \
        prod prod-up prod-down prod-build

# --- DEVELOPMENT COMMANDS --- #
# Start development environment (with hot reloading)
dev: dev-down dev-build dev-up

# Start development services
dev-up:
	@echo "Starting development environment..."
	docker compose -f docker-compose.dev.yml up app-minio-api app-minio-drive

# Stop development services
dev-down:
	@echo "Stopping development environment..."
	docker compose -f docker-compose.dev.yml down

# Rebuild development container
dev-build:
	@echo "Building development container..."
	docker compose -f docker-compose.dev.yml build --no-cache app-minio-api

# --- PRODUCTION COMMANDS ---
# Deploy production environment (with build)
prod: prod-down prod-build prod-up

# Start production services
prod-up:
	@echo "Starting production environment..."
	docker compose -f docker-compose.yml up -d

# Start production services with logs
prod-up-logs: prod-build
	@echo "Starting production environment with logs..."
	docker compose -f docker-compose.yml up

# Stop production services
prod-down:
	@echo "Stopping production environment..."
	docker compose -f docker-compose.yml down

# Rebuild production container
prod-build:
	@echo "Building production container..."
	docker compose -f docker-compose.yml build --no-cache app-minio-api

# --- BUILD & TEST COMMANDS ---
# Build the application
build:
	go build -o bin/server cmd/server/main.go

# Run the application locally (without Docker)
run:
	go run cmd/server/main.go

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/

# --- DOCUMENTATION ---
# Generate Swagger documentation
swagger-gen:
	@echo "Generating Swagger documentation..."
	@if ! command -v swag &> /dev/null; then \
		echo "Installing swag..." && \
		go install github.com/swaggo/swag/cmd/swag@latest; \
	fi
	@swag init -g cmd/server/main.go -o ./docs
	@echo "✅ Swagger documentation generated successfully!"
	@echo "🌐 Access Swagger UI at http://localhost:8080/swagger/index.html when the server is running."

# Default target
all: build