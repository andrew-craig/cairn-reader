# Cairn self-host stack.
#
# Default workflow pulls the pre-built image from GHCR:
#   make up      # start the stack (remote image)
#   make down    # stop the stack
#   make logs    # follow logs
#
# To test local code changes (e.g. to the email-ingest pipeline), build the
# consolidated cairn-selfhost image from source via the docker-compose.build.yml
# override. cmd/selfhost compiles every backend service, so a local build picks
# up changes anywhere in the repo:
#   make build-local   # build the cairn-selfhost image from local source
#   make up-local      # build, then start the stack using the local image

COMPOSE       := docker compose
COMPOSE_BUILD := docker compose -f docker-compose.yml -f docker-compose.build.yml

.PHONY: up down logs build-local up-local

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f

build-local:
	$(COMPOSE_BUILD) build cairn

up-local:
	$(COMPOSE_BUILD) up -d --build
