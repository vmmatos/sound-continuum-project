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
  application shell (`App.vue` renders `views/HomeView.vue` directly)
  proving the build/dev pipeline works, and a `services/` boundary
  (`services/health.ts`) calling the backend's `GET /health`
  non-blockingly. No state-management library is installed — Pinia (Card
  17) was removed after sitting unused with zero stores; reinstate it if a
  future card introduces real shared application state (see
  `decisions.md`). No routing library either — Vue Router was removed
  after sitting on exactly one route (`/`) since Card 16 with no
  navigation anywhere in the app; reinstate it if a future card introduces
  a second route. Frontend/backend separation is physical: `frontend/` vs.
  everything else at repo root; the frontend talks to the backend only
  over HTTP, via `VITE_API_BASE_URL`.
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
  that responsibility, not speculatively. (`router/` no longer exists —
  removed alongside Vue Router, see above.)
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

- Single-artist retrieval is implemented (Card 29, see
  [`docs/spotify-integration.md`](../spotify-integration.md) §25), still in
  `backend/internal/spotify/` — no new package, no scope change.
  `GET /api/spotify/artists/{id}` is new (`Client.Artist`, `Service.Artist`,
  `ArtistHandler`). `Artist` (shared by `Track.Artists`, `Album.Artists`,
  `SearchResult.Artists`) gained `type`, `images`, `genres`. No
  `followers`/`popularity` — removed by Spotify for Development Mode. No
  bulk artist retrieval (`GET /artists?ids=` unavailable in Development
  Mode) and no `/artists/{id}/top-tracks` (also removed) — no compatibility
  wrapper for either. `genres` is deprecated/optional metadata, decoded but
  not used for Sound Continuum genre classification. An empty artist ID is
  rejected client-side (`ErrEmptyArtistID`) before a request is built,
  mapped to `400` by the handler, mirroring `Track`'s pattern. `403`/`404`
  both fall through the existing `writeSpotifyError` default (`502`) — no
  new status-code special case. No automatic enrichment — `GetTrack` and
  playlist items still do not auto-call `Client.Artist`. Verified live: a
  real artist from an owned playlist's track decoded correctly (name, id,
  uri, external URL, three images); `genres` was `null` for that artist,
  decoding cleanly; a nonexistent artist ID returned `502`.

- The official Sound Continuum Spotify playlist can be created and
  persisted (Card 30, see
  [`docs/spotify-integration.md`](../spotify-integration.md) §26), still
  in `backend/internal/spotify/` — no new package. `POST
  /api/spotify/playlist` is new (`Client.CreatePlaylist`,
  `Service.InitializeOfficialPlaylist`, `InitializePlaylistHandler`),
  the project's first write endpoint and first non-GET route. A new
  singleton `official_playlist` SQLite table (`CHECK (id = 1)`, same
  pattern as `spotify_connection`) persists the Spotify playlist ID,
  name, URL, and creation time. Idempotency is guaranteed by construction:
  if a local row already exists, it's returned with no Spotify call at
  all — Spotify is only ever called to create the playlist once, on the
  very first successful initialization. OAuth scope gained
  `playlist-modify-public` (`user-read-private playlist-read-private
  playlist-modify-public`) — the curator must reconnect once. The
  backend's dev CORS middleware (`backend/cmd/server/main.go`) now
  answers `OPTIONS` preflight requests, needed because this is the first
  JSON `POST` endpoint (every prior route was a CORS-simple `GET`).
  `frontend/src/services/spotify.ts` + `HomeView.vue` add an "Initialize
  official playlist" button and a link once created — still no Pinia
  store, page-local state as established by Card 25. The playlist is
  created empty (public, non-collaborative, fixed description) — no
  track-adding, no editorial workflow, no Sound Continuum playlist
  discovery-by-name. Real Spotify verification against the live account
  is done: the official playlist exists (public, non-collaborative,
  correct name/description, 0 tracks), and a second initialization call
  confirmed no duplicate was created.

- M4 (Discovery & Curation) begins with the `CandidateTrack` domain model
  (Card 31, see [`decisions.md`](decisions.md)) in a new package,
  `backend/internal/candidate/` — the project's first domain package
  outside `spotify`/`health`. A `CandidateTrack` represents a track
  discovered and under editorial consideration for a future Sound
  Continuum edition; it is **not** a Spotify track and **not** yet a
  playlist selection. It carries its own internal `candidate.ID`, distinct
  from the external `SpotifyTrackID` (empty unless `Source ==
  SourceSpotify`), and only the track metadata (title/artist) the
  editorial process actually needs — no copy of Spotify's `Track`/
  `Artist`/`Album`. `Source`, `Status`, and the `EditorialNote`/
  `PotentialConnection` fields originally covered more ground (`Last.fm`/
  `Manual` sources, an `under review`/`selected`/`rejected` lifecycle,
  free-text curator notes) than any workflow actually constructed — a
  repo-wide audit found zero call sites for any of it, so it was trimmed
  back to what's used (`Source: Spotify` only, `Status: discovered` only,
  no editorial-note fields) rather than left as unused scaffolding; see
  [`decisions.md`](decisions.md). Supported editorial categories remain
  `Past`, `Present`, `Emerging`, `New Release` (kept ahead of use
  deliberately — see decisions.md). No ID generation, no persistence, no
  API endpoint, no CRUD, and no frontend UI exist yet.

- `CandidateTrack` also carries an independent `Type` (Card 32, see
  [`decisions.md`](decisions.md)): `Classic`, `Current`, `Discovery`,
  same `type X string` + `const` + `Valid()` convention as `Category`,
  required and validated at construction like `Category` (not defaulted
  like `Status`). `Type` describes *how* a candidate entered the
  editorial process; `Category` describes *where* it sits editorially —
  the two are deliberately unrelated, with no compatibility or inference
  rule between them: e.g. a `Discovery`-type candidate can carry
  `Category: Past`, and a `Classic`-type candidate can carry
  `Category: New Release`. Discovery is not synonymous with Emerging or
  New Release. Still no persistence, API endpoint, CRUD, or frontend UI.

- Classic music discovery is implemented (Card 33, see
  [`decisions.md`](decisions.md) and
  [`spotify-integration.md`](../spotify-integration.md) §27) — M4's first
  discovery workflow, in a new package, `backend/internal/discovery/`, no
  new dependency, reusing the existing `spotify.Service` and
  `candidate.CandidateTrack`. `discovery.PastReferenceArtists` (15 names)
  is the single canonical source for Sound Continuum's "Past" reference
  artists — the first place this list exists anywhere in the repo.
  `discovery.Service.DiscoverClassic` resolves each artist via exact,
  case-insensitive Spotify `Search` name match (no fuzzy matching,
  unresolved artists surfaced on `Result.UnresolvedArtists`, never
  dropped), walks `GET /artists/{id}/albums` (fixed
  `include_groups=album,single`) then `GET /albums/{id}/tracks` — two new
  `spotify.Client`/`spotify.Service` methods, `ArtistAlbums`/
  `AlbumTracks`, reusing the existing `Album`/`Track`/`Paging[T]` types —
  and builds `CandidateTrack`s with
  `Source=Spotify/Type=Classic/Category=Past/Status=discovered`,
  deduplicated by Spotify track ID. A candidate's `ID` reuses its Spotify
  track ID directly (no UUID dependency). Bounded by a constructor-
  injected `Config` (`DefaultConfig()`: 5 albums/artist, 10 tracks/album,
  150 candidates total) — no unbounded catalogue crawl. Per-artist/album/
  track failures are recorded on `Result.Failures` and the run continues;
  only a Spotify connection failure (`ErrNotConnected`/`ErrInvalidGrant`)
  aborts the whole run. `POST /api/discovery/classic` exposes it — no
  persistence, no request body, no popularity/ranking of any kind. Real
  Spotify verification: all 15 reference artists resolve, no unresolved
  artists, no failures, no duplicate candidates, official playlist
  unchanged, no candidate auto-selected. No Last.fm, no AI, no musical
  bridge logic, no Current/Discovery-type discovery — all explicitly
  deferred to later M4/M5 cards.

- Current music discovery is implemented (Card 34, see
  [`decisions.md`](decisions.md)) — M4's second discovery workflow, still in
  `backend/internal/discovery/`, no new package, reusing Card 33's
  `spotify.Service`/`candidate.CandidateTrack`/`spotifyCatalogue` seam/
  `walkPages`/`resolveArtist`/`isConnectionError`/`writeDiscoveryError`.
  `discovery.PresentReferenceArtists` (15 names) is the canonical "Present"
  reference-artist list, alongside Card 33's `PastReferenceArtists`.
  `discovery.Service.DiscoverCurrent` resolves each artist the same way as
  `DiscoverClassic`, then walks `ArtistAlbums` (already fixed to
  `include_groups=album,single`, so singles are included and compilations/
  appears-on stay excluded), filters releases to a configurable recency
  window (`CurrentConfig.LookbackDays`, default 90 days), sorts survivors
  release-date-descending (Spotify's artist-albums order isn't documented
  as chronological) with Spotify album ID as a stable tiebreaker, then
  walks `AlbumTracks` on the most recent ones and builds
  `CandidateTrack`s with `Source=Spotify/Type=Current/Category=Present/
  Status=discovered`, deduplicated by Spotify track ID exactly like
  Classic. Spotify's partial release dates (`release_date_precision`
  "month"/"year") resolve to the earliest instant consistent with that
  precision — a conservative reading for a recency filter, documented on
  `parseReleaseDate`. `discovery.Service` now holds two configs
  (`classicCfg Config`, `currentCfg CurrentConfig`) — `NewService` takes
  both. `CurrentConfig` adds `MaxAlbumsScannedPerArtist` (raw releases
  fetched per artist, default 50 — one Spotify page) as a separate bound
  from `MaxAlbumsPerArtist` (most recent qualifying releases kept per
  artist after filtering/sorting, default 5), plus the existing
  `MaxTracksPerAlbum`/`MaxTotalCandidates` shape (10/150 defaults).
  Building this surfaced a real, undocumented Spotify Development Mode
  constraint: `GET /artists/{id}/albums` rejects `limit>10` for this app
  (confirmed live), unlike `GET /albums/{id}/tracks` (still fine at 50) —
  `walkPages` now takes an explicit per-call page-size cap
  (`maxArtistAlbumsPageSize=10`/`maxAlbumTracksPageSize=50`) instead of one
  hardcoded 50 shared by every endpoint; Classic's behavior is unchanged.
  `discovery.Result` gained one field, `ReleasesOutsideWindow`, always 0
  for Classic. `POST /api/discovery/current` exposes it — no persistence,
  no request body, no popularity/ranking. No Last.fm, no AI, no Emerging/
  New Release discovery, no musical bridge logic, no editorial selection —
  all still deferred.

