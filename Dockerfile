# Multi-stage build optimized for ARM64 (Raspberry Pi)
FROM golang:1.21-alpine as builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application for ARM64 (no CGO needed with modernc.org/sqlite)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o selfhostly cmd/server/main.go

# Production stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Create app directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/selfhostly .

# Create directories
RUN mkdir -p /app/data /app/apps

# Expose port
EXPOSE 8080

# Run the application
CMD ["./selfhostly"]
