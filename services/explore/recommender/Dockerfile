# Build stage
FROM golang:1.25-alpine AS builder

# Set GOTOOLCHAIN to auto to allow using newer Go versions
ENV GOTOOLCHAIN=auto

WORKDIR /app

# Copy necessary directories to handle go.mod replace directives
COPY go.mod go.sum ./
COPY services/users services/users/
COPY services/explore services/explore/
COPY pkg pkg/

# Set working directory to explore service
WORKDIR /app/services/explore

# Download dependencies
RUN go mod download

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o explore_recommender ./recommender/cmd/explore_recommender

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/services/explore/explore_recommender .

# Copy migrations
COPY --from=builder /app/services/explore/recommender/migrations ./migrations

# Change ownership to non-root user
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

EXPOSE 8081

CMD ["./explore_recommender"]
