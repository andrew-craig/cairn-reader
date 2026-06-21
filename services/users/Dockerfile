# Build stage
FROM golang:1.24-alpine AS builder

# Set GOTOOLCHAIN to auto to allow using newer Go versions
ENV GOTOOLCHAIN=auto

WORKDIR /app

# Copy root go.mod and go.sum for dependency resolution
COPY go.mod go.sum ./

# Copy user service and shared pkg
COPY services/users services/users/
COPY pkg pkg/

# Set working directory to user service
WORKDIR /app/services/users

# Download dependencies
RUN go mod download

# Build the binary
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-X 'main.version=${VERSION}'" \
    -o user-service ./cmd/user-service

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/services/users/user-service .

# Copy migrations (needed for runtime migration execution)
COPY --from=builder /app/services/users/migrations ./migrations

# Change ownership to non-root user
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

EXPOSE 8080

CMD ["./user-service"]
