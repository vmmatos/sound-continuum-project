# Sound Continuum

A weekly music curation project connecting timeless classics, contemporary
music, and emerging artists through intentional musical bridges — a
continuous musical journey rather than disconnected weekly playlists.

Editorial philosophy: see [`docs/manifesto.md`](docs/manifesto.md).

## Status

MVP under development. M1 (concept) is complete; M2 (technical foundation)
is in progress. No production system exists yet. See
[`docs/memory/current-state.md`](docs/memory/current-state.md) for details
and [`docs/memory/roadmap.md`](docs/memory/roadmap.md) for direction.

## Project structure

`backend/` and `frontend/` are physically separate applications — neither
imports the other's code. New subdirectories are added only when the code
actually needs them, not speculatively.

```
backend/              Go application (net/http, no framework)
  cmd/server/          HTTP server entrypoint
  internal/            backend implementation, not imported externally
    health/             health-check handler
  Dockerfile            backend container image
  Makefile              run/test/build commands

frontend/              Vue 3 + Vite + TypeScript + Pinia application
  src/views/            page-level views
  src/services/         HTTP calls to the backend
  src/router/           Vue Router configuration
  src/stores/            Pinia shared application state
  src/components/       reusable UI components (created when first needed)
  src/types/             shared TypeScript types (created when first needed)
  src/assets/            static assets imported by the app (created when first needed)
  public/                static files served as-is by Vite
  Dockerfile             frontend container image

docker-compose.yml     Docker Compose dev environment (frontend + backend)
docs/manifesto.md      editorial philosophy (canonical)
docs/memory/           persistent project memory for future sessions
```

## Development

Two supported workflows — pick either, neither replaces the other.

### Local development

The simplest way to iterate on code, no Docker required.

Prerequisites: Go 1.24+, Node 20+, npm.

Environment:
- Backend reads `PORT` directly from the OS environment (Go doesn't
  autoload `.env` files) — [`backend/.env.example`](backend/.env.example)
  documents the default; only export `PORT` if you need to override it.
- Frontend: copy [`frontend/.env.example`](frontend/.env.example) to
  `frontend/.env` — Vite loads it automatically.

Backend (terminal 1):

```
cd backend
make run    # start the server on :8080 (or $PORT)
make test   # run tests
make build  # build a binary to bin/server
```

http://localhost:8080 — health check:

```
curl localhost:8080/health
# {"status":"ok"}
```

Frontend (terminal 2), lives in [`frontend/`](frontend/) — Vue 3, Vite,
TypeScript, Pinia:

```
cd frontend
npm install
npm run dev     # start the dev server
npm run build   # type-check and build for production
```

http://localhost:5173 — talks to the backend over HTTP only, via
`VITE_API_BASE_URL`.

### Docker Compose

Requires Docker and Docker Compose. Useful when you want the complete
containerized environment instead of running backend/frontend directly.

Start both frontend and backend (also use this to rebuild after dependency
or Dockerfile changes):

```
docker compose up --build
```

Frontend: http://localhost:5173
Backend: http://localhost:8080

Stop the environment (keeps persisted data):

```
docker compose down
```

SQLite persistence: the backend runs SQLite as an embedded database (not a
separate service/container). Its data directory, `/data` in the backend
container, is backed by the named volume `sqlite_data`, so the database
file survives `docker compose down`. No SQLite application code exists yet
— the volume just reserves the persistent location for when it does.

Remove the persisted database for a clean slate:

```
docker compose down -v
```
