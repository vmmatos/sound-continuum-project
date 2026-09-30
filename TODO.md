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
- **Superseded (Vue Router line)** — the app never grew a second route;
      removed via the over-engineering audit cleanup below.

## Done (Card 17 — Configure Pinia)

- [x] Pinia bootstrap verified (`app.use(createPinia())` in `main.ts`,
      installed via `frontend/package.json`)
- [x] `frontend/src/stores/` established as the store location convention
- [x] No stores created — no real shared application state exists yet
- **Superseded** — Pinia sat unused with zero stores through every later
      frontend card; removed (dependency, bootstrap, `stores/` placeholder)
      via the over-engineering audit cleanup below. Reinstate when a real
      store is needed.

## Done (Remove unused Pinia dependency)

- [x] `frontend/package.json`/`package-lock.json` — `pinia` removed
- [x] `frontend/src/main.ts` — `.use(createPinia())` removed
- [x] `frontend/src/stores/` (empty `.gitkeep` placeholder) removed
- [x] `npm run build` (type-check + Vite build) verified clean
- [x] Project memory updated (`current-state.md`, `decisions.md`)

## Done (Remove unused vue-router dependency and dead CategoryNewRelease enum)

- [x] `frontend/package.json`/`package-lock.json` — `vue-router` removed
- [x] `frontend/src/router/` (single-route config) removed
- [x] `frontend/src/App.vue` renders `HomeView` directly instead of
      `<RouterView />`; `frontend/src/main.ts` no longer installs a router
- [x] `npm run build` (type-check + Vite build) verified clean
- [x] `backend/internal/candidate` — `CategoryNewRelease` removed (zero
      production call sites since Card 31); two test fixtures repointed at
      `CategoryPresent`
- [x] `go build ./...`, `go vet ./...`, `go test ./...` verified clean
- [x] Project memory updated (`current-state.md`, `decisions.md`)
- [x] Root `README.md` project structure section corrected (also dropped a
      stale Pinia/`stores/` mention left over from that earlier removal)

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

## Done (Card 28 — Retrieve Track Information)

- [x] `GET /api/spotify/tracks/{id}` added (`Client.Track`,
      `Service.Track`, `TrackHandler`) — no scope change
- [x] `Track` gained `href`, `type`, `external_urls`, `explicit`,
      `disc_number`, `track_number`, `is_local`, `preview_url`,
      `external_ids.isrc`, and a full `Album` (new type); `Artist` gained
      `href`/`external_urls`
- [x] No `market` parameter (Spotify infers it from the user token this
      client always uses); no bulk retrieval (`GetTracks`/`ids=` is
      unavailable in Development Mode); no artist/album enrichment calls;
      no audio-features model
- [x] Empty track ID rejected client-side (`ErrEmptyTrackID`) before any
      request, mirroring `Search`'s limit-validation pattern; mapped to
      `400` by the handler
- [x] Backend tests: 9 new `client_test.go` cases (success, full metadata,
      multiple artists, album metadata, missing optional fields, 404, 401,
      malformed JSON, empty ID) + 3 new `handlers_test.go` cases (path-ID
      pass-through, 403→502, empty-ID→400) — same conventions, no real
      Spotify calls
- [x] Real Spotify integration test: `GetTrack` verified against a real
      track from an owned playlist (name, artists, album, duration, URI,
      external URL all decoded correctly); nonexistent ID returned 502; no
      tokens/secrets in logs
- [x] Project memory updated (`current-state.md`, `decisions.md`,
      `roadmap.md`)
- [x] `docs/spotify-integration.md` updated (§24)
- [x] No bulk track retrieval, no audio features, no editorial/scoring
      logic, no frontend track UI

## Done (Card 29 — Retrieve Artist Information)

- [x] `GET /api/spotify/artists/{id}` added (`Client.Artist`,
      `Service.Artist`, `ArtistHandler`) — no scope change
- [x] `Artist` gained `type`, `images`, `genres` (shared by `Track.Artists`,
      `Album.Artists`, `SearchResult.Artists`)
- [x] No `followers`/`popularity` (removed by Spotify for Development
      Mode); no bulk artist retrieval (`GET /artists?ids=` unavailable in
      Development Mode); no `/artists/{id}/top-tracks` (removed, no
      compatibility wrapper); `genres` treated as optional/deprecated
      metadata only, no genre normalization or classification logic; no
      automatic artist enrichment from `GetTrack`/playlist items
- [x] Empty artist ID rejected client-side (`ErrEmptyArtistID`) before any
      request, mirroring `Track`'s pattern; mapped to `400` by the handler
