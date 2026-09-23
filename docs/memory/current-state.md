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
  (`GET /health`). No database, no auth, no framework. Lives at repo root
  (`cmd/`, `internal/`, `go.mod`), not under a `backend/` directory.
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
  defines `backend` (root `Dockerfile`, port 8080) and `frontend`
  (`frontend/Dockerfile`, Vite dev server, port 5173), started together via
  `docker compose up --build`. SQLite persistence is provided through a
  named volume (`sqlite_data`) mounted at `/data` in the backend
  container — SQLite is intentionally not a separate service/container,
  since it's an embedded database. No SQLite application code exists yet.

Update this file after meaningful implementation progress. Keep it a
snapshot, not a detailed changelog — see [`decisions.md`](decisions.md) for
the reasoning behind changes.
