# Build stage
FROM golang:1.23-alpine AS builder

# Set GOTOOLCHAIN to auto to allow using newer Go versions
ENV GOTOOLCHAIN=auto

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Set working directory
WORKDIR /build

# Copy go mod files from root
COPY services/read/go.mod services/read/go.sum ./

# Copy pkg directory for dependencies (placed at /pkg to match replace directive)
COPY pkg/ /pkg/

# Download dependencies
RUN go mod download

# Copy source code
COPY services/read/fetcher/ ./fetcher/

# Build the application
WORKDIR /build/fetcher
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ../ingest-rss ./cmd/ingest_rss

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/ingest-rss .

# Copy migrations (if needed at runtime)
COPY services/read/fetcher/migrations ./migrations

# Change ownership to non-root user
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8081

# Run the application
CMD ["./ingest-rss"]