- [x] Backend tests: 8 new `client_test.go` cases (success, full metadata,
      multiple images with a null-dimension image, missing optional
      fields, 404, 401, malformed JSON, empty ID) + 3 new
      `handlers_test.go` cases (path-ID pass-through, 403→502,
      empty-ID→400) — same conventions, no real Spotify calls
- [x] Real Spotify integration test: `GetArtist` verified against a real
      artist from an owned playlist's track (António Calvário — name, id,
      uri, external URL, three images all decoded correctly; `genres` was
      `null` for this artist, decoding cleanly); nonexistent ID returned
      502; no tokens/secrets in logs
- [x] Project memory updated (`current-state.md`, `decisions.md`,
      `roadmap.md`)
- [x] `docs/spotify-integration.md` updated (§25)
- [x] No bulk artist retrieval, no top-tracks, no followers/popularity, no
      genre classification, no Last.fm, no editorial scoring, no frontend
      artist UI

## Done (Card 30 — Create and Publish Sound Continuum Playlist)

- [x] `POST /api/spotify/playlist` added (`Client.CreatePlaylist`,
      `Service.InitializeOfficialPlaylist`, `InitializePlaylistHandler`) —
      first write endpoint, first non-GET route
- [x] New `official_playlist` SQLite table (singleton, same `CHECK (id =
      1)` pattern as `spotify_connection`), persists Spotify playlist ID,
      name, URL, created_at
- [x] Idempotent by construction: a local row is returned with no Spotify
      call at all if one already exists — Spotify is called to create the
      playlist at most once, ever
- [x] Playlist created public, non-collaborative, named exactly
      `Sound Continuum — Weekly Journey`, with the agreed description,
      empty (no tracks)
- [x] OAuth scope gained `playlist-modify-public`; curator must reconnect
      once
- [x] Dev CORS middleware now answers `OPTIONS` preflight (needed for the
      first JSON POST endpoint)
- [x] Backend tests: `CreatePlaylist` success/403 in `client_test.go`;
      store round-trip + duplicate-save-fails in `store_test.go`; create-
      once, idempotent-no-Spotify-call, called-twice-creates-once, not-
      connected, authorization-required, failed-creation-persists-nothing,
      and full HTTP round trip in `handlers_test.go`
- [x] `frontend/src/services/spotify.ts` + `HomeView.vue` — "Initialize
      official playlist" button, link display once created, no new Pinia
      store
- [x] Project memory updated (`current-state.md`, `decisions.md`)
- [x] `docs/spotify-integration.md` updated (§26)
- [x] Real Spotify integration test — verified: playlist created, public,
      non-collaborative, correct name/description, 0 tracks; second call
      confirmed no duplicate
- [x] No track-adding, no editorial workflow, no playlist discovery by
      name/search, no distributed transaction for the Spotify-succeeds/
      local-save-fails case (documented limitation)

## Done (Card 31 — Define Candidate Track Domain Model)

- [x] `backend/internal/candidate/` — new package, the project's first
      domain model outside `spotify`/`health`
- [x] `CandidateTrack` domain type: internal `ID` distinct from the
      external `SpotifyTrackID` (empty unless source is Spotify); no track
      metadata beyond title/artist duplicated from Spotify
- [x] `Source` enum: `Spotify`, `Last.fm`, `Manual` (no Last.fm/manual
      integration implemented — domain concept only)
- [x] `Category` enum: `Past`, `Present`, `Emerging`, `New Release`
- [x] `Status` lifecycle: `discovered`, `under review`, `selected`,
      `rejected` — no workflow/transition logic, no approval/voting states
- [x] Lightweight editorial fields: `DiscoveryReason`, `EditorialNote`,
      `PotentialConnection` — no bridge graph, no scoring
- [x] `Validate()` + sentinel errors (`errors.go`), same pattern as
      `internal/spotify/errors.go`
- [x] Unit tests covering valid construction, each enum's invalid values,
      Spotify-required-ID vs. Manual-no-ID, and editorial fields without
      Spotify data
- [x] Project memory updated (`current-state.md`, `decisions.md`,
      `roadmap.md`)
- [x] No persistence/schema, no API endpoint, no CRUD, no frontend UI, no
      Spotify API calls, no Last.fm integration

## Done (Card 33 — Build Classic Music Discovery)

- [x] `backend/internal/discovery/` — new package, M4's first discovery
      workflow
- [x] `PastReferenceArtists` (15 names) — the single canonical source for
      Sound Continuum's "Past" reference artists, previously undocumented
      anywhere in the repo
