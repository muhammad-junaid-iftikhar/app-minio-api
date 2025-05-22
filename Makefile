.PHONY: all build run test clean docker-build docker-up docker-down swagger-gen dev dev-down dev-build dev-up

# --- DEV WORKFLOW ---
dev-down:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml down

dev-build:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml build --no-cache app-dev

dev-up:
	docker compose -f docker-compose.dev.yml up app-dev app-minio-drive vector

dev: dev-down dev-build dev-up

# --- PROD WORKFLOW ---
prod-down:
	docker compose -f docker-compose.yml down

prod-build:
	docker compose -f docker-compose.yml build --no-cache app

prod-up:
	docker compose -f docker-compose.yml up app

prod: prod-down prod-build prod-up

all: build

build: go build -o bin/server cmd/server/main.go

run: go run cmd/server/main.go

test: go test ./...

clean: rm -rf bin/

docker-build: docker-compose build

docker-up: docker-compose up -d

docker-down: docker-compose down

swagger-gen:
	@echo "Generating Swagger documentation..."
	@if ! command -v swag &> /dev/null; then \
		echo "Installing swag..." && \
		go install github.com/swaggo/swag/cmd/swag@latest; \
	fi
	@swag init -g cmd/server/main.go -o ./docs
	@echo "Swagger documentation generated successfully!"
	@echo "You can access the Swagger UI at http://localhost:8080/swagger/index.html when the server is running."