# Stage 1: Build the Go binary
FROM golang:1.23-alpine AS builder

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
FROM golang:1.23-alpine AS dev

WORKDIR /app

# Install air for live reload
RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.2
ENV PATH="/go/bin:${PATH}"
COPY . .

EXPOSE 8080

CMD ["/go/bin/air"]

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
