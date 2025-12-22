# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy the entire services directory to handle go.mod replace directives
COPY users users/
COPY explore explore/

# Set working directory to explore service
WORKDIR /app/explore

# Download dependencies
RUN go mod download

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o recommender ./recommender/cmd/recommender

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/explore/recommender .

EXPOSE 8081

CMD ["./recommender"]
