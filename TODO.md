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

## Done (Card 20 — Define Backend/Frontend Project Structure)

- [x] Go backend moved from repo root into `backend/` (`cmd/`,
      `internal/`, `go.mod`, `Makefile`, `Dockerfile`, `.dockerignore`,
      `.env.example`) — supersedes the Card 15/18 decision to keep it at
      repo root
- [x] `docker-compose.yml` backend build context updated to `./backend`
- [x] `README.md` project structure section rewritten to document
      `backend/`/`frontend/` responsibilities and subdirectory conventions
- [x] Project memory updated (`current-state.md`, `decisions.md`)
- [x] No frontend changes — existing structure already matched the target
- [x] No speculative folders created (`components/`, `types/`, `assets/`
      remain absent until actually needed)

## Done (Card 21 — Create Initial README)

- [x] Root `README.md` expanded into full entry point: what/why,
      philosophy (musical bridges), manifesto link, MVP scope, status, tech
      stack, project structure, local + Docker development, documentation
      links, contributing note
- [x] No new scope document created — scope linked from `CLAUDE.md` and
      `docs/memory/decisions.md`
- [x] No code, Docker, or environment changes
- [x] Project memory updated (`current-state.md`)

## Done (Card 23 — Research Current Music APIs & Data Sources)

- [x] `docs/spotify-api.md` — current Spotify Web API and Last.fm API
      research (auth, search, artists/tracks, Audio Features/Analysis
      status, Recommendations/Related Artists status, playlists, rate
      limits, deprecations, Spotify vs Last.fm comparison, data strategy,
      musical bridge and emerging-artist implications, M3 recommendation,
      open questions, sources)
- [x] Project memory updated (`current-state.md`, `decisions.md`)
- [x] No API integration, clients, SDKs, or dependencies added
- [x] No credentials committed

## Done (Card 24 — Define Spotify Integration Architecture & Local Dev Config)

- [x] `docs/spotify-integration.md` — architecture doc (OAuth flow, token
      ownership/storage, backend/frontend boundary, API boundary, error
      handling, rate-limit strategy, security boundaries, single-user MVP
      assumptions, Last.fm boundary, Mermaid diagram)
- [x] Local dev config centralized under `dev/` (`dev/docker-compose.yml`,
      `dev/.env`, `dev/.secrets.env`); root `docker-compose.yml` moved,
      build contexts updated to `../backend`/`../frontend`
- [x] `backend/.env.example` removed (superseded by `dev/`)
- [x] `.gitignore` covers `dev/.secrets.env`
- [x] Root `Makefile` docker targets point at `dev/docker-compose.yml`;
      `backend-run`/`frontend-run` source `dev/.env`(+`.secrets.env`)
- [x] `README.md` documents the new `dev/` layout and env file split
- [x] Project memory updated (`current-state.md`, `decisions.md`)
- [x] No Spotify OAuth, client, SDK, or dependency code added
- [x] No Last.fm code added
- [x] No credentials committed

## Done (Card 25 — Implement Spotify OAuth Authentication & Token Lifecycle)

- [x] `backend/internal/spotify/` — Authorization Code OAuth flow (no
      PKCE): state generation/validation, token exchange, lazy on-demand
      refresh, `invalid_grant` → `authorization_required`
- [x] `GET /api/spotify/auth`, `GET /api/spotify/callback`,
      `GET /api/spotify/status` wired into `cmd/server/main.go`
- [x] SQLite `spotify_connection` table (single row, `modernc.org/sqlite` —
      first backend dependency, pure Go, no CGO)
- [x] No scopes requested — `GET /v1/me` identity is enough for this card
- [x] Backend tests: state, client (exchange/refresh/`invalid_grant`/`Me`),
      store, handlers — all via stdlib `httptest` + in-memory SQLite, no
      mocking library, no real Spotify calls
- [x] `frontend/src/services/spotify.ts` + `HomeView.vue` — Connect Spotify
      button, connection status display, no new Pinia store (page-local
      state)
- [x] `dev/.env`'s `SPOTIFY_REDIRECT_URI` corrected to `127.0.0.1` (Spotify
      rejects bare `localhost` for non-HTTPS redirect URIs)
