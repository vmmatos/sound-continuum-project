.PHONY: help backend-run backend-build backend-test frontend-run frontend-build build test docker-up docker-down docker-build

.DEFAULT_GOAL := help

help:
	@echo "Sound Continuum — common developer commands"
	@echo ""
	@echo "  make backend-run       Run the backend (go run ./cmd/server)"
	@echo "  make backend-build     Build the backend binary"
	@echo "  make backend-test      Run backend tests (go test ./...)"
	@echo "  make frontend-run      Run the frontend dev server (npm run dev)"
	@echo "  make frontend-build    Type-check and build the frontend"
	@echo "  make build             backend-build + frontend-build"
	@echo "  make test              backend-test (no frontend test runner yet)"
	@echo "  make docker-up         docker compose -f dev/docker-compose.yml up --build"
	@echo "  make docker-down       docker compose -f dev/docker-compose.yml down"
	@echo "  make docker-build      docker compose -f dev/docker-compose.yml build"
	@echo ""
	@echo "No dev target runs both apps together — start backend-run and"
	@echo "frontend-run in two terminals. No lint targets — no linter is"
	@echo "configured for either stack yet."
	@echo ""
	@echo "backend-run/frontend-run source dev/.env (and dev/.secrets.env for"
	@echo "the backend) if present — see README for local config setup."

backend-run:
	set -a; [ -f dev/.env ] && . dev/.env; [ -f dev/.secrets.env ] && . dev/.secrets.env; set +a; cd backend && $(MAKE) run

backend-build:
	cd backend && $(MAKE) build

backend-test:
	cd backend && $(MAKE) test

frontend-run:
	set -a; [ -f dev/.env ] && . dev/.env; set +a; cd frontend && npm run dev

frontend-build:
	cd frontend && npm run build

build: backend-build frontend-build

test: backend-test

docker-up:
	docker compose -f dev/docker-compose.yml up --build

docker-down:
	docker compose -f dev/docker-compose.yml down

docker-build:
	docker compose -f dev/docker-compose.yml build