- [x] `Client.ArtistAlbums`/`Client.AlbumTracks` +
      `Service.ArtistAlbums`/`Service.AlbumTracks` added to
      `internal/spotify` (`GET /artists/{id}/albums`,
      `GET /albums/{id}/tracks`) — no second Spotify client
- [x] `DiscoverClassic`: exact-match artist resolution via `Search` (no
      fuzzy matching, unresolved artists surfaced, never dropped), bounded
      album/track pagination (`Config`: max albums/artist, max
      tracks/album, max total candidates), dedup by Spotify track ID
- [x] Every candidate: `Source=Spotify`, `Type=Classic`, `Category=Past`,
      `Status=discovered` — no popularity/ranking, no AI, no musical
      bridge logic
- [x] Per-artist/album/track failures recorded and the run continues;
      only a Spotify connection failure aborts the whole run
- [x] `POST /api/discovery/classic` — minimal endpoint, no persistence, no
      frontend UI
- [x] Unit tests (spotify client/service passthroughs + discovery package,
      19 scenarios) — no real Spotify calls in automated tests
- [x] Real Spotify verification: all 15 reference artists resolved, zero
      unresolved, zero failures, zero duplicate candidates, official
      playlist unchanged, no candidate auto-selected
- [x] Project memory updated (`current-state.md`, `decisions.md`,
      `roadmap.md`, `spotify-integration.md`)
- [x] No Last.fm, no persistence layer, no Current/Discovery-type
      discovery, no musical bridge logic, no final editorial selection

## Done (Card 34 — Build Current Music Discovery)

- [x] `PresentReferenceArtists` (15 names) added to
      `backend/internal/discovery/reference_artists.go` — the canonical
      "Present" reference-artist list, alongside Card 33's
      `PastReferenceArtists`
- [x] `DiscoverCurrent` reuses `resolveArtist`/`walkPages`/
      `isConnectionError`/`writeDiscoveryError`/the `spotifyCatalogue` seam
      from Card 33 unchanged; `Service` now holds `classicCfg Config` +
      `currentCfg CurrentConfig`, `NewService` takes both
- [x] `CurrentConfig`: configurable `LookbackDays` (default 90),
      `MaxAlbumsScannedPerArtist` (raw releases fetched per artist, default
      50) separate from `MaxAlbumsPerArtist` (most recent qualifying
      releases kept after filtering/sorting, default 5), plus
      `MaxTracksPerAlbum`/`MaxTotalCandidates`
- [x] `parseReleaseDate` resolves partial `release_date_precision`
      (month/year) to the earliest consistent instant — documented
      conservative handling, never overestimates recency
- [x] Recent releases explicitly sorted release-date-descending (Spotify
      album ID as stable tiebreaker) before truncating to
      `MaxAlbumsPerArtist` — Spotify's artist-albums order isn't documented
      as chronological
- [x] Singles included via the existing `include_groups=album,single`
      fetch (Card 33); no album-type filtering added
- [x] Every candidate: `Source=Spotify`, `Type=Current`, `Category=Present`,
      `Status=discovered` — no popularity/ranking, no AI, no musical
      bridge logic
- [x] `discovery.Result` gained `ReleasesOutsideWindow` (always 0 for
      Classic); dedup by Spotify track ID, same as Classic
- [x] `POST /api/discovery/current` — minimal endpoint, no persistence, no
      frontend UI
- [x] Unit tests (discovery package, 19 new scenarios covering resolution,
      candidate fields, window filtering, singles, dedup, unresolved
      artists, failure continuation, connection abort, scan/selection
      bounding, total-candidate cutoff, no popularity reordering,
      release-date-precision parsing, handler status mapping) — no real
      Spotify calls in automated tests
- [x] Real Spotify verification: small-subset run confirmed genuinely
      recent candidates, correct metadata, no duplicates, official
      playlist unchanged; full 15-artist reference set run confirmed safe
      under the bounded limits
- [x] Project memory updated (`current-state.md`, `decisions.md`,
      `roadmap.md`)
- [x] No Last.fm, no persistence layer, no Emerging/New-Release discovery,
      no musical bridge logic, no final editorial selection

## Done (Card 35 — Build Emerging Artist Discovery)

- [x] `backend/internal/lastfm/` — new package, the repo's first Last.fm
      integration; one file, `artist.getsimilar` only, no provider
      abstraction
- [x] `lastfm.Client.SimilarArtists` — injects `LASTFM_API_KEY`, sets an
      identifiable `User-Agent`, returns `[]SimilarArtist{Name, Match}`;
      a missing API key returns `ErrMissingAPIKey` with no request made
