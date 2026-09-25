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

- External music API research is complete (Card 23, see
  [`docs/spotify-api.md`](../spotify-api.md)). Key finding: Spotify's Web API
  changed substantially since this project's roadmap was written — Audio
  Features, Audio Analysis, Recommendations, and Related Artists are all
  deprecated for apps without pre-existing extended-quota access (a cutoff
  Sound Continuum, as a new app, is on the wrong side of), and Extended
  Quota Mode itself now requires an organizational application with 250k+
  monthly active users. M3 should be scoped around what's actually
  available now (Search, Artist/Track metadata, Playlist create/manage) and
  built for permanent Development Mode limits, not Extended Quota Mode.
  Last.fm was evaluated as a complementary source (similar artists/tracks,
  tags — all available with just an API key, no auth) but is not included
  in M3; it's relevant to M4's discovery engine instead. No integration
  code, API clients, or credentials were added — research only.

- Spotify integration architecture is defined (Card 24, see
  [`docs/spotify-integration.md`](../spotify-integration.md)) — no Spotify
  code exists yet. Local development configuration is now centralized
  under `dev/` (`dev/docker-compose.yml`, `dev/.env` for non-secret
  config, `dev/.secrets.env` for secrets; both gitignored), replacing the
  root `docker-compose.yml` and `backend/.env.example` (removed —
  superseded). The architecture: Go backend owns Spotify OAuth
  (Authorization Code, not PKCE — the backend is a confidential client),
  tokens, and all Spotify API communication; the frontend only calls
  Sound Continuum's own `/api/spotify/*` endpoints and never sees Spotify
  credentials or tokens. Refresh tokens will be stored in SQLite (already
  provisioned), not env files or process memory. Last.fm remains outside
  M3. Backend structure planned as `backend/internal/spotify/`, no
  provider abstraction. `frontend/.env.example` is unchanged.

- Spotify OAuth (Authorization Code, no PKCE) and token lifecycle are
  implemented (Card 25, see
  [`docs/spotify-integration.md`](../spotify-integration.md) §21) —
  `backend/internal/spotify/` provides `GET /api/spotify/auth`,
  `GET /api/spotify/callback`, `GET /api/spotify/status`. The refresh token
  is persisted in SQLite (single-row `spotify_connection` table), refreshed
  lazily on read (no background worker), and `invalid_grant` flips the
  connection to `authorization_required` without discarding it. This is
  the backend's first dependency: `modernc.org/sqlite` (pure Go, no CGO).
  No OAuth scope is requested — `GET /v1/me` identity is sufficient for
  this card. `SPOTIFY_REDIRECT_URI` in `dev/.env` is now
  `http://127.0.0.1:8080/api/spotify/callback` (Spotify rejects bare
  `localhost` for non-HTTPS redirect URIs); `dev/docker-compose.yml` sets
  `SQLITE_PATH=/data/sound-continuum.db` for the backend container.
  `frontend/src/services/spotify.ts` + `HomeView.vue` add a Connect Spotify
  button and connection status display — no new Pinia store, the state is
  page-local. No catalog, search, playlist, or Last.fm code exists yet.

- A Spotify Web API client is implemented (Card 26, see
  [`docs/spotify-integration.md`](../spotify-integration.md) §22), still in
  `backend/internal/spotify/` — no new package. `GET /api/spotify/me`,
  `GET /api/spotify/playlists`, `GET /api/spotify/playlists/{id}/items`,
  and `GET /api/spotify/search` are implemented, read-only. `AuthURL` now
  requests `user-read-private playlist-read-private` (added this card —
  Card 25 requested none), so the curator had to reconnect once. Token
  access reuses Card 25's lazy on-read refresh (`Service.connection`,
  extracted from the old `EnsureValidToken`) and adds reactive
  401-refresh-retry-once (`Service.withToken`) — one shared path, no
  duplicated refresh logic, no new invalidation mechanism beyond the
  existing `needs_reauth` flag. Errors map to a small typed taxonomy
  (`errors.go`: `ErrUnauthorized`/`ErrForbidden`/`ErrNotFound`/
  `ErrRateLimited`/`ErrAPIFailure`/`ErrTransport`/`ErrDecode`, carried on
  `*APIError` with status code + `Retry-After`). Response types
  (`types.go`: `Paging[T]`, `Playlist`, `PlaylistItem`, `Track`, `Artist`,
  `SearchResult`) are integration-layer only — no Sound Continuum domain
  model yet. Verified live against the real API: a playlist's item-count
  summary and each item's track payload are nested under JSON keys
  `"items"`/`"item"`, not the `docs/spotify-api.md` research's assumed
  `"tracks"`/`"track"` — corrected in `types.go`. Search's response shape
  was not exercised live. No playlist write, no Last.fm, no discovery/
  ranking/curation logic.

- Single-playlist retrieval and playlist-item type discrimination are
  implemented (Card 27, see
  [`docs/spotify-integration.md`](../spotify-integration.md) §23), still in
  `backend/internal/spotify/` — no new package, no scope change (Card 26's
  `playlist-read-private` already covers it). `GET /api/spotify/playlists/{id}`
  is new (`Client.Playlist`, `Service.Playlist`, `PlaylistHandler`); `Playlist`
  gained `href`, `collaborative`, `snapshot_id`, `external_urls.spotify`,
  `images`. `PlaylistItem` no longer unconditionally decodes into `Track` —
  it now inspects the nested item's `type` and populates exactly one of
  `Track`/`Episode`, or neither with `ItemType: "unavailable"` for a null
  item, via a custom `UnmarshalJSON`/`MarshalJSON` pair (`Episode` is a new,
  minimal type). Verified live: playlist metadata (including
  `items.total == 0`) stays available for a playlist the curator doesn't
  own, while that same playlist's items 403; a nonexistent playlist ID 400s.
  Both map through the existing `writeSpotifyError` default case (502, not a
  fabricated 404) — unchanged from Card 26. No Sound Continuum playlist
  discovery logic was added — no playlist name/ID configuration exists
  anywhere in the repo, and none was introduced; that lookup is deferred to
  a later application/service layer. No playlist write, no frontend UI.

- Single-track retrieval is implemented (Card 28, see
  [`docs/spotify-integration.md`](../spotify-integration.md) §24), still in
  `backend/internal/spotify/` — no new package, no scope change.
  `GET /api/spotify/tracks/{id}` is new (`Client.Track`, `Service.Track`,
  `TrackHandler`). `Track` gained `href`, `type`, `external_urls`,
  `explicit`, `disc_number`, `track_number`, `is_local`, `preview_url`,
  `external_ids.isrc`, and a full `Album` (new type); `Artist` gained
  `href`/`external_urls`. No `market` parameter — the client always uses a
  user access token, which Spotify resolves market from automatically. An
  empty track ID is rejected client-side (`ErrEmptyTrackID`) before a
  request is built, mapped to `400` by the handler, mirroring `Search`'s
  limit-validation pattern. `403`/`404` both fall through the existing
  `writeSpotifyError` default (`502`) — unchanged from Card 26/27, no new
  status-code special case. Verified live: a real track from an owned
  playlist decoded correctly (name, artists, album, duration, URI,
  external URL); a nonexistent track ID returned `502`. No bulk track
  retrieval, no audio features, no artist/album enrichment calls, no
  frontend track UI.

Update this file after meaningful implementation progress. Keep it a
snapshot, not a detailed changelog — see [`decisions.md`](decisions.md) for
the reasoning behind changes.
