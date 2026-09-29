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

Update this file after meaningful implementation progress. Keep it a
snapshot, not a detailed changelog — see [`decisions.md`](decisions.md) for
the reasoning behind changes.
