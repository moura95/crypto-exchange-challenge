# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git (required for go get)
RUN apk add --no-cache git

# Copy go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Install swag CLI for generating swagger docs
RUN go install github.com/swaggo/swag/cmd/swag@v1.8.12

# Copy source code
COPY . .

# Generate swagger documentation
RUN swag init -g cmd/main.go --output ./docs

# Build binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o server \
    cmd/main.go

# Runtime stage
FROM alpine:latest

# Install CA certificates for HTTPS requests
RUN apk --no-cache add ca-certificates wget

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Copy .env file (optional, can use ENV vars instead)
COPY .env .

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
CMD ["./server"]