- [x] Typed Last.fm errors (`ErrMissingAPIKey`, `ErrRateLimited` — code 29,
      `ErrAPIFailure`, `ErrTransport`, `ErrDecode`) + `APIError{Code,
      Message, Unwrap}`, mirroring `internal/spotify`'s own pattern
- [x] `EmergingReferenceArtists` (15 names) added to
      `backend/internal/discovery/reference_artists.go` — the canonical
      "Emerging" seed-artist list, alongside `PastReferenceArtists`/
      `PresentReferenceArtists`
- [x] `isCanonicalReferenceArtist` — excludes any Last.fm result already
      present in Past/Present/Emerging (case-insensitive, trimmed)
- [x] `DiscoverEmerging`: one Last.fm hop per seed only (never calls
      `SimilarArtists` with a discovered name), dedup by discovered-artist
      name, `resolveArtist` reused unchanged, recent catalogue reused via
      new shared `recentTracksForArtist` (extracted from `DiscoverCurrent`
      — same "recent" definition, no second one invented)
- [x] `EmergingConfig` (own struct): `MaxSimilarPerSeed`, `MaxDiscoveredArtists`,
      plus the same recent-catalogue fields as `CurrentConfig`; `Service`
      gains a `lastfm similarArtistFinder` dependency, `NewService` takes
      a `*lastfm.Client`
- [x] Every candidate: `Source=Spotify` (never `SourceLastFM`),
      `Type=Discovery`, `Category=Emerging`, `Status=discovered` — no
      popularity/ranking, no Last.fm-match-as-score, no AI
- [x] `discovery.Result` gained `EmergingProvenance
      []ArtistProvenance{SeedArtist, DiscoveredArtist, Source, Match}` —
      discovery metadata only, never used to select/order/score; empty for
      Classic/Current
- [x] One seed's Last.fm failure recorded (`Stage: "similar"`), run
      continues; a missing `LASTFM_API_KEY` or a Spotify connection
      failure aborts the whole run with a clear error — never a silent
      empty success
- [x] `POST /api/discovery/emerging` — minimal endpoint, no persistence, no
      frontend UI
- [x] Unit tests (`internal/lastfm`: 7 scenarios covering request shape,
      API key injection, response parsing, rate-limit/API/transport/decode
      errors; `internal/discovery`: 21 new scenarios covering single-hop
      enforcement, seed walking, exclusion, cross-seed dedup, unresolved
      surfacing, recency-window reuse, candidate fields, track dedup,
      both bound types, partial failure continuation, connection/config
      abort, no Match-based reordering, provenance, handler status
      mapping) — no real Last.fm/Spotify calls in automated tests
- [x] Real Last.fm verification: `artist.getsimilar` confirmed live against
      a seed artist via direct `curl` and via the running server
