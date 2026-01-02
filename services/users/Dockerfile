# Build stage
FROM golang:1.23-alpine AS builder

# Set GOTOOLCHAIN to auto to allow using newer Go versions
ENV GOTOOLCHAIN=auto

WORKDIR /app

# Copy user service and shared pkg
COPY services/users services/users/
COPY pkg pkg/

# Set working directory to user service
WORKDIR /app/services/users

# Download dependencies
RUN go mod download

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o user-service ./cmd/user-service

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/services/users/user-service .

# Copy migrations (needed for runtime migration execution)
COPY --from=builder /app/services/users/migrations ./migrations

EXPOSE 8080

CMD ["./user-service"]
