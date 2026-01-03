# Build stage
FROM golang:1.24-alpine AS builder

# Set GOTOOLCHAIN to auto to allow using newer Go versions
ENV GOTOOLCHAIN=auto

WORKDIR /app

# Copy necessary directories to handle go.mod replace directives
COPY services/users services/users/
COPY services/explore services/explore/
COPY pkg pkg/

# Set working directory to explore service
WORKDIR /app/services/explore

# Download dependencies
RUN go mod download

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o explore_fetcher ./fetcher/cmd/explore_fetcher

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/services/explore/explore_fetcher .

# Copy migrations
COPY --from=builder /app/services/explore/fetcher/migrations ./migrations

EXPOSE 8080

CMD ["./explore_fetcher"]
