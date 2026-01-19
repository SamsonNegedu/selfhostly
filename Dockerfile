# Multi-stage build optimized for ARM64 (Raspberry Pi)
FROM golang:1.21-alpine as builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application for ARM64
RUN CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -a -installsuffix cgo -o automaton cmd/server/main.go

# Production stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache sqlite

# Create app directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/automaton .

# Create directories
RUN mkdir -p /app/data /app/apps

# Expose port
EXPOSE 8080

# Run the application
CMD ["./automaton"]