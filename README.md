# Sound Continuum

A weekly music curation project connecting timeless classics, contemporary
music, and emerging artists through intentional musical bridges — treating
music as one continuous story rather than a set of isolated weekly drops.

## Philosophy

The connections between tracks matter as much as the tracks themselves. Each
weekly edition is a chapter that carries forward what came before and sets
up what comes next, sequenced by mood, tension, and release rather than a
fixed formula. Discovery is welcome, but a track earns its place because it
belongs in the story — not because it's obscure.

This is the short version — the full editorial philosophy is canonical:
[Read the Sound Continuum Manifesto](docs/manifesto.md).

## Scope

Sound Continuum is intentionally small and experimental right now. The
current goal is to validate the curation workflow, musical bridges, weekly
editions, discovery, and audience response — not to become a large music
platform. See the MVP constraints in [`CLAUDE.md`](CLAUDE.md) and the
reasoning behind them in
[`docs/memory/decisions.md`](docs/memory/decisions.md).

## Status

MVP under development. M1 (concept) is complete; M2 (technical foundation)
is in progress. No production system exists yet. See
[`docs/memory/current-state.md`](docs/memory/current-state.md) for details
and [`docs/memory/roadmap.md`](docs/memory/roadmap.md) for direction.

## Tech stack

- **Backend**: Go, standard library `net/http` — no framework, no external
  dependencies.
- **Frontend**: Vue 3, Vite, TypeScript, Pinia.
- **Docker / Docker Compose**: local containerized dev environment.
- **SQLite**: a data volume is reserved for it (see Docker Compose below),
  but no application code persists to it yet.

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
  src/router/            Vue Router configuration
  src/stores/            Pinia shared application state
  src/components/        reusable UI components
  src/types/             shared TypeScript types
  src/assets/            static assets imported by the app (these three: created when first needed)
  public/                static files served as-is by Vite
  Dockerfile             frontend container image

docker-compose.yml     Docker Compose dev environment (frontend + backend)
docs/manifesto.md      editorial philosophy (canonical)
docs/memory/           persistent project memory for future sessions
```

## Development

Two supported workflows — pick either, neither replaces the other.

A root [`Makefile`](Makefile) wraps the commands below as a convenience
layer — `make help` lists what's available:

```
make backend-run       # cd backend && go run ./cmd/server
make frontend-run      # cd frontend && npm run dev
make build             # backend-build + frontend-build
make test              # backend-test (no frontend test runner configured yet)
make docker-up         # docker compose up --build
```

It's optional — the direct commands below still work exactly the same and
remain the reference for what's actually being executed.

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

## Documentation

- [Manifesto](docs/manifesto.md) — canonical editorial principles
- [Project memory overview](docs/memory/README.md)
- [Current state](docs/memory/current-state.md)
- [Decisions](docs/memory/decisions.md)
- [Roadmap](docs/memory/roadmap.md)

## Contributing

Sound Continuum is intentionally developed as a small experimental MVP:
simple solutions, clear responsibilities, minimal dependencies, incremental
architecture. Avoid speculative infrastructure and premature abstraction.
AI-assisted development workflow rules live in [`CLAUDE.md`](CLAUDE.md).
