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

## In progress

- Nothing currently in progress.

## Planned

- M3: Spotify integration
- M4: Discovery engine
- M5: Musical ranking & bridges
- M6: Curator experience
- M7: Weekly editorial workflow
- M8: Feedback & evolution
