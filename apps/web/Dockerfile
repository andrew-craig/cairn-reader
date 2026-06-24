# Build stage — compile the Vite SPA
FROM node:22-alpine AS builder

WORKDIR /app

# Install workspace dependencies. The root manifest + lockfile drive the install;
# both workspace package.json files must be present for npm to link them. The
# install is scoped to web + shared so apps/mobile (also a workspace member, but
# not part of the web image) is neither required nor fetched.
COPY package.json package-lock.json ./
COPY apps/web/package.json apps/web/package.json
COPY apps/shared/package.json apps/shared/package.json
RUN npm ci -w @cairn/web -w @cairn/shared --include-workspace-root

# Build the web app. @cairn/shared is consumed from source via the Vite alias,
# so no separate build step is needed for it.
COPY apps/shared apps/shared
COPY apps/web apps/web
RUN npm run build -w @cairn/web

# Runtime stage — serve the static build with Caddy (the same server the rest of
# the stack already runs, so no extra dependency or config format is introduced).
FROM caddy:2-alpine

COPY apps/web/Caddyfile /etc/caddy/Caddyfile
COPY --from=builder /app/apps/web/dist /srv

EXPOSE 8080
