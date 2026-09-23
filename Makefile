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
	@echo "  make docker-up         docker compose up --build"
	@echo "  make docker-down       docker compose down"
	@echo "  make docker-build      docker compose build"
	@echo ""
	@echo "No dev target runs both apps together — start backend-run and"
	@echo "frontend-run in two terminals. No lint targets — no linter is"
	@echo "configured for either stack yet."

backend-run:
	cd backend && $(MAKE) run

backend-build:
	cd backend && $(MAKE) build

backend-test:
	cd backend && $(MAKE) test

frontend-run:
	cd frontend && npm run dev

frontend-build:
	cd frontend && npm run build

build: backend-build frontend-build

test: backend-test

docker-up:
	docker compose up --build

docker-down:
	docker compose down

docker-build:
	docker compose build
