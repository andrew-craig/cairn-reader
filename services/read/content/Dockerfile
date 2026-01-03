# Build stage
FROM golang:1.24-alpine AS builder

# Set GOTOOLCHAIN to auto to allow using newer Go versions
ENV GOTOOLCHAIN=auto

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /build/services/read

# Copy read service and pkg directory
COPY services/read/ ./
COPY pkg/ ../../pkg/

# Download dependencies
RUN go mod download

# Build the application
WORKDIR /build/services/read/content
RUN CGO_ENABLED=0 GOOS=linux go build -o /build/content-service ./cmd/content

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/content-service .

# Change ownership to non-root user
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Run the application
CMD ["./content-service"]
