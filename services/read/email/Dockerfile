# Build stage
FROM golang:1.25-alpine AS builder

# Set GOTOOLCHAIN to auto to allow using newer Go versions
ENV GOTOOLCHAIN=auto

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Set working directory
WORKDIR /build

# Copy go mod files and shared packages
COPY services/read/email/go.mod services/read/email/go.sum ./services/read/email/
COPY pkg/ ./pkg/

# Download dependencies
WORKDIR /build/services/read/email
RUN go mod download

# Copy source code
WORKDIR /build
COPY services/read/email/ ./services/read/email/

# Build the application
WORKDIR /build/services/read/email
RUN CGO_ENABLED=0 GOOS=linux go build -o /build/email-ingest ./cmd/email_ingest

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/email-ingest .

# Copy migrations
COPY services/read/email/migrations ./migrations

# Change ownership to non-root user
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8087

# Run the application
CMD ["./email-ingest"]