- [x] `dev/docker-compose.yml` — `SQLITE_PATH=/data/sound-continuum.db`
      override for the backend service
- [x] `backend/Dockerfile` — `COPY go.mod go.sum`, base image bumped to
      `golang:1.25-alpine` (required by `modernc.org/sqlite`)
- [x] Project memory updated (`current-state.md`, `decisions.md`)
- [x] `docs/spotify-integration.md` and `README.md` updated with the
      implemented endpoints, env vars, and manual OAuth test steps
- [x] No catalog/search/playlist/Last.fm code added

## Done (Card 26 — Implement Spotify API Client)

- [x] `backend/internal/spotify/` extended (no new package): `errors.go`
      (Spotify Web API error taxonomy — `APIError` + status-code
      sentinels, `Retry-After` on 429), `types.go` (`Paging[T]`,
      `Playlist`, `PlaylistItem`, `Track`, `Artist`, `SearchResult`)
- [x] `GET /api/spotify/me`, `GET /api/spotify/playlists`,
      `GET /api/spotify/playlists/{id}/items`, `GET /api/spotify/search`
      wired into `cmd/server/main.go`
- [x] `Service.connection`/`refresh`/`withToken` — proactive (30s leeway)
      + reactive (401-retry-once) token refresh, shared by all new
      operations; `EnsureValidToken` now a thin wrapper, same contract
- [x] OAuth scope added: `user-read-private playlist-read-private`
      (Card 25 requested none) — curator reconnects once
- [x] Backend tests: new `client_test.go`/`handlers_test.go` cases for
      each operation, query encoding, 401-refresh-retry, failed-refresh
      invalidation, 403/404/429/500 typed errors, malformed JSON, search
      limit rejection — same conventions as Card 25, no real Spotify calls
- [x] Real Spotify integration test: `/me`, `/playlists`,
      `/playlists/{id}/items` verified against the real API with the
      existing dev app; found and fixed two field-name mismatches
      (`items`/`item` vs. assumed `tracks`/`track`) not caught by
      research or unit tests; no tokens/secrets in logs
- [x] Project memory updated (`current-state.md`, `decisions.md`,
      `roadmap.md`)
- [x] `docs/spotify-integration.md` updated (§22 + amendments to
      §6/§12/§13/§19/§21)
- [x] No playlist creation/management, no Last.fm, no discovery/ranking/
      curation logic added

## Done (Card 27 — Retrieve and Inspect Spotify Playlists)

- [x] `GET /api/spotify/playlists/{id}` added (`Client.Playlist`,
      `Service.Playlist`, `PlaylistHandler`) — no scope change
- [x] `Playlist` gained `href`, `collaborative`, `snapshot_id`,
      `external_urls.spotify`, `images` (`Image`, new)
- [x] `PlaylistItem` reworked to a discriminated union (`ItemType` +
      `Track`/`Episode` pointers, `AddedBy`, `IsLocal`) via custom
      `UnmarshalJSON`/`MarshalJSON` — no longer silently casts every item
      to `Track`; a null item decodes to `ItemType: "unavailable"`,
      `Episode` is a new minimal type
- [x] Backend tests: new success/403/empty/episode/unavailable-item cases
      in `client_test.go`/`handlers_test.go`, existing Card 26 tests
      updated for the `Track` pointer change — same conventions, no real
      Spotify calls
- [x] Real Spotify integration test: `/playlists/{id}` and
      `/playlists/{id}/items` verified against the real API (existing dev
      connection, no reconnect needed); confirmed metadata stays available
      for a playlist the curator doesn't own while its items 403; no
      tokens/secrets in logs
- [x] Project memory updated (`current-state.md`, `decisions.md`,
      `roadmap.md`)
- [x] `docs/spotify-integration.md` updated (§23)
- [x] No playlist write, no Sound Continuum playlist discovery (deferred —
      no name/ID config exists), no frontend UI

## In progress

- Nothing currently in progress.

## Planned

- M3: Spotify integration
- M4: Discovery engine
- M5: Musical ranking & bridges
- M6: Curator experience
- M7: Weekly editorial workflow
- M8: Feedback & evolution
