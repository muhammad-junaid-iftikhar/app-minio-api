# Stage 1: Build the Go binary
FROM golang:1.23.0-alpine3.19 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Install swag for Swagger doc generation
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.2

COPY . .

# Generate Swagger docs automatically
RUN /go/bin/swag init -g cmd/server/main.go -o ./docs

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/server ./cmd/server/main.go

# --- Dev Stage: for local development with live reload ---
FROM golang:1.23.0-alpine3.19 AS dev

WORKDIR /app

# Install build tools and dependencies
RUN apk add --no-cache git

# Install air for live reload
RUN go install github.com/cosmtrek/air@v1.49.0

# Install swag globally
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.2

# Install golang-migrate for database migrations
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Ensure the PATH includes Go binaries
ENV PATH="/go/bin:${PATH}"

# Copy dependency files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application
COPY . .

# Verify swag is installed and working
RUN swag --version

# Generate initial Swagger docs
RUN swag init -g ./cmd/server/main.go -o ./docs --parseDependency --parseInternal

# Set environment variables for development
ENV PATH="/go/bin:${PATH}" \
    APP_ENV=development \
    SERVER_PORT=8080

# Expose the application port
EXPOSE 8080

# Command to run the application with live reload
CMD ["air", "-c", ".air.toml"]

# --- Production Stage: Minimal runtime image ---
FROM alpine:latest AS prod

WORKDIR /app

# Install runtime dependencies
RUN apk --no-cache add ca-certificates

# Copy the binary from builder
COPY --from=builder /bin/server /app/server

# Copy config and docs
COPY --from=builder /app/config /app/config
COPY --from=builder /app/docs /app/docs

# Make the binary executable
RUN chmod +x /app/server

# Expose the port the app runs on
EXPOSE 8080

# Command to run the application
CMD ["/app/server"]
