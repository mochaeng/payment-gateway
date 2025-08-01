# Build stage
FROM golang:1.23.2-alpine AS builder

WORKDIR /app

# Install dependencies for potential CGO
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o payment-gateway \
    ./cmd/main.go

# Final stage - minimal image
FROM scratch

# Copy CA certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /app/payment-gateway /payment-gateway

# Expose port
EXPOSE 8080

# Run the application
ENTRYPOINT ["/payment-gateway"]
