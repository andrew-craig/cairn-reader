# Build stage
FROM golang:1.24-alpine AS builder

ENV GOTOOLCHAIN=auto

WORKDIR /app

# Copy root go.mod (needed for module resolution)
COPY go.mod go.sum ./

# Copy all source needed for the consolidated binary
COPY pkg/ pkg/
COPY services/ services/
COPY cmd/selfhost/ cmd/selfhost/

# Set working directory to selfhost module
WORKDIR /app/cmd/selfhost

# Download dependencies
RUN go mod download

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o cairn-selfhost .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Create data directory for JWT keys
RUN mkdir -p /data/keys && chown -R appuser:appuser /data

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/cmd/selfhost/cairn-selfhost .

# Change ownership to non-root user
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

EXPOSE 8080

CMD ["./cairn-selfhost"]
