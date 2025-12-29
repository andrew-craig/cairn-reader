# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Set working directory
WORKDIR /build

# Copy go mod files from root
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY fetcher/ ./fetcher/
COPY pkg/ ./pkg/

# Build the application
WORKDIR /build/fetcher
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ../rss-fetcher-service ./cmd/fetcher

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/rss-fetcher-service .

# Copy migrations (if needed at runtime)
COPY fetcher/migrations ./migrations

# Change ownership to non-root user
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8081

# Run the application
CMD ["./rss-fetcher-service"]