- Emerging artist discovery is implemented (Card 35, see
  [`decisions.md`](decisions.md)) — M4's third and final discovery
  workflow, and the repo's first Last.fm integration. A new standalone
  `internal/lastfm` package (one file, no provider abstraction) wraps
  `artist.getsimilar` only: `lastfm.Client.SimilarArtists` injects the API
  key, sets an identifiable `User-Agent`, and returns typed
  `[]lastfm.SimilarArtist{Name, Match}` — a missing API key returns
  `ErrMissingAPIKey` without making a request, which is what makes a
  missing `LASTFM_API_KEY` fail the whole discovery run loudly (503)
  instead of silently returning an empty result.
  `discovery.EmergingReferenceArtists` (15 names) is the canonical
  "Emerging" seed-artist list, alongside `PastReferenceArtists`/
  `PresentReferenceArtists`; a new `isCanonicalReferenceArtist` helper
  excludes any Last.fm result already present in any of the three lists
  (case-insensitive, trimmed). `discovery.Service.DiscoverEmerging` walks
  each seed once through `SimilarArtists` (exactly one hop — it never calls
  Last.fm with a discovered artist's name), deduplicates discovered artists
  by lowercased/trimmed name, resolves the remainder through the same
  `resolveArtist` Classic/Current already use, then explores their recent
  catalogue via a new shared `recentTracksForArtist` helper — extracted
  from `DiscoverCurrent`'s own scan/filter/sort/truncate/track-walk block,
  since Card 35 needed the exact same "recent" definition a third time and
  duplicating it again would have been real, not speculative, duplication.
  `discovery.Service` now also holds a `lastfm similarArtistFinder`
  dependency and an `emergingCfg EmergingConfig` (own struct, same
  reasoning as `Config`/`CurrentConfig` staying separate: it owns
  Last.fm-hop-specific bounds — `MaxSimilarPerSeed`, `MaxDiscoveredArtists`
  — the other two workflows have no use for). `NewService` now takes a
  `*lastfm.Client` alongside `*spotify.Service`. Candidates use
  `Source=Spotify` (Last.fm is a discovery signal, not a candidate-identity
  source — `candidate.Source` has no Last.fm value at all, see
  decisions.md), `Type=Discovery`, `Category=Emerging`,
  `Status=discovered`, deduplicated by Spotify track ID exactly like
  Classic/Current. `discovery.Result` gained `EmergingProvenance
  []ArtistProvenance{SeedArtist, DiscoveredArtist, Match}` — Last.fm's
  own similarity value, kept as discovery metadata only and never used to
  select, order, or score anything; it lives on `Result`, not on
  `CandidateTrack`, which has no structured provenance field. One seed's
  Last.fm failure is recorded on `Result.Failures` (`Stage: "similar"`) and
  the run continues; only a missing Last.fm API key or a Spotify connection
  failure aborts the whole run. `POST /api/discovery/emerging` exposes it —
  no persistence, no request body, no popularity/ranking of any kind. Real
  Last.fm + Spotify verification: `artist.getsimilar` confirmed live
  against a seed artist; the full bounded workflow correctly aborts with a
  clear 503 when Spotify has no stored connection (same pre-existing
  behavior as Classic/Current) and with a clear 503 when `LASTFM_API_KEY`
  is unset — never a silent empty success either way. No AI, no musical
  bridge logic, no editorial selection, no persistence — all still
  deferred. M4 (Discovery Engine) is now feature-complete for its three
  planned discovery workflows.

- A Candidate Pool now orchestrates all three discovery workflows (Card #36,
  see [`decisions.md`](decisions.md)) — the first card where Sound
  Continuum acts as one discovery pipeline rather than three isolated
  features. Added entirely inside `backend/internal/discovery`
  (`pool.go`), no new package: `Service.DiscoverPool` calls the existing
  `DiscoverClassic`/`DiscoverCurrent`/`DiscoverEmerging` unchanged (no
  discovery logic duplicated), merges their candidates, and deduplicates
  cross-workflow duplicates by Spotify track ID via a documented, `Type`-
  keyed priority rule (`Classic` > `Current` > `Discovery`/Emerging — see
  decisions.md for the full reasoning) that is stable regardless of merge
  order. `CandidatePool` carries the merged, deduplicated `Candidates`,
  counts (`TotalCandidates`/`ClassicCandidates`/`CurrentCandidates`/
  `EmergingCandidates`/`DuplicatesRemoved`), and each workflow's own
  unmodified `Result` (`ClassicResult`/`CurrentResult`/`EmergingResult`),
  so per-workflow counts, unresolved artists, failures, and Emerging's
  provenance all stay inspectable. Every candidate keeps the
  `Source`/`Type`/`Category`/`Status` its originating workflow assigned —
  nothing is normalized or re-classified. Ordering is deterministic and
  non-popularity-based (`Type` priority, then `TrackArtist`, `TrackTitle`,
  `SpotifyTrackID` — `CandidateTrack` has no release date field, so that
  leg of the card's suggested order was dropped). `DiscoverPool` always
  runs all three workflows and never returns a top-level error: a workflow
  abort (Spotify connection failure, or for Emerging a missing
  `LASTFM_API_KEY`) is recorded on `CandidatePool.WorkflowErrors` while the
  other two workflows' candidates are kept — a partial failure never
  discards successful results, and a configuration failure never looks
  like a silent empty pool. `POST /api/candidates/pool` exposes it
  (`PoolHandler`) — no persistence (the pool is a transient, in-memory
  discovery-run result, matching every other M4 workflow), no request
  body, no frontend UI, no ranking, no editorial selection. 12 new unit
  tests (`pool_test.go`) cover inclusion of all three workflows, merge/
  counts, cross-workflow dedup and the priority rule, per-workflow partial
  failure (Classic-only and Current-only), an Emerging per-seed failure
  that continues the run, a missing-Last.fm-config abort, provider errors
  surfacing their real message, deterministic ordering, no editorial
  selection, empty results, and the HTTP handler round trip — all against
  fakes, no real network calls.

  The mandatory real end-to-end run partially succeeded and surfaced a
  genuine, previously-undiscovered Spotify Development Mode constraint:
  `GET /artists/{id}/albums` returned `429` with `Retry-After: ~4h30m`
  (confirmed by inspecting the real response) for this app, evidently
  after Card #36's own testing (several full-pool attempts against the
  live API, alongside the cumulative testing across Cards 33-35) exhausted
  a long-lived, per-app quota specific to that endpoint — not a per-request
  burst limit. What *was* verified live and working: the existing Spotify
  connection (`tintim_22`) and OAuth/token reuse, real `Search`-based
  artist resolution, and real Last.fm `artist.getsimilar` calls (confirmed
  genuine similarity matches, e.g. seed "Toxe" → "Rabit" at `Match: 1.0`).
  Critically, `DiscoverPool`'s partial-failure design was verified correct
  against this *real*, unplanned provider failure: every `ArtistAlbums`
  429 was recorded as a non-aborting `Stage: "albums"` entry on the
  relevant workflow's `Failures`, `WorkflowErrors` correctly stayed empty
  (429 is not a connection/config abort condition), and the official
  playlist's item count was confirmed unchanged (0 before and after) via
  the live server. What could **not** be confirmed live this session,
  because of the rate limit: an actual Spotify track surviving all the way
  into a merged `CandidatePool.Candidates` entry. This should be re-run
  once the rate limit clears (the response's own `Retry-After` puts that
  at roughly 4.5 hours from the session's testing) — the orchestration
  code itself needs no change; this is a live-provider quota state, not an
  implementation gap.

- A Recent Track Filter now sits between Candidate Pool aggregation and
  editorial consideration (Card #37, see [`decisions.md`](decisions.md)) —
  added entirely inside `backend/internal/discovery`
  (`recent_track_filter.go`), no new package. `Service.FilterRecentTracks`
  reads the official Sound Continuum playlist's items via the existing
  `spotify.Service.PlaylistItems` (already paginating; the filter now
  walks every page rather than a bounded sample — a dedicated loop, not
  Card #33's `walkPages`, since the whole playlist must be inspected, not
  a bounded crawl), builds an in-memory Spotify-track-ID → most-recent-
  `added_at` index (episodes/unavailable/malformed items skipped
  individually, duplicate track IDs keep the max `added_at`), and splits
  every `CandidatePool` candidate into `EligibleCandidates` or
  `RecentlyUsedCandidates` by comparing each candidate's `SpotifyTrackID`
  against that index. A track is recently used when its `added_at` is
  `>= now - LookbackDays` (documented `>=` boundary, unit-tested exactly
  at the edge). The lookback (`discovery.DefaultRecentTrackLookbackDays`,
  28) is a plain `int` on `Service`, not its own config struct — a
  single-field, single-caller config type would have been unnecessary
  machinery — and is overridable via
  `RECENT_TRACK_LOOKBACK_DAYS` in `cmd/server/main.go` — the only backend
  numeric config read from an env var today, following the existing
  `PORT`/`SQLITE_PATH` precedent rather than the discovery package's
  constructor-only convention, since this one is a curator-tunable
  editorial knob. `discovery.Service` gained an unexported `now func()
  time.Time` field (defaulted to `time.Now` in `NewService`, overridden
  directly by same-package tests) — the repo's first clock-injection
  seam, added only because the card requires deterministic boundary
  tests; every other `time.Now()` call in the codebase is still direct.
  `spotify.Service` gained `OfficialPlaylist(ctx)`, a thin read-only
  wrapper over the existing `Store.GetOfficialPlaylist` returning the new
  `ErrOfficialPlaylistNotConfigured` sentinel when no playlist has been
  initialized yet (distinct from a transport/API failure). No OAuth scope
  change — `playlist-read-private` (Card #26) already covers reading the
  official playlist's items regardless of its public/private flag, since
  reads always go through the curator's authenticated token. No
  `candidate.Status` change — a recently-used candidate keeps `Status:
  discovered`; Eligible/Recently-Used is a filter-result-level split only.
  `POST /api/candidates/pool` (`PoolHandler`) still runs
  `DiscoverPool` unchanged (always 200, partial-failure-tolerant across
  the three workflows) and now also calls `FilterRecentTracks`, attaching
  the result to a new `CandidatePool.RecentTrackFilter` field — but unlike
  `DiscoverPool`, a filter failure (official playlist not configured,
  Spotify connection/API failure) makes the endpoint return an HTTP error
  instead of a pool, so a temporary Spotify outage can never silently
  bypass the repetition guardrail. 26 new unit tests (15 in
  `recent_track_filter_test.go`, covering every case Card #37 lists —
  empty/no-match/all-recently-used playlists, exact boundary, duplicate
  entries, multi-page pagination, episodes/unavailable items ignored, a
  candidate with no Spotify ID, consistency across Type/Category, Status
  preservation, and both failure modes — plus 2 new `PoolHandler` tests
  and 2 new `spotify.Service.OfficialPlaylist` tests) — all against fakes,
  no real network calls.

  Real Spotify verification (via Docker Compose) confirmed the official
  playlist had been deleted outside the app (see the Card #30 idempotency
  limitation noted in decisions.md), so a fresh one was created through
  the existing, unmodified `InitializeOfficialPlaylist` flow before
  testing. Against that fresh, empty, public playlist: `POST
  /api/candidates/pool` returned 200 with `RecentTrackFilter{LookbackDays:
  28, TotalCandidates: 0, EligibleCount: 0, RecentlyUsedCount: 0,
  PlaylistTracksInspected: 0}` — the empty-playlist path, `RECENT_TRACK_
  LOOKBACK_DAYS`'s default, and pagination termination all exercised for
  real; the playlist's item count was confirmed unchanged (0 before and
  after) via `GET /api/spotify/playlists/{id}`, confirming no write.
  `TotalCandidates` was 0 because all three discovery workflows hit the
  same `GET /artists/{id}/albums` 429 rate limit already documented for
  Card #36 — a pre-existing Spotify Development Mode quota state, not a
  Card #37 defect — so an actual candidate landing in `RecentlyUsedCandidates`
  against real data was not confirmed live; that specific behavior is
  covered by the fake-based unit tests instead. The official playlist was
  not modified to manufacture a test case, per the card's own constraint.

- Candidates now carry structured Spotify metadata (Card #38, see
  [`decisions.md`](decisions.md)) — a new `candidate.CandidateMetadata`
  (`backend/internal/candidate/metadata.go`, package `candidate`): `Title`,
  `Artists []CandidateArtist` (`SpotifyArtistID`/`Name`/`SpotifyURL` each,
  never collapsed to a string), `Album CandidateAlbum`
  (`SpotifyAlbumID`/`Name`/`AlbumType`/`ReleaseDate`/
  `ReleaseDatePrecision`/`Artwork []CandidateImage`/`SpotifyURL`),
  `DurationMS`, `Explicit`, `SpotifyURL`, `SpotifyURI`. `CandidateTrack`
  gains one new field, `Metadata *CandidateMetadata` — nil until enriched,
  never populated with fabricated/placeholder values; every other existing
  field (`ID`/`SpotifyTrackID`/`Source`/`Category`/`Type`/`Status`/
  `TrackTitle`/`TrackArtist`) is unchanged. `ReleaseDate`/
  `ReleaseDatePrecision` preserve exactly what Spotify reports (a
  year-only release date stays year-only) — precision is never invented.
  Enrichment itself (`backend/internal/discovery/metadata_enrichment.go`,
  `Service.EnrichCandidateMetadata` + `mapSpotifyTrack`) runs inside
  `discovery`, not `candidate` or `spotify` — same reasoning already
  recorded for `pool.go`/`recent_track_filter.go`: `discovery` is the one
  package that legitimately depends on both. It calls the existing Card #28
  `spotify.Service.Track` (added to the `spotifyCatalogue` seam, no new
  Spotify client/auth/endpoint) once per eligible candidate — sequentially,
  no worker pool — and maps the response's already-embedded `Artists` list
  directly, with no separate `GET /artists/{id}` call. No popularity,
  followers, or genres are mapped — deprecated Spotify fields, not ranking
  signals this card introduces. `PoolHandler`
  (`backend/internal/discovery/pool.go`) now runs
  `EnrichCandidateMetadata` over `RecentTrackFilter.EligibleCandidates`
  only, after the Card #37 filter — recently-used candidates are never
  enriched. `CandidatePool` gained one field, `MetadataEnrichment
  EnrichmentResult`. A candidate with no Spotify track ID (no Manual/
  Last.fm `candidate.Source` value exists yet — see decisions.md) is passed
  through unmodified and counted on `EnrichmentResult.SkippedCount`, no
  Spotify request attempted. A Spotify connection failure
  (`ErrNotConnected`/`ErrInvalidGrant`) aborts the whole
  `EnrichCandidateMetadata` call and `PoolHandler` returns an HTTP error,
  matching Card #37's own precedent; any other per-candidate failure
  (not found, rate limited, unauthorized, malformed response, generic API
  failure) is recorded on `EnrichmentResult.Failures` with that candidate
  kept unenriched (`Metadata` stays nil) and the run continues — the
  endpoint still returns 200, matching the existing Classic/Current/
  Emerging per-item-failure-continues convention. 20 new unit tests
  (`metadata_enrichment_test.go`) cover every mapping/precision/failure
  case the card lists, plus one new `PoolHandler` integration test
  confirming eligible candidates are enriched and recently-used candidates
  are not. No persistence, no new endpoint, no ranking/scoring, no new
  package, no new dependency.

- Candidates now carry structured discovery provenance (Card #39, M4's
  final card, see [`decisions.md`](decisions.md)) — a new
  `candidate.DiscoveryProvenance` (`backend/internal/candidate/
  provenance.go`, package `candidate`): `Method` (a new `DiscoveryMethod`
  enum — `classic_reference_artist`/`current_reference_artist`/
  `lastfm_similar_artist`/`manual`, same `type X string` + `const` +
  `Valid()` convention as `Type`/`Category`/`Status`/`Source`), `Provider`
  (a new `ProvenanceProvider` enum — `Spotify`/`Last.fm`/empty), `Seed`
  and `DiscoveredArtist` (`*candidate.SeedArtist{Provider,
  ProviderArtistID, Name}`, a stable provider ID preferred over a display
  name where one is actually resolved), and `LastFMMatch` (Last.fm's own
  similarity score, discovery metadata only, mirroring
  `discovery.ArtistProvenance.Match`). `CandidateTrack` gains one new
  field, `Provenance []DiscoveryProvenance` — additive alongside Card
  #38's `Metadata` field; `Validate()` rejects an entry with an invalid
  `Method`, nothing else. The pre-existing free-text `DiscoveryReason`
  field (Card #31) was removed as part of this card, not kept alongside
  `Provenance`: its three fixed per-workflow strings were fully subsumed
  by `Provenance[].Method`, a typed enum carrying the identical fact
  structurally, so keeping both would have been duplicated information
  with no caller reading `DiscoveryReason` for anything `Method` doesn't
  already answer.
  This is deliberately **not** the same concept as the existing `Source`
  field: `Source` answers "which provider supplied the track" (unchanged,
  still `SourceSpotify` only); `Provenance` answers "how did this
  candidate enter the pipeline." An Emerging candidate can therefore carry
  `Source: Spotify` and provenance `Provider: Last.fm` simultaneously — see
  `decisions.md`. `DiscoverClassic`/`DiscoverCurrent`
  (`backend/internal/discovery/discovery.go`) each attach one provenance
  entry per candidate using the reference artist's Spotify ID already
  resolved earlier in the same loop (`classic_reference_artist`/
  `current_reference_artist`, `Provider: Spotify`, `Seed` = the reference
  artist) — no extra Spotify call. `DiscoverEmerging` attaches
  `lastfm_similar_artist` provenance (`Provider: Last.fm`, `Seed` = the
  original Emerging reference-artist seed name — never itself resolved
  through Spotify, so no provider ID — `DiscoveredArtist` = the Last.fm
  result's already-resolved Spotify artist, `LastFMMatch` = Last.fm's own
  similarity value) — reusing the same `artistID`/`sim.Match` already in
  scope, no extra Last.fm or Spotify call either. `pool.go`'s
  `DiscoverPool` dedup step no longer discards the losing candidate's
  provenance when the same Spotify track is found by more than one
  workflow: it now merges both candidates' `Provenance` via a new
  `candidate.MergeProvenance` (dedup by method + seed/discovered-artist
  identity) while classification (`Type`/`Category`/`Status`, decided by
  `candidateTypePriority`) is unchanged — deduplication removes the
  duplicate candidate, not useful provenance. `FilterRecentTracks` and
  `EnrichCandidateMetadata` needed no code changes — both already copy
  `CandidateTrack` by value without touching unrelated fields, so
  `Provenance` survives automatically; this is confirmed by new tests, not
  just by inspection. No handler changes were needed either —
  `PoolHandler`/`ClassicHandler`/etc. already JSON-encode the whole
  `CandidateTrack`/`CandidatePool` with no field allowlist, so `Provenance`
  is exposed through the existing `POST /api/candidates/pool` (and
  `/api/discovery/*`) responses automatically. No `Source` enum value was
  added for manual candidates — a curator manually picking a real Spotify
  track keeps `Source: Spotify`; only its provenance `Method` is `manual`,
  with no `Seed`/`Provider` fabricated (`candidate/provenance_test.go`
  covers this directly, since no workflow in this repo constructs a
  manual candidate today). 20 new unit tests across `candidate` (enum
  validation, `MergeProvenance` union/dedup/empty-input cases, manual
  provenance with no fabricated data) and `discovery`
  (Classic/Current/Emerging provenance field correctness including no
  extra provider calls, Pool dedup provenance merging, Source-vs-Provider
  distinction, and provenance preservation through `FilterRecentTracks`/
  `EnrichCandidateMetadata`) — all against fakes, no real network calls.

  Real Spotify + Last.fm verification (via `go run ./cmd/server` with
  `dev/.env`/`dev/.secrets.env` loaded) confirmed the existing Spotify
  connection (`tintim_22`) and real artist resolution via `Search` for all
  three workflows (`ArtistsInspected` matched each reference/seed list's
  size, `UnresolvedArtists` populated normally, e.g. non-Latin-script
  Last.fm results the app can't resolve by exact name match — expected,
  pre-existing behavior). No candidate could be constructed and no
  provenance field could be exercised against live data this session,
  because `GET /artists/{id}/albums` is still returning `429` for every
  artist — the same per-app Spotify Development Mode quota state first
  documented in Card #36 and still in effect as of this session's testing
  (confirmed live in each workflow's `Result.Failures`, all
  `Stage: "albums"`). This is a pre-existing live-provider quota state,
  not a Card #39 defect; correctness of the provenance fields for real
  data is covered by the fake-based unit tests instead, matching the
  precedent already accepted for Cards #36/#37. Separately, `POST
  /api/candidates/pool` itself currently 502s because the locally-recorded
  official playlist (`4lzlrXudAc6oruI9ASksuq`) no longer exists on
  Spotify's side (`GET /api/spotify/playlists/{id}` also 404s) — a
  recurrence of the known Card #30/#37 idempotency limitation (the
  official playlist was deleted outside the app again), not touched or
  caused by this card. Every discovery/pool call made during this
  session's verification was read-only by construction (no write endpoint
  was invoked), so no playlist was created, modified, or published to.
  M4 (Discovery Engine) is now feature-complete: discovery, aggregation,
  filtering, enrichment, and provenance all exist; persistence and musical
  bridge logic remain open for M5+.

- M5 (Musical Ranking & Bridges) begins with the Candidate Scoring Model
  (Card #40, see [`decisions.md`](decisions.md) and the dedicated
  [`docs/scoring-model.md`](../scoring-model.md)) — a new package,
  `backend/internal/scoring`, defining the conceptual structure of a Sound
  Continuum candidate score without implementing any individual factor's
  algorithm. `scoring.Factors` holds six normalized `[0,1]` values (Fit,
  Freshness, DiscoveryBonus, Diversity, PlaylistFit, RepetitionPenalty)
  each as `*float64` — nil meaning "not yet available," reusing the
  nil-means-unavailable convention already established by
  `candidate.DiscoveryProvenance.LastFMMatch`. `scoring.Weights` (five
  positive weights summing to 1.0, plus an independent
  `RepetitionWeight`) is constructor-supplied via `DefaultWeights()`, not
  scattered as inline constants — Fit 0.35, PlaylistFit 0.25,
  DiscoveryBonus 0.15, Diversity 0.15, Freshness 0.10, RepetitionWeight
  0.30, all justified against the manifesto in
  `docs/scoring-model.md`. `scoring.Calculate` combines them: missing
  positive factors are excluded from a weight-renormalized average (never
  substituted as 0, so incomplete data never reads as a negative signal),
  and the repetition penalty applies as a multiplicative discount
  (`FinalScore = BaseScore * (1 - penalty*weight)`) rather than a
  subtraction — this keeps `FinalScore` naturally bounded in `[0,1]` with
  no clamping, since a subtractive penalty could otherwise drive an
  already-low base score negative. `scoring.CandidateScore` exposes every
  factor, the weights, `AvailableWeight`, and `FinalScore` as independent
  fields — the explainability mechanism the card requires, no opaque
  single number. `ModelVersion = "v1"` is a plain string label, no
  history or persistence. This card does not modify `CandidateTrack`, does
  not rank or sort a candidate pool, does not select or reject candidates,
  and is not wired into `discovery`/`main.go` — no production code
  constructs real factor values yet, since every per-factor algorithm is
  future M5 work; `Calculate` is exercised only by its own unit tests.
  16 new unit tests (`scoring/score_test.go`) cover weight/factor range
  validation, the weight-sum rule (including its float tolerance
  boundary), every missing-factor combination (including the "zero
  positive factors available" edge case, which yields a nil
  `FinalScore`), the multiplicative repetition combination (including its
  bounded-at-zero edge case), error propagation on invalid input,
  explainability, and determinism.

- M5 continues with Card #41 (Define Musical Fit): a new package,
  `backend/internal/musicaldna`, holds one shared dimension vocabulary,
  `Profile` (`Mood`, `Energy`, `Texture`, `CulturalInfluence`, each
  optional), reused for three roles — a candidate's editorially-tagged
  characteristics, `ProjectDNA` (Sound Continuum's stable, project-wide
  identity), and `WeeklyDirection` (the current edition's explicit
  direction, `EditionID` + `Profile` + free-text `Notes`,
  `NewWeeklyDirection`-constructed, no UI/persistence). `ProjectDNA`'s
  `Profile` is empty by default (`DefaultProjectDNA()`) — the manifesto
  and M1 describe editorial process/philosophy, not concrete musical
  values, so every dimension stays unset until a real, durable
  project-wide trait is documented (see `decisions.md`'s per-dimension
  mapping). `backend/internal/scoring/fit.go` implements the Fit factor:
  `CalculateFit(candidateProfile, project, direction, weights)` compares
  the candidate's `Profile` against both `ProjectDNA.Profile` and
  `WeeklyDirection.Profile` using case-insensitive, trimmed exact-string
  match per dimension (no fuzzy/embedding similarity, no genre matching).
  Missing dimensions are renormalized within each comparison, and the two
  comparisons are renormalized against each other via `FitWeights`
  (`WeeklyWeight` 0.75, `ProjectWeight` 0.25, plus `Mood` 0.35/`Energy`
  0.25/`Texture` 0.25/`CulturalInfluence` 0.15) — the same
  missing-data-renormalization idiom `scoring.Calculate` already uses for
  `Factors`. Because `ProjectDNA` is empty by construction today, Fit is
  driven entirely by `WeeklyDirection` in practice. `Fit` is `nil`, never
  a fabricated `0.0`, when nothing is comparable on either side.
  `FitResult.Dimensions` carries all 8 (dimension × component) entries for
  explainability. No confidence score and no editorial-override mechanism
  were built (both are documented, deliberate omissions — `Factors.Fit`
  is already a plain settable `*float64`). The candidate-side `Profile` is
  a plain function argument, not a new `CandidateTrack` field — Card #38's
  struct is unchanged, and no persistence was introduced for candidate
  tagging. `CalculateFit` never takes a `candidate.CandidateTrack`, so it
  is independent of `CandidateType`/`Category`/release date/discovery
  provenance/playlist sequence by construction. 21 new unit/integration
  tests across `musicaldna` and `scoring` (`fit_test.go`,
  `fit_integration_test.go`) cover strong/weak fit, partial and fully
  missing data, weight validation, determinism, explicit independence
  from classification/freshness/provenance, explainability, score
  integration, and one integration test building a realistic
  `CandidateTrack` through `CandidateScore` with no live Spotify call.
  `scoring.Factors`/`scoring.Weights`/`scoring.Calculate` are unchanged;
  no ranking, selection, or playlist mutation. See
  [`docs/scoring-model.md`](../scoring-model.md) for the full Fit model
  and [`decisions.md`](decisions.md) for the manifesto-to-`ProjectDNA`
  mapping.

- M5 continues with Card #42 (Define Freshness):
  `backend/internal/scoring/freshness.go` implements the Freshness factor —
  how long since a candidate's Spotify track ID last appeared in the
  official Sound Continuum playlist, distinct from release date. Never-used
  candidates get exactly `1.0`; used candidates follow a half-life recovery
  curve, `Freshness(t) = 1 − 0.5^(t / HalfLifeDays)` with a default
  60-day half-life (`scoring.DefaultFreshnessConfig`) — continuous,
  monotonically increasing, asymptotic toward but never reaching `1.0`, and
  with no discontinuity across the 28-day Recent Track Filter boundary
  (Card #37). `CalculateFreshness(lastUsedAt *time.Time, now time.Time,
  config FreshnessConfig)` never reads the system clock or calls Spotify —
  `now` and `lastUsedAt` are both explicit inputs, so results are
  deterministic. Playlist-history retrieval is reused, not duplicated:
  `discovery.Service.PlaylistTrackHistory` (a newly exported wrapper around
  Card #37's existing `recentTrackIndex` pagination walk) is the one
  mechanism both `FilterRecentTracks` and Freshness's wiring depend on;
  `scoring.FreshnessLastUsedAt(history map[string]time.Time,
  spotifyTrackID string)` looks up one candidate's most recent appearance
  from that history without `scoring` importing `discovery`, keeping
  `CalculateFreshness` I/O-free. A retrieval failure (auth/API/network
  error) still propagates as an error, never as a false `Freshness = 1.0`;
  a successfully-retrieved empty playlist legitimately yields `1.0` for
  everyone. `CalculateFreshness` takes no `candidate.CandidateTrack`, so it
  cannot see (and cannot be affected by) `CandidateType`, `Category`,
  release date, or discovery provenance. 28 new tests across
  `scoring/freshness_test.go`, `scoring/freshness_integration_test.go`,
  `discovery/recent_track_filter_test.go`, and
  `discovery/freshness_integration_test.go` cover the curve's gradient and
  boundary behavior, normalization, monotonicity, determinism,
  independence, duplicate-entry/empty-playlist/retrieval-failure handling,
  and the full discovery-history-to-scoring reuse path. `scoring.Factors`,
  `scoring.Weights` (Freshness weight unchanged at 0.10), and
  `discovery.FilterRecentTracks`'s own behavior are unchanged; no ranking,
  selection, or playlist mutation. See
  [`docs/scoring-model.md`](../scoring-model.md) for the full Freshness
  model and [`decisions.md`](decisions.md) for the curve-choice and
  history-reuse reasoning.

- M5 continues with Card #43 (Define Discovery Bonus):
  `backend/internal/scoring/discovery_bonus.go` implements the Discovery
  Bonus factor — the editorial value of surfacing a `CategoryEmerging`
  candidate as a genuine discovery, never a proxy for how unknown it is.
  `CalculateDiscoveryBonus(category candidate.Category,
  editorialDiscoveryValue *float64) (DiscoveryBonusResult, error)` v1's
  formula is deliberately trivial — `DiscoveryBonus =
  editorialDiscoveryValue` — applied only when `category ==
  candidate.CategoryEmerging` (necessary but never sufficient on its own)
  and a value was explicitly supplied by the caller; there is no `if
  Category == Emerging: DiscoveryBonus = fixed value` shortcut anywhere.
  `DiscoveryBonusResult{Value, Category, Eligible, Supplied}` makes "not
  eligible," "eligible but unassessed," and "eligible and scored"
  independently inspectable: `Value` is `nil` for the first two cases and
  an explicit `0.0` editorial assessment ("no meaningful discovery value")
  is preserved exactly, never collapsed into `nil`. No
  `DiscoveryBonusWeights`/config struct was added — v1's identity formula
  has no configurable knob, unlike `FitWeights`'s dimension weights or
  `FreshnessConfig`'s half-life. `CalculateDiscoveryBonus` takes no
  `candidate.CandidateTrack`, `CandidateType`, discovery provenance, or
  Last.fm similarity/match value — `candidate.DiscoveryProvenance.LastFMMatch`
  remains exactly what it was documented as since Card #39, discovery
  metadata only, never a ranking signal — nor Spotify popularity/followers
  (which don't exist anywhere in this codebase's Spotify model, removed for
  Development Mode), release date, Fit, Freshness, Diversity, PlaylistFit,
  or RepetitionPenalty; none of those are formula inputs, by construction.
  A candidate's real `CandidateTrack.Provenance` stays available to a
  caller for editorial explanation alongside the resulting
  `DiscoveryBonusResult` — the two are shown together, never combined into
  one number, and provenance is never threaded into the calculation itself.
  `ErrDiscoveryBonusValueOutOfRange` (added to `scoring/errors.go`) rejects
  a non-nil editorial value outside `[0,1]` or NaN before any computation,
  mirroring `CalculateFit`/`CalculateFreshness`'s validate-first convention.
  22 new tests across `scoring/discovery_bonus_test.go` (eligibility/supply
  combinations, explicit-zero-vs-nil, the category boundary, validation,
  determinism, and eight independence regression guards covering
  CandidateType/provenance/Last.fm match/release date/Fit/Freshness/
  Diversity+PlaylistFit/RepetitionPenalty) and
  `scoring/discovery_bonus_integration_test.go` (a full candidate →
  `CandidateScore` path with real provenance left untouched, a
  non-Emerging candidate staying nil end-to-end, the existing `0.15`
  default weight confirmed unchanged, and `Calculate`'s missing-factor
  renormalization confirmed correct with Discovery Bonus present) exercise
  every case, all against fakes/pure values, no real network calls.
  `scoring.Factors`, `scoring.Weights`, `scoring.Calculate`,
  `candidate.CandidateTrack`, `scoring.CalculateFit`,
  `scoring.CalculateFreshness`, and `discovery/pool.go` are all unchanged;
  no ranking, selection, `Status` mutation, playlist mutation, persistence,
  or HTTP/API wiring — `CalculateDiscoveryBonus` stays exactly as unwired
  from any pipeline/handler as `CalculateFit`/`CalculateFreshness` are
  today. See [`docs/scoring-model.md`](../scoring-model.md) for the full
  Discovery Bonus model and [`decisions.md`](decisions.md) for the
  exclusion-list and parameter-vs-field reasoning.

- M5 continues with Card #44 (Define Diversity):
  `backend/internal/scoring/diversity.go` implements the Diversity factor —
  does a candidate contribute meaningful variation to the *current edition
  being assembled*, across Artist/Era/Sound concentration, discouraging
  concentration without rewarding difference for its own sake. All new code
  stays inside the existing `scoring` package — no new package, no new
  persisted Edition entity. `CurrentEditionContext{Tracks []EditionTrack}`
  (each `EditionTrack`: `ArtistSpotifyIDs []string`, `Era *string`,
  `Sound musicaldna.Profile`) is a small, transient, in-memory,
  function-level input distinct from `candidate.CandidateTrack`, the
  candidate pool, and the official historical Spotify playlist (Freshness's
  and the Recent Track Filter's domain, Cards #37/#42) — Diversity is
  contextual to the edition in progress, never calculated from a candidate
  in isolation. `scoring.CalculateDiversity(candidateArtistSpotifyIDs
  []string, candidateEra *string, candidateSound musicaldna.Profile, ctx
  *CurrentEditionContext) DiversityResult` takes no
  `candidate.CandidateTrack`/`CandidateType`/`Category`/`Source`/
  `DiscoveryProvenance`/other-factor value, so it is independent of all of
  them by construction — the same guarantee Fit/Freshness/Discovery Bonus
  already give; unlike those three it returns no error, since nothing it
  accepts is a caller-supplied float or config needing range validation.
  Artist Diversity reuses the candidate's existing Spotify artist identity
  (`candidate.CandidateArtist.SpotifyArtistID`) against edition-context
  concentration only — deliberately not Repetition Penalty, which uses
  playlist history instead. Era Diversity uses a coarse decade
  (`scoring.DiversityEra(releaseDate string) *string`, first 4 digits as
  year) with no CandidateType/Category/genre inference. Sound Diversity
  reuses `musicaldna.Profile` (Card #41's vocabulary) but computes
  independently from `CalculateFit`: concentration against the edition's
  population of profiles, not a target match — no genre-as-sound-proxy, no
  Spotify Audio Features. All three dimensions and each Sound sub-dimension
  share one deterministic, bounded formula,
  `contribution(occurrences) = 1 / (1 + occurrences)` — no hardcoded
  per-occurrence thresholds. Missing dimensions are excluded and the
  remaining ones renormalized (never treated as `0`), the same idiom
  `Calculate`/`CalculateFit` already use. A `nil` context and a non-nil
  context with zero tracks both yield `Diversity = nil`, for two different,
  independently inspectable reasons
  (`DiversityResult.ContextProvided`/`EditionEmpty`) — an empty edition is
  explicitly never read as "maximally diverse." `DiversityResult` exposes
  `Value`, `ContextProvided`/`EditionSize`/`EditionEmpty`, and each of
  `Artist`/`Era`/`Sound` (with per-dimension `Available`/`Occurrences`/
  `Value`, and Sound additionally breaking out all four
  Mood/Energy/Texture/CulturalInfluence sub-dimensions) for explainability.
  `Factors.Diversity`/`Weights.Diversity = 0.15`/`score.go` are unchanged —
  a caller threads `DiversityResult.Value` into `Factors.Diversity` exactly
  as the other positive factors already do. 27 new unit/integration tests
  across `scoring/diversity_test.go` and
  `scoring/diversity_integration_test.go` cover every case Card #44 lists:
  new/repeated/increasingly-concentrated artist, different/concentrated
  era, different/similar sound, empty edition vs. missing context (both
  nil, distinguishable), missing-dimension renormalization,
  explicit-availability-vs-unavailable distinction, normalization bounds,
  determinism, `CandidateScore` integration, the unchanged `0.15` weight,
  missing-factor renormalization, per-dimension independence from each
  other, and independence from `CandidateType`/`Category`/provenance/
  Last.fm match/the other four factors — one integration test builds a
  real `candidate.CandidateTrack` through metadata enrichment and
  provenance exactly like Cards #38/#39 produce, with no live Spotify/
  Last.fm call. `candidate`, `musicaldna`, `discovery`, and `score.go` are
  all unchanged; no ranking, selection, persistence, or HTTP/API wiring —
  `CalculateDiversity` stays exactly as unwired from any pipeline/handler
  as `CalculateFit`/`CalculateFreshness`/`CalculateDiscoveryBonus` are
  today. See [`docs/scoring-model.md`](../scoring-model.md) for the full
  Diversity model and [`decisions.md`](decisions.md) for the
  package-reuse, context-representation, and formula-choice reasoning.

- M5 continues with Card #45 (Define Repetition Penalty), M5's sixth and
  final scoring factor:
  `backend/internal/scoring/repetition_penalty.go` implements
  `CalculateRepetitionPenalty(trackLastUsedAt, artistLastUsedAt *time.Time,
  now time.Time, config RepetitionPenaltyConfig) (RepetitionPenaltyResult,
  error)` — a soft scoring signal for recent track/artist reuse, distinct
  from the Card #37 hard 28-day Recent Track Filter and the Card #42
  Freshness factor, following the M1 repetition philosophy that repetition
  is never an absolute prohibition. Two dimensions, each a linear decay
  from severity `1.0` at 0 days since last use (consecutive reuse, the
  strongest case) to exactly `0` at a 90-day horizon
  (`DefaultRepetitionHorizonDays`, its own constant — deliberately distinct
  from both the 28-day filter and Freshness's 60-day half-life), combined
  as `max(TrackRepetition, ArtistRepetition)` rather than summed, since a
  track repetition event is already an artist repetition event and summing
  would double-count it. Track identity is the Spotify track ID (reusing
  `scoring.FreshnessLastUsedAt`, Card #42's lookup helper, directly — no
  new track lookup needed); artist identity reuses
  `candidate.CandidateMetadata.Artists[].SpotifyArtistID` via a new
  `scoring.RepetitionArtistLastUsedAt(artistHistory map[string]time.Time,
  spotifyArtistIDs []string) *time.Time`, which returns the most recent
  occurrence across all of a candidate's artists — the same identity
  Diversity (Card #44) already reuses, no second artist-identity model.
  `backend/internal/discovery/recent_track_filter.go` gained
  `Service.PlaylistArtistHistory(ctx) (map[string]time.Time, error)`,
  built from the same single pagination walk as Card #37/#42's
  `PlaylistTrackHistory` (refactored into `recentTrackAndArtistIndex`,
  with `recentTrackIndex`'s own return value unchanged and re-verified by
  a new test) — no second playlist-history retrieval mechanism, no extra
  Spotify calls beyond what `PlaylistTrackHistory` already makes. A
  genuinely empty official playlist yields `RepetitionPenalty = 0` for
  every candidate; an unavailable playlist (not configured, Spotify
  API/connection failure) propagates as an error from both
  `PlaylistTrackHistory` and `PlaylistArtistHistory`, never a silent `0`.
  `now` is always an explicit parameter, never the system clock, and clock
  skew clamps to 0 days, mirroring `CalculateFreshness`.
  `scoring.Factors.RepetitionPenalty`/`scoring.Weights.RepetitionWeight =
  0.30`/`scoring.Calculate`'s multiplicative combination formula are all
  unchanged — a caller threads `RepetitionPenaltyResult.Value` into
  `Factors.RepetitionPenalty` exactly as the five positive factors already
  do; `RepetitionPenalty = 1.0` still produces `FinalScore = BaseScore ×
  0.70`, confirmed by a deterministic fixture test. 30 new unit/integration
  tests across `scoring/repetition_penalty_test.go`,
  `scoring/repetition_penalty_integration_test.go`,
  `discovery/recent_track_filter_test.go` (new `PlaylistArtistHistory`
  cases), and `discovery/repetition_penalty_integration_test.go` cover
  every case Card #45 lists: never-used track/artist, recently-used track,
  older-than-the-hard-filter-but-inside-the-horizon track, outside-horizon
  track, new track by a recently-used artist, new track by a never-used
  artist, both recent (max combination, no double-count), consecutive
  reuse, duplicate playlist entries, empty playlist, missing/unavailable
  history, clock-skew clamping, normalization bounds, determinism, Spotify
  track/artist identity, independence from the other five factors (proven
  by construction — the function accepts no `CandidateTrack`), the
  `CandidateScore`/`0.30`-weight/`FinalScore` integration, and that a high
  penalty never itself rejects a candidate. `candidate`, `musicaldna`,
  `scoring.Factors`/`Weights`/`Calculate`, and `discovery.
  FilterRecentTracks`'s own behavior are all unchanged — like Fit,
  Freshness, Discovery Bonus, and Diversity before it,
  `CalculateRepetitionPenalty` stays unwired from any
  pipeline/handler/`main.go`. M5 now has five of its six factors
  implemented (Fit, Freshness, Discovery Bonus, Diversity, Repetition
  Penalty); only Playlist Fit, ranking, and automatic selection remain
  open. See [`docs/scoring-model.md`](../scoring-model.md) for the full
  Repetition Penalty model and [`decisions.md`](decisions.md) for the
  max-vs-sum, horizon-choice, and playlist-history-extension reasoning.

- M5 continues with Card #46 (Define Playlist Fit), M5's final scoring
  factor — the Candidate Scoring Model's six factors are now all
  implemented: `backend/internal/scoring/playlist_fit.go` implements
  `CalculatePlaylistFit(candidateSound musicaldna.Profile, ctx
  *CurrentEditionContext) PlaylistFitResult` — a sequence-aware factor
  distinct from Musical Fit, Diversity, Repetition Penalty, Discovery
  Bonus, Freshness, and popularity, whose sole transition anchor is the
  single immediately-preceding track in the edition being assembled
  (`ctx.Tracks[len(ctx.Tracks)-1]`), never every track in the edition and
  never an arbitrary insertion position. It reuses
  `scoring.CurrentEditionContext`/`EditionTrack` (Card #44) as-is — no
  second "current edition" representation, and `EditionTrack` gains no
  ID/title field. Four dimensions (Mood, Energy, Texture, Cultural
  Influence), equal fixed 25% weight each, renormalized over whichever are
  available on both the candidate and the previous track — no
  `PlaylistFitWeights` config struct. Mood/Texture/Cultural Influence score
  a case-insensitive trimmed exact match as `1.0` and any mismatch as a
  flat `0.5` baseline (not `0.0`), so an intentional contrast still
  contributes meaningfully rather than being scored as a failure. Energy
  uses a small, fixed 5-level ordinal vocabulary ("very low".."very high")
  local to `playlist_fit.go` only (`musicaldna.Profile.Energy` stays
  `*string` everywhere else) with `score = 1 - |levelDiff|/4` — symmetric,
  so low→medium and medium→low score identically and a transition toward
  higher energy is never automatically "better" — falling back to the
  same match-or-baseline rule when either value is outside the vocabulary.
  `PlaylistFit = nil` (never a fabricated number) whenever there is no
  previous track (nil context or a non-nil context with zero tracks, both
  distinguishable via `PlaylistFitResult.ContextProvided`/
  `PreviousTrackIndex`, without a redundant third bool) or no dimension is
  comparable at all. `CalculatePlaylistFit` takes only
  `musicaldna.Profile`/`*CurrentEditionContext` — never a
  `candidate.CandidateTrack` or any other factor's calculated value — so it
  is independent of CandidateType, Category, Fit, Freshness, Discovery
  Bonus, Diversity, Repetition Penalty, popularity, and historical
  (non-edition) playlist usage by construction, and it is never derived
  from any other factor's score. `PlaylistFitResult` carries `Value`,
  `ContextProvided`, `PreviousTrackIndex`, and one
  `PlaylistFitDimensionResult` per dimension for explainability.
  `scoring.Factors.PlaylistFit`/`scoring.Weights.PlaylistFit = 0.25`
  (Card #40)/`scoring.Calculate`'s combination formula are all
  unchanged — a caller threads `PlaylistFitResult.Value` into
  `Factors.PlaylistFit` exactly as the other five factors already do;
  `score.go` needed no changes. 17 new unit/integration test functions
  (two with table-driven subtests, for energy progression/symmetry and
  normalization bounds) across `scoring/playlist_fit_test.go` and
  `scoring/playlist_fit_integration_test.go` cover every case Card #46
  lists: strong/weak transitions, intentional contrast scoring strongly,
  no-previous-track and missing-profile nil cases, partial-dimension
  renormalization, energy progression and symmetry, mood/texture/cultural-
  influence relationships without exact equality, independence from every
  other factor and from CandidateType/Category/popularity/historical
  playlist usage (proven by construction — the function accepts no
  `CandidateTrack`), the A→B→C→D sequence-anchor case (must use C→D, never
  A→D or B→D), the `CandidateScore.Factors.PlaylistFit` integration, the
  unchanged `0.25` weight, 0.0–1.0 normalization bounds, and determinism.
  `candidate`, `musicaldna`, `diversity.go`, `fit.go`, and `score.go` are
  all unchanged. Like Fit/Freshness/Discovery Bonus/Diversity/Repetition
  Penalty before it, `CalculatePlaylistFit` stays unwired from any
  pipeline/handler — a pure, tested library function. M5 now has all six
  scoring factors implemented (Fit, Freshness, Discovery Bonus, Diversity,
  Repetition Penalty, Playlist Fit); only ranking and automatic selection
  remain open. See [`docs/scoring-model.md`](../scoring-model.md) for the
  full Playlist Fit model and [`decisions.md`](decisions.md) for the
  mismatch-baseline, energy-vocabulary, context-reuse, and
  nil-case-representation reasoning.

- Card #47 (Explore Musical Similarity) is complete — investigation only,
  no production code. A musical similarity engine is not justified now and
  stays deferred. See [`decisions.md`](decisions.md) for the reasoning and
  [`docs/research/musical-similarity.md`](../research/musical-similarity.md)
  for the full investigation.

- Card #48 (Detect Potential Musical Bridges) is implemented:
  `scoring.DetectPotentialBridge` (`backend/internal/scoring/bridge.go`),
  a standalone, deterministic evidence-gathering function for a candidate
  pair — `BridgeResult.PotentialBridge = true/false`, never a ranking,
  weighted score, or automatic selection, and kept entirely outside
  `scoring.Factors`/`scoring.Calculate`/`CandidateScore`. It reuses Card
  #46's unexported `playlistFitMatchOrBaseline`/`playlistFitEnergyScore`
  comparators verbatim for the four explicit `musicaldna.Profile`
  dimensions (Mood/Energy/Texture/Cultural Influence — exact match 1.0,
  known mismatch 0.5 baseline, missing excluded) and adds three contextual
  signals from Card #47's catalogued-available set: shared Spotify artist
  identity, Last.fm artist similarity (caller-supplied, no Last.fm call
  inside the function), and release-era match (via the existing
  `scoring.DiversityEra`). `BridgeTrack` (artist IDs, era,
  `musicaldna.Profile`) mirrors Card #44's `EditionTrack` shape rather than
  introducing a third track representation. The decision rule is a plain
  evidence count, not a weighted formula: each of the 7 signals (4
  dimensions + 3 contextual) counts as a "meaningful relationship" when
  available and scoring above the existing 0.5 baseline; `PotentialBridge`
  is true once the count reaches `DefaultMinimumBridgeEvidence` (2) — one
  exported constant, documented and trivially changed later, satisfying the
  card's "no single signal is proof" requirement without inventing
  per-dimension weights. Genre is not a function parameter at all — it
  cannot contribute evidence by construction, matching Card #47's
  low-reliability finding. 15 new unit tests
  (`backend/internal/scoring/bridge_test.go`) cover strong/partial
  relationships, the evidence-count boundary at exactly 1 vs. 2, gradual vs.
  abrupt energy progression, contrast not being auto-rejected, no
  meaningful evidence, the Last.fm-alone-does-not-force-a-bridge case,
  missing data being excluded rather than negative, and determinism;
  independence from `CandidateScore`/Freshness/Diversity/Repetition
  Penalty/Playlist Fit/popularity holds by construction (no such type
  appears in the function's signature). No HTTP endpoint, no persistence,
  no frontend, no change to `candidate`/`musicaldna`/`discovery`/`lastfm`/
  any other `scoring` file. See
  [`docs/bridge-detection.md`](../bridge-detection.md) for the full model
  and [`decisions.md`](decisions.md) for the standalone-vs-weighted
  reasoning.

- Card #49 (Rank Candidate Tracks) is implemented:
  `scoring.Rank(entries []scoring.CandidateScoreEntry)
  []scoring.RankedCandidate` (`backend/internal/scoring/rank.go`), added
  to the existing `scoring` package — no new package. It is the final
  pure step in `Candidate Pool → Evaluate factors → CandidateScore → Sort
  → Ranked View → Curator review`: `Rank` takes candidates already paired
  with a `CandidateScore` built by the existing `Calculate` (Card #40's
  weighting/renormalization is reused unchanged, never duplicated) and
  sorts them `FinalScore` descending, with a nil `FinalScore` sorting
  last (treated as the weakest state, never a fabricated zero) and a
  `Candidate.ID`-ascending tie-breaker (chosen over `SpotifyTrackID`
  because `ID` is guaranteed non-empty for every valid `CandidateTrack`
  regardless of `Source`). `RankedCandidate` carries the original,
  untouched `CandidateTrack` (metadata/provenance/status preserved) plus
  the full `CandidateScore` and a 1-based `Rank` — ranking is curation
  assistance only: it never mutates `Status`, never calls `Calculate`
  itself, never filters or drops an entry, and `Rank 1` never means
  `Selected`. `Rank` has no knowledge of discovery, the candidate pool,
  or the Recent Track Filter — a caller only ever builds entries from
  `RecentTrackFilterResult.EligibleCandidates`, so a recently-used
  candidate structurally never reaches `Rank`. Like every other M5
  factor, `Rank` stays unwired from any pipeline or HTTP endpoint: real
  end-to-end ranking of live pool candidates needs per-candidate
  `musicaldna.Profile` tags, a `CurrentEditionContext`, and editorial
  discovery values, none of which any workflow collects yet (M6
  "Curator Experience" territory, out of this card's scope — confirmed
  with the curator during planning). 10 new unit tests
  (`scoring/rank_test.go`) cover basic ordering, weight integration via
  real `Calculate` output, missing-factor renormalization and
  explicit-zero preservation, repetition-penalty-reduced ordering,
  deterministic tie-breaking across repeated runs, full candidate
  metadata (status included) preservation, empty pool, single candidate,
  independence from popularity/release-date/genre/Last.fm-similarity, and
  nil-`FinalScore` ordering. No changes to `candidate`/`discovery`/any
  other `scoring`
  file; `scoring` still does not import `discovery`. M5 now only has
  automatic selection remaining open. See
  [`docs/scoring-model.md`](../scoring-model.md) for the full ranking
  model and [`decisions.md`](decisions.md) for the package-placement and
  tie-breaker reasoning.

- Card #50 (Generate Candidate Explanations) is implemented — M5's final
  card: `scoring.GenerateExplanation(ExplanationInput) CandidateExplanation`
  (`backend/internal/scoring/explanation.go`), added to the existing
  `scoring` package, no new package. It turns an already-computed
  `CandidateScore` (Card #40) — plus, optionally, a `BridgeResult` for the
  same candidate pair (Card #48) — into one short, deterministic,
  human-readable sentence (`CandidateExplanation.Text`) plus a structured
  `[]ExplanationReason` tag list for possible future UI use. The text is
  built entirely from `CandidateScore.Factors`' six already-normalized
  values (not each factor's full result struct — see decisions.md for why
  that's sufficient and not a shortcut) and, when supplied,
  `BridgeResult.Dimensions`/`Signals`; it never recomputes or mutates
  `CandidateScore`, `CandidateTrack.Status`, or ranking, and it is not a
  second scoring factor. A factor is mentioned only when its value clears
  `explanationMentionThreshold` (0.6) — Fit, Playlist Fit, Discovery Bonus,
  Diversity — and a `nil` or low factor is always silently omitted, never
  described as weak/zero. Freshness distinguishes "new to the Sound
  Continuum playlist" (`Factors.Freshness == 1.0`, the exact value only a
  never-used candidate can produce) from "has not appeared recently in
  Sound Continuum" (high but not exactly 1.0) — never release-date
  freshness. Repetition Penalty (`explanationRepetitionThreshold`, 0.3)
  appends a neutral caveat clause, never a rejection claim. At most
  `explanationMaxFragments` (3) factors are named, by value, to stay
  concise; a detected potential bridge is always mentioned regardless of
  that cap, described from the first dimension/signal `BridgeResult`
  actually reports as evidence — never genre, which `bridge.go` already
  excludes by construction. With nothing to report, the result is the
  neutral fallback `"No strong scoring signal available."`. 24 new unit/
  integration tests across `scoring/explanation_test.go` and
  `scoring/explanation_integration_test.go` cover every case Card #50
  lists: strong fit, never-used and not-recently-used freshness (plus a
  low-freshness-omitted case), discovery value, diversity contribution,
  playlist fit, a bridge mention from dimension evidence and from signal
  evidence alone, no bridge mention when not detected, a meaningful and a
  negligible repetition penalty, multiple strong factors staying capped/
  concise, missing factors never described negatively, an explicit-zero
  factor staying silently omitted, the no-signal fallback, a
  repetition-only sentence (not the generic fallback), determinism, and no
  mutation of the input `CandidateScore`. Like every other M5 factor since
  Card #41, `GenerateExplanation` stays unwired from any pipeline/handler —
  no HTTP endpoint, no persistence, no frontend, no change to
  `Rank`/`RankedCandidate`. `candidate`, `discovery`, and every other
  `scoring` file are unchanged. M5 (Musical Ranking & Bridges) is now
  feature-complete except automatic selection, which remains out of scope.
  See [`docs/scoring-model.md`](../scoring-model.md) for the full
  Candidate Explanations model and [`decisions.md`](decisions.md) for the
  Factors-vs-full-result-structs and stays-unwired reasoning.

- M6 (Curator Experience) begins with the Candidate Review screen (Card #51,
  a design/layout card — see [`decisions.md`](decisions.md)): a new frontend
  screen, `frontend/src/views/CandidateReviewView.vue`, rendered alongside
  the existing `HomeView` in `App.vue` (no Vue Router reinstated — still one
  flat page). `frontend/src/components/CandidateCard.vue` and
  `FactorBar.vue` are the first components under `frontend/src/components/`
  (previously nonexistent — everything lived in `HomeView.vue`).
  `frontend/src/types/candidateReview.ts` defines TypeScript types mirroring
  `candidate.CandidateTrack`/`CandidateMetadata`/`DiscoveryProvenance` and
  `scoring.Factors`/`CandidateScore`/`RankedCandidate`/`BridgeResult`/
  `CandidateExplanation` field-for-field in PascalCase, since those backend
  structs carry no JSON tags and would marshal with exact Go field names —
  unlike `spotify.ts`'s existing snake_case types, which mirror
  tagged handler-local structs. No backend endpoint combines
  Rank+GenerateExplanation+DetectPotentialBridge yet (only
  `POST /api/candidates/pool`, returning raw unscored candidates, exists),
  so `frontend/src/services/candidateReview.ts` exposes
  `getCandidateReviewPool()` returning realistic static mock data today,
  with a documented TODO describing the one-function swap to a real `fetch`
  once that endpoint exists — no other file needs to change when that
  happens. The screen shows rank, title/artist/album, `FinalScore` as a
  plain number (never stars or a judgement label; "Not yet scored" when
  `null`), the Card #50 explanation text, the six scoring factors as
  compact value+bar rows (skipping `null` factors), a potential-bridge
  section (only when `Bridge.PotentialBridge` is true, listing only
  dimensions/signals the result actually marks as evidence), and a
  provenance line for non-manual discovery methods. Candidates render in
  the order the mock data provides — no `.sort()`/`.reorder()` of any kind
  in the new code, matching the card's requirement that ranking stay
  backend-owned. The only action is a non-destructive "Open details"
  placeholder (console.log only) — no Select/Reject/Publish/Approve. Loading/
  empty/error states follow `HomeView.vue`'s existing typed-ref +
  `v-if`/`v-else-if` convention, no spinner/toast library. Initially shipped
  with no new dependency; a follow-up commit (below) later adopted Tailwind
  CSS v4 and seven shadcn-vue components for visual refinement — see
  `decisions.md`. No backend change, no persistence. Verified in a real
  browser (Playwright CLI screenshot against the Vite dev server) at
  desktop and ~400px widths — renders correctly, no horizontal scroll,
  bridge/provenance sections appear only where mock data supplies them,
  the unscored mock candidate shows "Not yet scored" and stays in its
  original (last) position.

- Card #51's Candidate Review screen got its first real visual direction
  (follow-up commit, see `decisions.md`) — Tailwind CSS v4 +
  `@tailwindcss/vite` and seven shadcn-vue components (`Card`, `Badge`,
  `Button`, `Progress`, `Separator`, `Tooltip`, `Skeleton`, style preset
  `reka-mira`, base color `zinc`) are now the frontend's first UI-level
  dependencies, scoped to this screen. `CandidateCard.vue` was rewritten
  around these components and split into `CandidateFactors.vue` (the six
  scoring-factor rows, now using shadcn `Progress`) and `BridgeEvidence.vue`
  (the "Musical bridge" section); `FactorBar.vue` was deleted (fully
  replaced by `CandidateFactors.vue`). The app is now dark-only
  (`<html class="dark">` in `index.html`) — `HomeView.vue` also renders
  dark, replacing its previous `prefers-color-scheme`-adaptive look, for one
  consistent visual identity. The palette overrides shadcn's zinc defaults
  with a near-black background and a warm amber/copper accent
  (`--primary`), used only for the `FinalScore` number, active `Progress`
  fill, and focus rings — never as a surface tint. Typography stays
  system-font-only (no new font dependency), with hierarchy built from
  weight/size/tracking alone: track title dominant, artist secondary, album
  quiet. `FinalScore` keeps its compact `#1`/`0.87`-stacked treatment (a
  `Tooltip` on the score clarifies "Internal ranking signal — not a quality
  rating," directly reflecting the product rule that the score is never a
  judgement). Provenance display was rewritten from raw
  `Method.replace(/_/g, ' ')` to an explicit editorial-label mapping (e.g.
  "Discovered via Last.fm · Related artist: ... (match 0.81)"). One mock
  `Explanation.Text` was shortened to stop describing internal
  not-yet-evaluated factor state ("Good diversity contribution and no
  recent repetition concerns.") — no other mock field changed; candidate
  ranks, order, `FinalScore` values, and the null-score entry are
  unchanged. No icon library was added (`@lucide/vue`, installed by the
  shadcn-vue CLI's init step, was removed again since none of the seven
  components use it). No selection/publish/reject action was introduced —
  "Open details" remains the only button on a candidate card. No backend
  change. Verified in a real browser (Playwright CLI screenshots) at
  desktop and ~400px widths.

- Candidate Review cards now display album artwork (Card #52) — a
  frontend-only presentation change, no backend code touched. Investigation
  found the data already flowed end to end: `candidate.CandidateAlbum.Artwork
  []CandidateImage` (Card #38) was already mapped from Spotify and already
  serialized unfiltered through `POST /api/candidates/pool`, and the
  frontend's `CandidateAlbum`/`CandidateImage` types
  (`frontend/src/types/candidateReview.ts`) already matched it
  field-for-field — `CandidateCard.vue` simply never read
  `Metadata.Album.Artwork` in its template. A new presentational component,
  `frontend/src/components/CandidateTrackMetadata.vue` (props: `title`,
  `artist`, `album: CandidateAlbum | null`), renders a 56px (`h-14 w-14`)
  square artwork thumbnail (`album.Artwork[0]?.URL`, `object-cover
  rounded-md`) beside the existing title/artist/album-name text, matching
  the existing `CandidateFactors.vue`/`BridgeEvidence.vue` sub-component
  pattern. Missing or failed artwork (empty `Artwork` array, or an `@error`
  on the `<img>`) falls back to a neutral `bg-muted`/`border-border` square
  of the same footprint — never a broken image, never an external
  placeholder, never fabricated artwork. Category/type badges stay exactly
  where they were in `CandidateCard.vue`'s header; `CardContent`/
  `CardFooter` (explanation, bridge, factors, provenance, "Open details")
  are untouched. Verified both paths in a real browser (Playwright,
  headless Chromium): the mock's three fake `i.scdn.co` artwork URLs and
  one empty-`Artwork` entry all correctly render the fallback (4 placeholder
  divs confirmed via DOM inspection); swapping one mock URL for a real
  `data:` image URI confirmed the `<img>` success path renders at the
  correct ~56-64px size (`complete: true`, correct `naturalWidth/Height`,
  bounding box ~63×63px) — that swap was a temporary local-only test, not
  committed. `npm run build` (`vue-tsc -b && vite build`) passes. No
  backend tests were run since no backend file changed.

- The Candidate Review screen now consumes a real backend endpoint (Card
  #53, see [`decisions.md`](decisions.md)) for the two scoring factors
  with genuine production inputs today — Freshness and Repetition
  Penalty. A new package, `backend/internal/review`, composes the
  existing `discovery` and `scoring` packages (`review -> discovery,
  scoring`; `scoring` still imports neither `discovery` nor `review`).
  `review.Service.ReviewPool` runs the real production pipeline —
  `discovery.Service.DiscoverPool` -> `FilterRecentTracks` ->
  `EnrichCandidateMetadata` -> per-candidate `scoring.CalculateFreshness`/
  `CalculateRepetitionPenalty` (playlist track/artist history fetched
  once per request via `PlaylistTrackHistory`/`PlaylistArtistHistory`,
  never per candidate) -> `scoring.Calculate` -> `scoring.Rank` ->
  `scoring.GenerateExplanation` — with no discovery/scoring algorithm
  duplicated, modified, or bypassed. `Fit`, `DiscoveryBonus`, `Diversity`,
  and `PlaylistFit` stay nil: their inputs (`musicaldna.Profile` tags,
  `CurrentEditionContext`, an editorial discovery value) don't exist in
  any production workflow yet, and none is fabricated. `Bridge`/
  `BridgeTrack` are always nil — Card #48's pairwise bridge detection has
  no caller here. `GET /api/candidates/review` exposes it
  (`cmd/server/main.go`); an empty eligible pool returns `200` with no
  entries, matching `PoolHandler`'s own "zero candidates is not an error"
  precedent. `review.Service` depends on `*discovery.Service` through a
  small unexported interface seam (`candidatePoolSource`), the same
  one-off pattern `discovery.spotifyCatalogue` already established (Card
  #33), needed because `discovery.Service`'s fields are unexported and
  its only constructor takes a concrete `*spotify.Service` — production
  callers (`main.go`) are unaffected. 17 new Go tests
  (`backend/internal/review`) cover every case Card #53 lists: no-history
  candidate, recently-played track, previously-used-artist-different-
  track, empty eligible pool, deterministic ranking, `AvailableWeight`
  exactly equal to Freshness's weight (`0.10` — the only positive factor
  this card populates), explanation sourced only from
  `scoring.GenerateExplanation`, and `Bridge` always nil, plus one HTTP
  handler integration test.

  `frontend/src/services/candidateReview.ts`'s mock
  `getCandidateReviewPool()` is replaced with the real `fetch` the file's
  own Card #51 TODO already specified — no other frontend file needed to
  change for that swap. One real contract mismatch was found and fixed:
  `CandidateReviewPool.EditionContext` had no real backend source (it
  would mean fabricating `CurrentEditionContext`/`WeeklyDirection`-style
  editorial content, an explicit non-goal), so it was removed from
  `frontend/src/types/candidateReview.ts` and
  `CandidateReviewView.vue` (the existing "Candidate Review" label and
  candidate count already identify the screen). `CandidateCard.vue` gains
  a small partial-score UI: when a scored candidate's `AvailableWeight`
  is below ~1.0 (every real candidate from this card, since Freshness
  alone is `0.10` of the five-factor weight pool), the score now shows a
  "Partial · N% signal" caption and an updated tooltip, distinct from
  both a fully-evaluated score and "Not yet scored" — `AvailableWeight`/
  `FinalScore` stay the literal source of truth, no new metric invented.

  Real Spotify-connected verification: the locally-recorded official
  playlist had again been deleted outside the app (a recurrence of the
  known Card #30/#37/#39 idempotency limitation), so a fresh one was
  created through the existing, unmodified `InitializeOfficialPlaylist`
  flow before testing. Against that fresh playlist, `GET
  /api/candidates/review` returned `200` with zero entries — all three
  discovery workflows hit the same long-lived Spotify Development Mode
  `GET /artists/{id}/albums` rate limit already documented for Cards
  #36/#37/#39 (confirmed via `POST /api/candidates/pool`'s own
  `Failures`), so the populated-candidate path could not be exercised
  against live data this session — a pre-existing provider quota state,
  not a Card #53 defect. The empty-pool path was confirmed for real, end
  to end, including in a real browser (no mock data, "No candidates
  available for review." rendered correctly at desktop and ~390px
  widths). The populated-card path (artwork, partial-score caption,
  factors, explanation, no bridge section) was confirmed in a real
  browser via a temporary, uncommitted local fixture swap in
  `candidateReview.ts`, restored immediately after — the same verification
  pattern Card #52 already established for a case live data couldn't
  reach this session. `npm run build` and `go build ./...`/`go vet
  ./...`/`go test ./...` all pass.

- Candidate Review can now tell apart a genuinely empty candidate pool from
  a Discovery-degraded one (Card #126, see [`decisions.md`](decisions.md)) —
  `review.ReviewPool` (`backend/internal/review/review.go`) gained
  `WorkflowErrors []discovery.WorkflowError` and
  `Failures []discovery.Failure`, both reused directly from
  `discovery.CandidatePool` (already in scope via the existing
  `DiscoverPool` call — no new Discovery call, no interface change, no new
  failure type). `Failures` merges the three workflows'
  (`ClassicResult`/`CurrentResult`/`EmergingResult`) own `Result.Failures`
  into one slice; `WorkflowErrors` is `CandidatePool.WorkflowErrors`
  unchanged. Both are informational only — a non-empty `Entries` list is
  never gated or altered by their presence, matching the card's "valid
  candidates still rank normally" requirement.
  `frontend/src/views/CandidateReviewView.vue` gained one new status,
  `'degraded'` (alongside the existing `'loading'`/`'ok'`/`'empty'`/
  `'error'`), shown only when `Entries` is empty and either field is
  non-empty, with generic copy that never names Spotify or "rate limited"
  specifically — the backend contract carries raw `Artist`/`Stage`/`Err`
  strings, not a structured failure category safe to narrate more
  specifically. `frontend/src/types/candidateReview.ts` gained matching
  `WorkflowError`/`DiscoveryFailure` interfaces (both array fields
  `| null`, since Go marshals a nil slice as `null`). 8 new Go tests
  (`backend/internal/review`) cover a clean empty pool, an empty pool with
  `WorkflowErrors` only, an empty pool with per-item `Failures` only,
  failures alongside valid candidates (which still rank/display
  normally), and the JSON contract (clean + populated) through the real
  HTTP handler — all existing review tests pass unmodified. No automated
  frontend test was added (explicit user decision this session — see
  `decisions.md`); the two new states were verified manually instead, in
  a real Chromium browser (a one-off `npx -p playwright` run, no new
  project dependency), both via mocked API responses for all three states
  and live against the real, Spotify-connected dev backend — which, while
  testing, was found still hitting the same long-lived Spotify Development
  Mode `GET /artists/{id}/albums` rate limit documented for Cards
  #36/#37/#39/#53 (confirmed via the response's own new `Failures` field,
  plus a stale `lastfm: missing API key` `WorkflowErrors` entry for
  Emerging in this session's environment) — exactly the scenario this card
  exists to make visible, now correctly rendered as "degraded" rather than
  a silent empty state. `go build ./...`/`go vet ./...`/`go test ./...`
  and `npm run build` all pass. `discovery`, scoring, ranking, and
  `EnrichCandidateMetadata`'s own `EnrichmentFailure`s are unchanged and
  out of scope — an enrichment failure keeps its candidate in `Entries`
  with `Metadata: nil`, so it can never cause the empty-vs-degraded
  ambiguity this card addresses.

- Bug fix (root cause of the Card #37 "Known limitation" entry above):
  `Service.InitializeOfficialPlaylist`
  (`backend/internal/spotify/handlers.go`) now checks Spotify itself
  before creating the official playlist, closing the gap that had
  produced real duplicate empty "Sound Continuum — Weekly Journey"
  playlists on the connected account across different local SQLite files
  (host `make backend-run` vs. Docker volumes). See
  [`decisions.md`](decisions.md) for the full design (exact, owned-name
  match via a new `allOwnedPlaylists` full-pagination walk; a local row
  still short-circuits with zero Spotify calls; more than one match fails
  closed with the new `*AmbiguousOfficialPlaylistError`/`409`, listing
  every candidate ID, rather than guessing). 6 new tests in
  `handlers_test.go` (adopt-single-match, ambiguous-fails-closed,
  pagination-finds-a-later-page-match, near-name-doesn't-match,
  other-owner-doesn't-match — plus 3 existing create-path tests updated
  to also stub the new `GET /v1/me/playlists` call). `go build
  ./...`/`go vet ./...`/`go test ./...` all pass (full suite).

  Live verification (`make docker-up`, Spotify connected) caught a real
  bug in this fix's own first version before it shipped: comparing
  `Playlist.Owner.ID` against `Connection.SpotifyUserID` never matched,
  because `SpotifyUserID` stores `Profile.UserID()` — which prefers
  Spotify's newer `account_id` (Card #53-era decision) — while
  `Owner.ID` is still only ever the legacy `id`. On this account those two
  values differ (`tintim_22` vs `MyvpUGE9N3`), so the owner check always
  failed, every existing playlist was treated as "no match," and the very
  first live test of this fix **created a fourth duplicate** instead of
  detecting the three that already existed — exactly the bug this card
  set out to close, reproduced by the fix itself. Caught immediately via
  the live test, not left in the shipped version: the comparison now
  calls `Service.Me` directly and compares against `Profile.ID` (the same
  legacy `id` namespace `Owner.ID` uses), re-verified live afterward —
  with all **four** real duplicates now present, a repeat
  `POST /api/spotify/playlist` correctly returned `409` listing all four
  IDs, and created no fifth. The errant fourth playlist
  (`6k8aY0Nh52Rgkf57FjP7iT`) is still sitting on the real account, same as
  the original three — the curator's call to delete, not done
  automatically.

  The immediate `GET /api/candidates/review` `503` on this Docker instance
  (empty local `official_playlist` row) was unblocked via a one-time
  manual reconciliation — `docker compose exec backend sh -c "apk add
  --no-cache sqlite && sqlite3 ..."` (the alpine image ships neither
  `sqlite3` nor `python3`) inserting a row pointing at one of the four
  existing duplicates (`3Ng7HvDMkGoRtLBjsGeA7V`, the original — arbitrary
  among the three pre-existing ones, specifically not the accidental
  fourth — all have 0 tracks) — not a code change, local dev-volume data
  only.

- Candidate Review now has a working Keep action, and Spotify Development
  Mode's ongoing rate limit no longer blocks local development of it (Card
  #56, see [`decisions.md`](decisions.md)). `candidate.StatusSelected` is
  reinstated (one of the three lifecycle values Card 31 had trimmed for
  lack of a caller) and set exclusively via a new persisted-selection
  overlay, not through `NewCandidateTrack`. A new package,
  `backend/internal/selection` (`store.go`/`service.go`), persists Keep
  decisions in a new SQLite table, `candidate_selection`, reusing the
  existing `*sql.DB`/`CREATE TABLE IF NOT EXISTS` pattern
  `spotify.Store` already established — no new datastore. `POST
  /api/candidates/{id}/keep` (`selection.Service.KeepHandler`) is
  idempotent by construction (`INSERT ... ON CONFLICT DO UPDATE`).
  `review.Service.ReviewPool` (`backend/internal/review/review.go`) gained
  a `selectionLookup` interface field (mirroring the existing
  `candidatePoolSource` seam) and overlays `AllSelected()`'s result onto
  every freshly-discovered candidate on each call — the candidate pool
  itself is still never persisted; only the Keep decision is, so a page
  refresh always reflects the latest persisted state with no extra
  frontend fetch. `frontend/src/components/CandidateCard.vue` gained a
  `Keep`/`Kept ✓` shadcn `Button` in the existing `CardFooter` (no new
  dependency, no layout redesign), with local `ref`s for in-flight/failure
  state mirroring `HomeView.vue`'s existing `officialPlaylist` pattern;
  `frontend/src/types/candidateReview.ts`'s `Status` widened to
  `'discovered' | 'selected'`; `frontend/src/services/candidateReview.ts`
  gained `keepCandidate()`.

  Separately, `SPOTIFY_MOCK_MODE` (default `false`, `dev/.env`) lets
  Candidate Discovery/Review run entirely offline against deterministic
  dummy data, with zero real Spotify API calls, while Spotify Development
  Mode's `GET /artists/{id}/albums` rate limit (documented since Card #36,
  still in effect as of Cards #53/#126) remains unresolved. The existing
  `discovery.spotifyCatalogue` interface (Card #33's test-only seam,
  already covering every Spotify touchpoint in the discovery pipeline) is
  reused for a second, production purpose:
  `discovery.NewService`'s first parameter is widened from the concrete
  `*spotify.Service` to this interface — existing production and test
  callers are unaffected, since both already satisfy it structurally. A new
  package, `backend/internal/spotifymock` (`catalogue.go`), implements the
  same 6 methods as pure, deterministic functions of their own input (no
  shared state, no `net/http` import anywhere in the package); `Search`
  echoes its query back as the returned artist's `Name` — required, not
  cosmetic, since `discovery.resolveArtist` only accepts an exact
  case-insensitive match — so every one of Sound Continuum's canonical
  reference artists resolves deterministically with no per-name data
  hand-authored. `cmd/server/main.go` is the single branch point between
  `spotifyService` and `spotifymock.NewCatalogue()`; `review.Service` and
  every discovery-derived endpoint (`/api/candidates/pool`,
  `/api/discovery/{classic,current,emerging}`, `/api/candidates/review`)
  share the one `discoveryService` instance, so mock mode covers the whole
  surface automatically with no scattered `if mock` checks. Direct
  `spotify.Service` endpoints (OAuth connect, `/api/spotify/*`, official
  playlist init) are unaffected by the flag. Last.fm is untouched — out of
  scope — so `DiscoverEmerging` still calls the real Last.fm API even in
  mock mode; a missing `LASTFM_API_KEY` degrades exactly as before (a
  `WorkflowError`, already surfaced by the Card #126 degraded state).

  Verified live (mock mode, real browser via Playwright): `GET
  /api/candidates/review` returned 240 deterministic Classic/Current mock
  candidates (Emerging degraded with a `WorkflowError` due to no
  `LASTFM_API_KEY` in the test environment — expected, not a defect);
  zero `spotify.com` network requests were observed from the browser;
  Keep/re-Keep (idempotent) both confirmed via the UI, with the kept state
  surviving a fresh page load. `go build ./...`/`go vet
  ./...`/`go test ./...` (all packages, including new
  `selection`/`spotifymock` packages) and `npm run build` all pass. Real
  Spotify behavior (`SPOTIFY_MOCK_MODE=false`, the default) is unchanged —
  not re-verified live this session since Spotify remains rate-limited
  (the same pre-existing condition documented for Cards #36/#37/#39/#53/
  #126), but every existing real-mode test still passes unmodified.

- Candidate Review's Keep action is now reversible, and a second decision
  state, Maybe, is implemented alongside it (Card #57, see
  [`decisions.md`](decisions.md)). `candidate.StatusUnderReview` is
  reinstated — the second of the three lifecycle values Card 31 trimmed,
  now with a real caller — `Status.Valid()` is a 3-way switch
  (`discovered`/`selected`/`under review`). No new persistence: the Card
  #56 `candidate_selection` table (one row per `candidate_id`) already
  stored `status` as a column, so Keep/Maybe are mutually exclusive by
  construction — writing one overwrites the other, no extra check needed.
  `selection.Store` gained `Maybe` (upsert `under review`) and `Clear`
  (idempotent delete, returns the candidate to neutral/`discovered`),
  alongside the existing `Keep`; `Keep`/`Maybe` now share an unexported
  `setStatus` helper and `AllSelected`/`AllUnderReview` share an unexported
  `allWithStatus` helper — small refactors, not new abstractions.
  `POST /api/candidates/{id}/maybe` and `POST /api/candidates/{id}/clear`
  are new, siblings of the existing `/keep` endpoint; all three stay plain,
  idempotent "set"/"clear" operations — the undo/toggle behavior the UI
  needs is a frontend decision about which endpoint to call, not backend
  state. `review.Service.ReviewPool`'s `selectionLookup` gained
  `AllUnderReview`, overlaid in a second pass right after the existing Keep
  overlay (the two can never collide, since the database can't have a
  candidate in both sets at once). `frontend/src/components/
  CandidateCard.vue` gained a `Maybe`/`Maybe ✓` button next to Keep (same
  shadcn `Button` convention, no new dependency); clicking an already-active
  button now calls a new `clearCandidateDecision()` instead of re-applying
  the same action, making Keep undoable for the first time;
  `frontend/src/types/candidateReview.ts`'s `Status` widened to
  `'discovered' | 'selected' | 'under review'`. No scoring/ranking/
  discovery/Spotify/mock-mode change. New/extended tests across
  `candidate`/`selection`/`review` (store-level, HTTP-level, and overlay
  coverage for Maybe, Clear, idempotency, and Keep/Maybe mutual exclusion
  in both directions) — `go build ./...`/`go vet ./...`/`go test ./...` and
  `npm run build` all pass. Manual validation was done against the running
  `SPOTIFY_MOCK_MODE=true` backend via `curl` (Keep, Maybe, Keep→Maybe,
  Clear, idempotent repeat-Clear, all returning the correct resulting
  `status` and reflected correctly by `GET /api/candidates/review`) — no
  browser-automation tool was available this session to repeat Card #56's
  Playwright-based UI click-through.

- Candidate Review's Keep/Maybe decisions now have a third, mutually
  exclusive counterpart: Skip (Card #58, see [`decisions.md`](decisions.md)).
  `candidate.StatusRejected` is reinstated — the last of the three lifecycle
  values Card 31 originally trimmed, now with a real caller — `Status.
  Valid()` is a 4-way switch (`discovered`/`selected`/`under review`/
  `rejected`). No new persistence or mechanism: the Card #56 `candidate_
  selection` table and its `setStatus`/`allWithStatus` helpers already
  generalize to a third status with zero schema change.
  `selection.Store` gained `Reject` (upsert `rejected`, calls the existing
  `setStatus`) and `AllRejected` (calls the existing `allWithStatus`), both
  one-line wrappers exactly mirroring `Keep`/`Maybe` and `AllSelected`/
  `AllUnderReview`. `POST /api/candidates/{id}/skip`
  (`selection.Service.RejectHandler`) is a new sibling of `/keep`/`/maybe`,
  named after the curator-facing action ("skip"), not the domain status
  ("rejected") — the same naming precedent `/keep`→`selected` and
  `/maybe`→`under review` already established. `review.Service.ReviewPool`'s
  `selectionLookup` gained `AllRejected`, overlaid in a third pass right
  after the existing Keep/Maybe overlays — the three can never collide,
  since the database can't have a candidate in more than one of the three
  sets at once. `frontend/src/components/CandidateCard.vue` gained a
  `Skip`/`Skipped ✓` button next to Maybe (same shadcn `Button` convention,
  no new dependency); each of Keep/Maybe/Skip's click handlers now clears
  both of the other two local flags on success (previously Keep/Maybe only
  cleared each other), so the UI never shows more than one active at once
  now that there are three. `frontend/src/types/candidateReview.ts`'s
  `Status` widened to `'discovered' | 'selected' | 'under review' |
  'rejected'`; `frontend/src/services/candidateReview.ts` gained
  `skipCandidate()`. No scoring/ranking/discovery/Spotify/mock-mode change;
  rejected candidates are kept, never physically removed from the pool. New/
  extended tests across `candidate`/`selection`/`review` (store-level,
  HTTP-level, and overlay coverage for Skip, idempotency, and Skip's mutual
  exclusion with Keep and Maybe in both directions) — `go build ./...`/
  `go vet ./...`/`go test ./...` and `npm run build` all pass. Manual
  validation was done against the running `SPOTIFY_MOCK_MODE=true` backend
  via `curl` (Skip, repeat-Skip, Skip↔Keep, Skip↔Maybe, Skip→Clear, all
  returning the correct resulting `status` and reflected correctly by
  `GET /api/candidates/review`) — no browser-automation tool was available
  this session either, same fallback as Card #57.

- Candidate Review now shows a Weekly Playlist Preview (Card #59, see
  [`decisions.md`](decisions.md)) — a frontend-only addition, no backend
  change. `frontend/src/components/WeeklyPlaylistPreview.vue` is a new
  component, rendered above the candidate-card list in
  `CandidateReviewView.vue`'s `'ok'` state, that filters the same `entries`
  array the screen already holds down to `Status === 'selected'` (Keep) and
  lists them in their existing order — `Entries`' own order is already
  `scoring.Rank`'s deterministic `FinalScore`-descending order (Card #49),
  reused unchanged rather than inventing a second ordering rule. A local,
  display-only `WEEKLY_TRACK_TARGET = 15` constant (no equivalent exists
  anywhere else in the codebase) drives a non-blocking "above the 15-track
  weekly target" note when exceeded — selected candidates are never
  truncated or discarded. Each row reuses `CandidateTrackMetadata.vue`
  (Card #52) unchanged for artwork/title/artist/album. Empty state ("No
  tracks kept yet...") follows the screen's existing
  `text-sm text-muted-foreground` convention.

  A real gap was found and fixed in the same change: `CandidateCard.vue`'s
  `setDecision` (Card #58) only ever flipped its own local
  `isSelected`/`isMaybe`/`isSkipped` refs — it never wrote back to
  `entry.Ranked.Candidate.Status` on the shared entry object, so a second
  reactive consumer of the same `entries` array (this new preview) would
  never see a Keep/Maybe/Skip/Clear change. `setDecision` now takes the
  target `CandidateTrack['Status']` and writes
  `props.entry.Ranked.Candidate.Status` on success (back to `'discovered'`
  on Clear) — `Status` stays the single source of truth for selection
  state, no second frontend selection state introduced.

  Known architectural limitation (not fixed by this card, out of its
  scope): the preview is derived entirely from one `GET
  /api/candidates/review` response, which rebuilds the candidate pool from
  scratch every call (the pool itself is never persisted — only the Keep/
  Maybe/Skip decision is, per Card #56). If discovery is fully degraded for
  a request (e.g. the long-documented Spotify Development Mode
  `GET /artists/{id}/albums` rate limit, Cards #36/#37/#39/#53/#126) and
  `Entries` comes back empty, a previously-Kept candidate's persisted
  decision still exists in SQLite but won't appear in that request's
  preview — fixing this would mean persisting enriched candidates
  themselves, which this card's scope explicitly excludes (no new
  persistence system).

  `go build ./...`/`go vet ./...`/`go test ./...` (no backend file changed)
  and `npm run build` both verified clean. Manual validation (mock mode,
  via a one-off `npx playwright` run against the real dev servers, no new
  project dependency): empty state confirmed with zero decisions; Keeping 3
  candidates (at ranks 1/2/6, clicked out of rank order) showed all 3 in
  rank order in the preview, not click order; Clear, Keep→Maybe, and
  Keep→Skip each correctly removed their candidate from the preview and
  decremented the count (3→2→1→0 across the sequence); Maybe→Keep correctly
  added it back; zero `spotify.com` network requests observed throughout.

- The Weekly Playlist Preview is now manually reorderable (Card #60, see
  [`decisions.md`](decisions.md)) — frontend-only, still no backend change.
  `WeeklyPlaylistPreview.vue` gained a local `order: ref<string[]>` (curator
  manual sequence, independent of `scoring.Rank`/`FinalScore`), reconciled
  against Keep/Maybe/Skip changes by a `watch` that appends newly-Kept IDs at
  the end and drops IDs no longer Kept, never resetting an existing manual
  position. Reordering uses native HTML5 Drag and Drop (a handle toggles
  `draggable` on mousedown, so only the handle starts a drag, not the whole
  row) plus Up/Down icon buttons per row as the keyboard-accessible
  equivalent — no new dependency (no sortable primitive exists in `reka-ui`
  or `@vueuse/core`). Numbering falls out of the existing `v-for` index, same
  as Card #59. Order state lives in `WeeklyPlaylistPreview.vue` itself, not
  lifted into `CandidateReviewView.vue` — the component is never torn down
  mid-session, so it already survives every Keep/Maybe/Skip click the same
  way `keptEntries` always has. Order is in-session only (lost on reload,
  falling back to rank order) — accepted per the card's own scope.
- Vitest + `@vue/test-utils` + `jsdom` are now installed (`frontend/
  package.json` devDependencies, `npm run test`, `vitest` config block in
  `vite.config.ts`) — the repo's first frontend test runner, added for Card
  #60 after Cards 22/#126 had both previously declined one; this card's
  test requirement outweighed that precedent, per an explicit user decision.
  `WeeklyPlaylistPreview.spec.ts` (6 cases) covers: only Kept tracks render
  in rank order; reordering via `moveTo` changes displayed order and
  renumbers; a newly-Kept track is appended without disturbing manual order;
  removing a Kept track preserves the remaining relative order; reordering
  never mutates `Status`; the empty state.
- Manual verification: real Chromium (via a one-off `playwright-core`
  script, no new project dependency), against the running
  `SPOTIFY_MOCK_MODE=true` backend. Confirmed live: the drag handle's
  `mousedown` correctly flips the row's `draggable` attribute to `true`;
  clicking the Up/Down buttons reorders tracks and renumbers correctly;
  keeping an additional candidate appends it without disturbing the existing
  manual order; skipping a Kept candidate removes it while the rest keep
  their relative order; zero `spotify.com` network requests during any
  reorder action; dark theme styling confirmed via screenshot. Native mouse-
  driven HTML5 drag itself (`dragstart`/`drop`) could not be triggered via
  Chromium's CDP-based synthetic input in this automated session — a known
  limitation of automating native drag-and-drop under headless/CDP control,
  not an application defect (the same `moveTo` function both the drag
  handlers and the Up/Down buttons call is what's exercised and confirmed
  working, by both the unit tests and the live keyboard path) — real mouse
  drag in an actual user session is expected to work and should be spot-
  checked by a human.

- The Weekly Playlist Preview now has an explicit confirm/lock step (Card
  #61, see [`decisions.md`](decisions.md)) — frontend-only, still no
  backend change. `WeeklyPlaylistPreview.vue` gained one new ref,
  `confirmed`, distinguishing the editable playlist (Card #60's `order`/
  `keptEntries`, unchanged) from the confirmed/locked playlist. Confirming
  (only actionable with ≥1 Kept track) shows a `role="status"` "Final
  playlist confirmed · N tracks · Ready to publish" banner, hides the drag
  handle and Up/Down reorder buttons, and shows an "Edit playlist" button
  that flips back to editable. No frozen snapshot of the confirmed order
  exists — `order` itself can't change while locked (controls hidden,
  `moveTo` guarded), and the Card #60 `watch(keptIds, ...)` reconciliation
  now invalidates (`confirmed = false`) whenever it would actually change
  `order`'s membership/sequence, so a Keep/Maybe/Skip change made elsewhere
  on `CandidateCard` — which has no notion of this component's lock state
  — can never leave a stale "confirmed" label next to a changed playlist.
  `confirmedPlaylist` (a computed exposing `keptEntries` itself, `null`
  unless confirmed — no precomputed shape like a separate track-ID list,
  since there's no caller yet to size one for) is exposed via `defineExpose`
  as the integration point a future M3 publishing flow would read via a
  template ref — no such flow exists yet in this repo (confirmed by
  inspection: only `InitializeOfficialPlaylist` exists, no add-tracks
  endpoint), so nothing consumes it yet. `CandidateTrack.Status` is never
  read or written by confirmation — selection decisions and playlist
  confirmation remain independent concepts. 10 new Vitest cases (16 total
  in `WeeklyPlaylistPreview.spec.ts`) cover capture, pre-confirm reorder,
  stability across unrelated re-renders, locking, auto-invalidation from an
  elsewhere Keep/Maybe/Skip change, Edit-playlist round-trips, the
  empty-state guard, and the exposed `confirmedPlaylist`; the existing 6
  Card #59/#60 tests are untouched. `go build ./...`/`go vet ./...`/`go
  test ./...` (no backend file changed) and `npm run build` both verified
  clean. Manual verification (mock mode, real Chromium via a one-off
  `playwright-core` script against the running dev servers, no new project
  dependency) confirmed live: keep 3 + reorder + confirm locks the exact
  reordered sequence; skipping a Kept candidate elsewhere while locked
  immediately reverts the preview to editable with the updated membership;
  Edit → reorder → reconfirm and Edit → Keep a new candidate → reconfirm
  both produce the correct new confirmed state; the Confirm button renders
  `disabled` with zero Kept tracks and the existing empty-state message is
  unchanged; keyboard activation (Tab + Enter) works for both Confirm and
  Edit, with focus moving to the other action's button each time; the
  status paragraph exposes `role="status"`; zero `spotify.com` network
  requests throughout. (A leftover `candidate_selection` SQLite row set from
  earlier Cards #56-#60 manual-verification sessions was found in the local
  dev database and cleared before this session's verification, to get a
  clean baseline — not a defect, just local dev-session residue.)

- The end-to-end weekly curation lifecycle is now documented (Card #62, see
  [`docs/weekly-workflow.md`](../weekly-workflow.md) and
  [`decisions.md`](decisions.md)) — documentation only, no code changed.
  Generate Pool, Review, Build Journey, and Confirm Final Playlist are
  confirmed implemented (M4/M6, Cards #37-#61). Compare is only partially
  implemented: the Recent Track Filter (hard gate) and Repetition Penalty
  (soft signal) both compare against raw official-playlist history, but no
  `Edition` entity or edition-to-edition comparison exists anywhere in the
  codebase. Plan Next, Publish, and Archive are not implemented at all —
  the only Spotify write endpoint remains `InitializeOfficialPlaylist`
  (empty-playlist creation), SQLite holds no edition-level record, and
  Card #61's `confirmedPlaylist` has no consumer yet. M7 (Weekly Editorial
  Workflow) moves from Planned to In progress on the strength of this
  definition alone — no implementation work was done.

- An `Edition` domain model and persistence now exist (Card #139, see
  [`decisions.md`](decisions.md)) — a new package,
  `backend/internal/edition`, the first genuinely new persisted domain
  entity since `candidate_selection` (Card #56). `Edition` carries a
  `Status` lifecycle (`draft`/`confirmed`/`publishing`/`published`/
  `archived`, explicit validated transitions via `Status.CanTransitionTo`
  — Publishing can return to Confirmed on a recoverable failure, never a
  dead end), an exact ordered `ConfirmedTracks` snapshot (reusing
  `candidate.CandidateMetadata`, persisted as one JSON TEXT column —
  array order is the order, no separate position column to drift out of
  sync), and publication fields (`SpotifyPlaylistID`/`URL`,
  `PublishedAt`, `LastPublishAttemptAt`/`LastPublishError`, `ArchivedAt`).
  A new `editions` SQLite table (keyed rows, same
  `CREATE TABLE IF NOT EXISTS`-in-`NewStore` convention as every other
  store) holds it; a partial `UNIQUE INDEX ... WHERE status != 'archived'`
  on a constant expression enforces at most one active (non-archived)
  Edition at the database level, including under real concurrent
  `CreateDraft` calls — no application lock, no new transaction (this
  backend still has neither, repo-wide). `POST /api/editions/confirm`
  (`edition.Service.ConfirmFromReview`/`ConfirmHandler`) is the one new
  endpoint: it lazily creates a Draft if none is active, confirms it with
  the curator's exact Kept tracks + order, allows re-confirming while
  still Draft/Confirmed (Card #61's Edit → reorder → reconfirm flow), and
  returns `ErrSnapshotLocked` once publishing has started.
  `WeeklyPlaylistPreview.vue`'s `confirmPlaylist()` is now `async` —
  it calls the new `frontend/src/services/edition.ts`'s `confirmEdition`
  and only locks the UI once that persists successfully, surfacing a
  failure instead of a false "confirmed" state; no Pinia store was
  introduced (none exists in this codebase). `Store.StartPublishing`/
  `RecordPublishSuccess`/`RecordPublishFailure`/`Archive` are real, tested
  Go methods with no HTTP route and no caller yet — prepared for, not
  implemented by, Cards #68 (Publish)/#69 (Archive). 31 new Go tests
  (`backend/internal/edition/{store,service}_test.go`, including a real
  multi-goroutine `-race`-clean test of the single-active-edition
  invariant) and 3 new Vitest cases extending
  `WeeklyPlaylistPreview.spec.ts` (19 total) — `go build ./...`/
  `go vet ./...`/`go test ./...` and `npm run build`/`npm run test` all
  pass. Manually verified against a live `SPOTIFY_MOCK_MODE=true` server:
  confirm persists and returns the Edition, an empty snapshot is rejected
  (400), re-confirming overwrites the same Edition's order, and the
  `editions` table/index exist exactly as designed (inspected directly via
  `sqlite3`). `docs/weekly-workflow.md` stages 6/7/8 and the "known gaps"
  table are updated to reflect the model now existing without claiming
  Publish/Archive/Compare are implemented.

Update this file after meaningful implementation progress. Keep it a
snapshot, not a detailed changelog — see [`decisions.md`](decisions.md) for
the reasoning behind changes.
