# Current State

- Project is in MVP development.
- M1 (editorial/concept milestone) is complete.
- M2 (technical foundation) is in progress.
- No production system exists yet.
- No authentication is required for the MVP.
- No complex infrastructure is required.
- Spotify integration will be introduced later (M3).
- Editorial workflow is still being developed.
- Go backend exists: `cmd/server` (HTTP server), `internal/health`
  (`GET /health`). No database, no auth, no framework. Lives at `backend/`
  (`cmd/`, `internal/`, `go.mod`, `Makefile`, `Dockerfile`), physically
  separated from `frontend/` (Card 20).
- Vue 3 + Vite + TypeScript frontend exists at `frontend/`: a minimal
  application shell (`views/HomeView.vue`) proving the build/dev pipeline
  works, a `services/` boundary (`services/health.ts`) calling the
  backend's `GET /health` non-blockingly, and a single `/` route via Vue
  Router. Pinia is configured (`main.ts`) and `frontend/src/stores/` is the
  established convention for future stores (Card 17); no stores exist yet —
  none are needed until a future card introduces real application state.
  Frontend/backend separation is physical: `frontend/` vs. everything else
  at repo root; the frontend talks to the backend only over HTTP, via
  `VITE_API_BASE_URL`.
- Docker Compose dev environment exists (Card 18): root `docker-compose.yml`
  defines `backend` (`backend/Dockerfile`, port 8080) and `frontend`
  (`frontend/Dockerfile`, Vite dev server, port 5173), started together via
  `docker compose up --build`. SQLite persistence is provided through a
  named volume (`sqlite_data`) mounted at `/data` in the backend
  container — SQLite is intentionally not a separate service/container,
  since it's an embedded database. No SQLite application code exists yet.
- Direct host development is supported and documented (Card 19), fully
  equivalent to and independent of Docker Compose: `make run` for the
  backend, `npm run dev` for the frontend. `backend/.env.example` (`PORT`)
  and `frontend/.env.example` (`VITE_API_BASE_URL`) document the
  environment; Go reads `PORT` from the OS environment directly (no
  `.env` autoloading), Vite autoloads `frontend/.env`. The backend now
  sends `Access-Control-Allow-Origin: http://localhost:5173` on every
  response (`cmd/server/main.go`), a minimal stdlib fix needed because
  the frontend dev server and backend are different origins — without it
  the browser silently blocks the frontend's health check. README's
  `## Development` section documents local dev and Docker Compose as two
  independent, parallel workflows.
- Top-level project structure is defined and documented (Card 20):
  `backend/` and `frontend/` are physically separate applications with no
  shared/common code directory between them, each internally organized by
  responsibility (backend: `cmd/`, `internal/`; frontend: `views/`,
  `services/`, `router/`, `stores/`, plus `components/`, `types/`,
  `assets/` reserved by convention). The structure is intentionally
  minimal — new directories are introduced only when actual code requires
  that responsibility, not speculatively.
- Root `README.md` established as the project's full entry point (Card 21):
  what/why, philosophy (musical bridges), a link to the manifesto, current
  MVP scope (linking `CLAUDE.md` and `decisions.md` — no dedicated scope
  doc exists), status, tech stack, project structure, local and Docker
  development workflows, and a documentation links section covering all of
  `docs/memory/`.
- Root `Makefile` added as the common developer task entry point (Card 22),
  alongside the existing `backend/Makefile` (untouched, still `run`/
  `test`/`build`). Root targets delegate rather than duplicate:
  `backend-run`/`backend-build`/`backend-test` call into `backend/Makefile`
  via `$(MAKE)`; `frontend-run`/`frontend-build` call the existing `npm`
  scripts; `build`/`test` compose the backend and frontend targets;
  `docker-up`/`docker-down`/`docker-build` wrap the Card 18 Compose setup.
  No lint targets exist — no linter is configured for either stack. No
  frontend test target — no test runner is configured. No `dev` target
  that runs both apps at once — that would need a process supervisor the
  project doesn't have; `backend-run` + `frontend-run` in two terminals
  stays the workflow. `make help` is the default target.

Update this file after meaningful implementation progress. Keep it a
snapshot, not a detailed changelog — see [`decisions.md`](decisions.md) for
the reasoning behind changes.