- [x] Real Spotify verification: full bounded `DiscoverEmerging` run
      against the live server correctly aborted with a clear 503 ("Spotify
      is not connected") — same pre-existing behavior as Classic/Current
      in this dev environment (no stored Spotify connection); confirmed
      missing `LASTFM_API_KEY` also aborts with a clear 503, never an
      empty 200
- [x] Project memory updated (`current-state.md`, `decisions.md`,
      `roadmap.md`); `README.md`/`docs/spotify-integration.md` env docs
      updated to list `LASTFM_API_URL`/`LASTFM_API_KEY`
- [x] No Last.fm user authentication, no scoring/ranking engine, no
      recursive/multi-hop discovery, no persistence layer, no musical
      bridge logic, no final editorial selection, no playlist modified

## Done (Card 36 — Build Candidate Pool)

- [x] `backend/internal/discovery/pool.go` — `CandidatePool`, `WorkflowError`,
      `Service.DiscoverPool`, `Service.PoolHandler`, no new package
- [x] `DiscoverPool` orchestrates the existing `DiscoverClassic`/
      `DiscoverCurrent`/`DiscoverEmerging` unchanged, merges their
      candidates, and deduplicates cross-workflow duplicates by Spotify
      track ID via a documented `Classic > Current > Discovery` priority
      rule (`candidateTypePriority`) — deterministic regardless of merge
      order, not list-position-based
- [x] Every candidate's `Source`/`Type`/`Category`/`Status` preserved
      unchanged from its originating workflow; no normalization into a
      generic category
- [x] Deterministic, non-popularity ordering: `Type` priority, then
      `TrackArtist`, `TrackTitle`, `SpotifyTrackID` (no release-date leg —
      `CandidateTrack` has no release date field)
- [x] Counts: `TotalCandidates`, `ClassicCandidates`, `CurrentCandidates`,
      `EmergingCandidates`, `DuplicatesRemoved` — no rigid quotas, no
      rankings
- [x] `ClassicResult`/`CurrentResult`/`EmergingResult` embedded on
      `CandidatePool` unmodified, so each workflow's own counts,
      `UnresolvedArtists`, `Failures`, and (for Emerging)
      `EmergingProvenance` stay inspectable
- [x] Partial-failure-tolerant: `DiscoverPool` always runs all three
      workflows and never returns a top-level error; a workflow abort
      (Spotify connection failure, or for Emerging a missing
      `LASTFM_API_KEY`) is recorded on `WorkflowErrors` while the other
      two workflows' candidates are kept, never silently discarded
- [x] `POST /api/candidates/pool` — minimal endpoint, no persistence, no
      request body, no frontend UI; `CandidatePool` is transient (an
      in-memory discovery-run result), matching the existing M4 pattern
- [x] Unit tests (`backend/internal/discovery/pool_test.go`, 12 scenarios):
      all three workflows included, merge/counts, cross-workflow dedup with
      documented priority, per-workflow partial failure (Classic/Current),
      Emerging per-seed failure continuing, missing Last.fm config
      surfaced, provider errors not swallowed, deterministic ordering, no
      editorial selection, empty results, handler HTTP round trip
- [~] Real Spotify + Last.fm end-to-end run: OAuth reuse, real Search-based
      artist resolution, and real Last.fm `artist.getsimilar` (genuine
      matches) all verified live. `GET /artists/{id}/albums` hit a genuine
      Spotify Development Mode quota (`429`, `Retry-After: ~4h30m`) after
      this card's own testing — the partial-failure design was verified
      correct against this real failure (non-aborting per-artist
      `Failures`, `WorkflowErrors` empty), and the official playlist was
      confirmed unchanged (0 items, before and after) — but an actual
      Spotify track surviving into `CandidatePool.Candidates` was **not**
      confirmed live this session; re-run once the quota clears (see
      `current-state.md` for the full account). Not a code defect.
- [x] Project memory updated (`current-state.md`, `decisions.md`,
      `roadmap.md`)
- [x] No persistence layer, no editorial selection, no musical bridge
      logic, no ranking, no generic discovery framework, no playlist
      modification

## Done (Card #37 — Avoid Recently Used Tracks)

- [x] `backend/internal/discovery/recent_track_filter.go` — new file,
      `Service.FilterRecentTracks`, `DefaultRecentTrackLookbackDays`
      (28-day default, plain `int` — no single-field config struct),
      `RecentTrackFilterResult`, `RecentlyUsedCandidate`, `ReasonRecentlyUsed`
- [x] Walks the full official playlist via `spotify.Service.PlaylistItems`
      (existing, already paginating) with a dedicated pagination loop —
      never assumes the first page is enough; episodes/unavailable/
      malformed items skipped individually; duplicate Spotify track IDs
      keep the maximum `added_at`
- [x] Documented `>=` lookback boundary, unit-tested exactly at the edge
- [x] `discovery.Service` gained an unexported `now func() time.Time`
      clock seam (defaulted to `time.Now`, overridden by same-package
      tests) — this card's one deliberate exception to the repo's
      direct-`time.Now()` convention, needed for deterministic boundary
      tests
- [x] `RECENT_TRACK_LOOKBACK_DAYS` read in `cmd/server/main.go`
      (`os.Getenv`/`strconv.Atoi`, falls back to 28), added to `dev/.env`
- [x] `spotify.Service.OfficialPlaylist` — new read-only accessor,
      `ErrOfficialPlaylistNotConfigured` sentinel; no OAuth scope change
      (`playlist-read-private` already covers it)
- [x] `discovery.spotifyCatalogue` interface extended with
      `PlaylistItems`/`OfficialPlaylist`; `fakeCatalogue` extended to match
- [x] `POST /api/candidates/pool` (`PoolHandler`) now also runs
      `FilterRecentTracks`, attaching `CandidatePool.RecentTrackFilter`;
      unlike `DiscoverPool`, a filter failure (playlist not configured,
      Spotify connection/API failure) returns an HTTP error instead of a
      pool — never a silent all-eligible fallback
- [x] No `candidate.Status` change — recently-used candidates keep
      `Status: discovered`; no new blacklist, no artist/album repetition
      rules, no scoring/ranking
- [x] 26 new unit tests (15 `recent_track_filter_test.go`, 2 `PoolHandler`,
      2 `spotify.Service.OfficialPlaylist`) — all against fakes, no real
      network calls; existing tests remain green
- [x] `go build ./...`, `go vet ./...`, `go test ./...` verified clean
- [~] Real Spotify integration verification (via Docker Compose): official
      playlist had been deleted outside the app (found live — see the
      Card #30 idempotency limitation logged in `decisions.md`); recreated
      via the existing unmodified init flow, then confirmed empty-playlist
      behavior, `RECENT_TRACK_LOOKBACK_DAYS` default, full pagination
      termination, and no playlist mutation (0 items before/after) for
      real. All three discovery workflows hit the pre-existing Card #36
      `ArtistAlbums` 429 rate limit, so `TotalCandidates` was 0 — an
      actual candidate landing in "recently used" against real data was
      not confirmed live; covered by unit tests instead. Not a code defect.
- [x] Project memory updated (`current-state.md`, `decisions.md`,
      `roadmap.md`)
- [x] No persistence layer, no editorial selection, no musical bridge
      logic, no ranking, no generic filtering framework, no playlist
      modification

## Done (Card #39 — Add Source Tracking / Discovery Provenance)

- [x] `backend/internal/candidate/provenance.go` — new file:
      `DiscoveryMethod` (`classic_reference_artist`/
      `current_reference_artist`/`lastfm_similar_artist`/`manual`),
      `ProvenanceProvider` (`Spotify`/`Last.fm`/empty), `SeedArtist`
      (`Provider`, `ProviderArtistID`, `Name`), `DiscoveryProvenance`
      (`Method`, `Provider`, `Seed`, `DiscoveredArtist`, `LastFMMatch`),
      `MergeProvenance` — same `type X string`+`const`+`Valid()`
      convention as `Type`/`Category`/`Status`/`Source`
- [x] `CandidateTrack` gains `Provenance []DiscoveryProvenance` (additive,
      alongside Card #38's `Metadata`); `Validate()` rejects an invalid
      `Method`
- [x] Removed the pre-existing free-text `DiscoveryReason` field (Card
      #31) and its three per-workflow constants — fully subsumed by
      `Provenance[].Method`, a typed enum carrying the same fact
      structurally
- [x] `candidate.Source` completely unchanged — still `SourceSpotify`
      only; no `Manual`/`Last.fm` value added, since no workflow
      constructs a candidate without Spotify identity
- [x] `DiscoverClassic`/`DiscoverCurrent` each attach one provenance entry
      per candidate using the reference artist's Spotify ID already
      resolved earlier in the loop — no extra Spotify call
- [x] `DiscoverEmerging` attaches `lastfm_similar_artist` provenance
      (seed = the original Emerging seed name, discovered artist = the
      already-resolved Spotify artist, `LastFMMatch` = Last.fm's own
      similarity value) — no extra Last.fm or Spotify call
- [x] `DiscoverPool`'s dedup step now merges both candidates' provenance
      via `candidate.MergeProvenance` instead of discarding the losing
      candidate's provenance; classification (`Type`/`Category`/`Status`)
      still decided by the existing `candidateTypePriority`
- [x] `FilterRecentTracks`/`EnrichCandidateMetadata` needed no code
      changes — both already copy `CandidateTrack` by value, so
      `Provenance` survives automatically (confirmed by new tests)
- [x] `POST /api/candidates/pool` and `/api/discovery/*` expose
      `Provenance` automatically (no handler changes — existing JSON
      encoding has no field allowlist)
- [x] 20 new unit tests (`candidate`: enum validation, `MergeProvenance`
      union/dedup/empty-input, manual provenance with no fabricated data;
      `discovery`: Classic/Current/Emerging provenance field correctness
      + no extra provider calls, Pool dedup provenance merging,
      Source-vs-Provider distinction, preservation through
      `FilterRecentTracks`/`EnrichCandidateMetadata`) — all against
      fakes, no real network calls
- [x] `go build ./...`, `go vet ./...`, `go test ./...` verified clean
- [~] Real Spotify + Last.fm verification: connection reuse and real
      `Search`-based artist resolution confirmed live for all three
      workflows; no candidate could be constructed and no provenance
      field exercised against live data because `GET /artists/{id}/albums`
      is still `429`-rate-limited (the same pre-existing Card #36 quota
      state); `POST /api/candidates/pool` itself 502s because the
      locally-recorded official playlist no longer exists on Spotify's
      side (a recurrence of the known Card #30/#37 limitation) — neither
      is a Card #39 defect; every call made was read-only, no playlist
      touched. See `current-state.md` for the full account.
- [x] Project memory updated (`current-state.md`, `decisions.md`,
      `roadmap.md`)
- [x] No persistence, no new endpoint, no ranking/scoring, no generic
      provenance framework, no event sourcing, no playlist modification

## Done (Card #40 — Define Candidate Scoring Model)

- [x] `backend/internal/scoring` — new package, M5's first card
- [x] `scoring.Factors` — six normalized `[0,1]` values (Fit, Freshness,
      DiscoveryBonus, Diversity, PlaylistFit, RepetitionPenalty), each
      `*float64` (nil = not yet available); no factor algorithm
      implemented — every value stays abstract for later M5 cards
- [x] `scoring.Weights` + `DefaultWeights()` — five positive weights
      summing to 1.0 (Fit 0.35, PlaylistFit 0.25, DiscoveryBonus 0.15,
      Diversity 0.15, Freshness 0.10) plus an independent
      `RepetitionWeight` (0.30), justified against the manifesto in
      `docs/scoring-model.md`
- [x] `scoring.Calculate` — deterministic combination formula: missing
      positive factors excluded from a weight-renormalized average (never
      substituted as 0); repetition applied as a multiplicative discount
      (`BaseScore * (1 - penalty*weight)`), keeping `FinalScore` naturally
      bounded in `[0,1]` with no clamping
- [x] `scoring.CandidateScore` — exposes every factor, the weights,
      `AvailableWeight`, and `FinalScore` independently (explainability);
      `ModelVersion = "v1"` label, no history/persistence
- [x] No `CandidateTrack` field changes, no ranking, no sorting, no
      selection, no `Status` mutation, no playlist mutation, no
      popularity-based scoring, no ML
- [x] Not wired into `discovery`/`main.go` — no production code
      constructs real `Factors` yet; `Calculate` is exercised only by its
      own tests, since every per-factor algorithm is future M5 work
- [x] 16 new unit tests (`scoring/score_test.go`): weight/factor range
      validation, weight-sum rule + float tolerance boundary, every
      missing-factor combination (incl. zero-available → nil
      `FinalScore`), multiplicative repetition combination (incl. bounded-
      at-zero), error propagation, explainability, determinism
- [x] `go build ./...`, `go vet ./...`, `go test ./...` verified clean
- [x] `docs/scoring-model.md` — new dedicated doc (factor definitions,
      Fit-vs-PlaylistFit and DiscoveryBonus-vs-Freshness distinctions,
      Repetition-Penalty-vs-Recent-Track-Filter distinction, weights
      table + reasoning, formula + worked example, missing-factor
      strategy, explainability, "what this is not")
- [x] Project memory updated (`current-state.md`, `decisions.md`,
      `roadmap.md`)

## Done (Card #41 — Define Musical Fit)

- [x] `backend/internal/musicaldna` — new package: `Profile` (Mood,
      Energy, Texture, CulturalInfluence, each optional), `ProjectDNA`
      (stable, project-wide, empty by default per the manifesto/M1
      mapping in `decisions.md`), `WeeklyDirection` (`EditionID` +
      `Profile` + `Notes`, `NewWeeklyDirection`-constructed) — one
      dimension vocabulary reused for candidate/project/weekly roles, no
      competing representation
- [x] `scoring.CalculateFit` — compares a candidate's `Profile` against
      both `ProjectDNA.Profile` and `WeeklyDirection.Profile`,
      case-insensitive trimmed exact match per dimension, renormalized
      over available dimensions and then over available components
      (`FitWeights`: WeeklyWeight 0.75 / ProjectWeight 0.25, Mood 0.35 /
      Energy 0.25 / Texture 0.25 / CulturalInfluence 0.15); `Fit` is
      `nil`, never `0.0`, when nothing is comparable
- [x] `scoring.FitResult.Dimensions` — 8 entries (4 dimensions × 2
      components) for explainability regardless of whether `Value` is set
- [x] No confidence score, no editorial-override mechanism — both
      documented as deliberate omissions (`Factors.Fit` already a plain
      settable `*float64`)
- [x] Candidate `Profile` is a `CalculateFit` argument only — no
      `CandidateTrack` field added, no persistence introduced
- [x] `CalculateFit` takes no `candidate.CandidateTrack` — independent of
      `CandidateType`/`Category`/release date/discovery provenance/
      playlist sequence by construction, verified by explicit tests
- [x] 21 new tests (`musicaldna/weekly_direction_test.go`,
      `scoring/fit_test.go`, `scoring/fit_integration_test.go`):
      strong/weak fit, partial and fully missing data, weight validation,
      determinism, classification/freshness/provenance independence,
      explainability, score integration, and one no-network integration
      test through `CandidateScore`
- [x] `go build ./...`, `go vet ./...`, `go test ./...` verified clean
- [x] `scoring.Factors`, `scoring.Weights`, `scoring.Calculate`, and
      `candidate.CandidateTrack` unchanged; no ranking, no selection, no
      `Status` mutation, no playlist mutation
- [x] `docs/scoring-model.md` — Fit section replaced with the full model;
      Freshness/Discovery Bonus distinctions from Fit added
- [x] Project memory updated (`current-state.md`, `decisions.md`,
      `roadmap.md`)

## Done (Card #42 — Define Freshness)

- [x] `scoring.CalculateFreshness(lastUsedAt *time.Time, now time.Time,
      config FreshnessConfig) (FreshnessResult, error)` — half-life
      recovery curve, `Freshness(t) = 1 − 0.5^(t / HalfLifeDays)`, default
      `HalfLifeDays` 60 (`DefaultFreshnessConfig`); continuous,
      monotonically increasing, asymptotic toward but never reaching
      `1.0` for a used candidate, no discontinuity at the 28-day Recent
      Track Filter boundary
- [x] Never-used candidates (`lastUsedAt == nil`) get exactly `Freshness
      = 1.0`; `now`/`lastUsedAt` are explicit parameters, never read from
      the system clock, so Freshness is deterministic
- [x] `discovery.Service.PlaylistTrackHistory` — new exported method
      wrapping Card #37's existing `recentTrackIndex`; `FilterRecentTracks`
      refactored to call it — one playlist-history retrieval mechanism,
      not two
- [x] `scoring.FreshnessLastUsedAt(history map[string]time.Time,
      spotifyTrackID string) *time.Time` — bridges `PlaylistTrackHistory`'s
      plain map to a per-candidate lookup without `scoring` importing
      `discovery`, keeping `CalculateFreshness` I/O-free
- [x] Duplicate playlist entries use the most recent valid `added_at`;
      episodes/unavailable items ignored — both inherited unchanged from
      Card #37's retrieval
- [x] Missing/unavailable playlist history surfaces as an error, never a
      false `Freshness = 1.0`; a successfully-retrieved empty playlist
      legitimately yields `1.0` for every candidate
- [x] `CalculateFreshness` takes no `candidate.CandidateTrack` —
      independent of `CandidateType`/`Category`/release date/discovery
      provenance by construction, verified by explicit tests
- [x] `scoring.FreshnessResult` (`Value`, `LastUsedAt`,
      `TimeSinceLastUse`) for explainability
- [x] 28 new tests across `scoring/freshness_test.go`,
      `scoring/freshness_integration_test.go`,
      `discovery/recent_track_filter_test.go`, and
      `discovery/freshness_integration_test.go`: never-used, gradient
      ordering, boundary behavior around the 28-day window, normalization,
      monotonicity, determinism, independence, invalid config, clock-skew
      clamping, duplicate entries, empty playlist, retrieval failure, and
      the full discovery-history-to-`CandidateScore` reuse path
- [x] `go build ./...`, `go vet ./...`, `go test ./...` verified clean
- [x] `scoring.Factors`, `scoring.Weights` (Freshness weight unchanged at
      0.10), `scoring.Calculate`, `candidate.CandidateTrack`, and
      `scoring.CalculateFit` unchanged; no ranking, no selection, no
      `Status` mutation, no playlist mutation, no persistence
- [x] `docs/scoring-model.md` — Freshness section replaced with the full
      model (curve, config, reuse mechanism, explainability)
- [x] Project memory updated (`current-state.md`, `decisions.md`,
      `roadmap.md`)

## In progress

- Nothing currently in progress.

## Planned

- M4: Discovery engine — feature-complete as of Card #39 (classic,
      current, emerging discovery, the Candidate Pool, the Recent Track
      Filter, Metadata Enrichment, and discovery provenance are all done,
      see Cards 33-39 above); candidate persistence remains open for M5+
- M5: Musical ranking & bridges — Candidate Scoring Model defined (Card
      #40); Fit factor implemented (Card #41); Freshness factor
      implemented (Card #42); Discovery Bonus, Diversity, Repetition
      Penalty, Playlist Fit, ranking, and editorial selection remain open
- M6: Curator experience
- M7: Weekly editorial workflow
- M8: Feedback & evolution
