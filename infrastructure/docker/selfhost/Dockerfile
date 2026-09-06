# Lockfile stage — derive a web+shared-only view of the workspace lockfile.
# apps/mobile is also a workspace member, so the shared root package-lock.json
# changes on every mobile-only dependency bump. Feeding that straight into the
# web install makes a mobile change bust the npm ci + vite build cache layers.
# Dropping apps/mobile from the workspace list and regenerating the lockfile
# offline (no network, no node_modules) yields output that is byte-identical
# unless a dependency web or shared actually uses moves — so the COPY --from
# below is a BuildKit content-hash cache hit for mobile-only changes.
# npm 11+ prunes the orphaned workspace entry fully, hence node:24 here.
FROM node:24-alpine AS lockfile
WORKDIR /app
COPY package.json package-lock.json ./
COPY apps/web/package.json apps/web/package.json
COPY apps/shared/package.json apps/shared/package.json
RUN node -e "const fs=require('fs');const p=require('./package.json');p.workspaces=p.workspaces.filter(w=>w!=='apps/mobile');fs.writeFileSync('package.json',JSON.stringify(p,null,2)+'\n')" \
    && npm install --package-lock-only --offline

# Web build stage — compile the Vite SPA to static assets that the Go binary serves
FROM node:22-alpine AS web-builder

WORKDIR /app

# Install workspace dependencies from the pruned lockfile. Both workspace
# package.json files must be present for npm to link them.
COPY --from=lockfile /app/package.json /app/package-lock.json ./
COPY apps/web/package.json apps/web/package.json
COPY apps/shared/package.json apps/shared/package.json
RUN npm ci

# @cairn/shared is consumed from source via the Vite alias, so no build step for it.
COPY apps/shared apps/shared
COPY apps/web apps/web
RUN npm run build -w @cairn/web

# Build stage
FROM golang:1.25-alpine AS builder

ENV GOTOOLCHAIN=auto

# Build version, injected into the binary via -ldflags. Defaults to "dev";
# the Build Selfhost workflow passes the resolved image version (e.g. 1.2.3).
ARG VERSION=dev

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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-X main.version=${VERSION}" -o cairn-selfhost .

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

# Copy the built web UI. The binary serves it from WEB_DIR (default /app/web)
# as a fallback for any non-API route.
COPY --from=web-builder /app/apps/web/dist ./web

# Change ownership to non-root user
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

EXPOSE 8099

CMD ["./cairn-selfhost"]
