# TODO

## Done (Card 15 — Initiate Go Backend)

- [x] `docs/manifesto.md` — canonical editorial principles
- [x] `docs/memory/` project memory system (README, project-context,
      current-state, decisions, roadmap)
- [x] `CLAUDE.md` harness rules
- [x] Go module + `cmd/server` HTTP server
- [x] `GET /health` endpoint
- [x] Health endpoint test
- [x] `Makefile` (`run`, `test`, `build`)
- [x] Root `README.md`

## Done (Card 16 — Vue 3 + Vite Frontend Foundation)

- [x] `frontend/` — Vue 3 + Vite + TypeScript app
- [x] Pinia configured (no stores yet — none needed)
- [x] Vue Router configured (single `/` route)
- [x] `services/health.ts` — non-blocking `GET /health` check
- [x] `VITE_API_BASE_URL` env-based backend URL configuration
- [x] Frontend section in root `README.md`

## Done (Card 17 — Configure Pinia)

- [x] Pinia bootstrap verified (`app.use(createPinia())` in `main.ts`,
      installed via `frontend/package.json`)
- [x] `frontend/src/stores/` established as the store location convention
- [x] No stores created — no real shared application state exists yet

## Done (Card 18 — Configure Docker Compose)

- [x] Root `Dockerfile` (backend, build context `.`)
- [x] `frontend/Dockerfile` (Vite dev server)
- [x] Root `docker-compose.yml` — `backend` + `frontend` services
- [x] Named volume `sqlite_data` mounted at `/data` in backend container
      (no separate SQLite service/container)
- [x] `.gitignore` — `*.db`
- [x] Docker Compose section in root `README.md`
- [x] Project memory updated (`current-state.md`, `decisions.md`)

## Done (Card 19 — Create Local Development Environment)

- [x] Direct host dev documented and validated: `make run` (backend),
      `npm run dev` (frontend), no Docker required
- [x] `.env.example` (root) and `frontend/.env.example` confirmed as the
      required env vars — no new vars invented
- [x] `.gitignore` confirmed already correct (`.env`, `*.db` ignored,
      `.env.example` tracked) — no changes needed
- [x] Minimal dev-only CORS added to the backend (stdlib
      `net/http` only) — frontend/backend are different origins, so the
      browser was silently blocking the documented health check
- [x] `README.md` restructured into `## Development` with
      `### Local development` / `### Docker Compose` as explicit,
      parallel options
- [x] Project memory updated (`current-state.md`)
- [x] Docker Compose from Card 18 confirmed still working

## In progress

- Nothing currently in progress.

## Planned

- M3: Spotify integration
- M4: Discovery engine
- M5: Musical ranking & bridges
- M6: Curator experience
- M7: Weekly editorial workflow
- M8: Feedback & evolution
