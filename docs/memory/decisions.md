# Decisions

Record only meaningful architectural and product decisions here, not
trivial implementation details. Each entry: Decision, Context, Reason,
Consequences.

---

**Decision:** Keep the MVP intentionally simple.

**Context:** Sound Continuum is a new, experimental project.

**Reason:** There is no guarantee the concept will gain traction. The first
goal is validating the musical curation concept, workflow, and audience
response — not building for scale that may never be needed.

**Consequences:** Every other decision below follows from this one.

---

**Decision:** Use a simple Go backend.

**Context:** MVP needs an HTTP API surface for the frontend to eventually
call.

**Reason:** Go's standard library is sufficient for a small API; no need
for a framework at this stage.

**Consequences:** `net/http` only, no framework, minimal dependencies.

---

**Decision:** Use Vue 3 + Vite + Pinia for the frontend.

**Context:** MVP needs a frontend, not yet built as of Card 15.

**Reason:** Lightweight, conventional, fast dev loop.

**Consequences:** Frontend work (later cards) should follow this stack.

---

**Decision:** Avoid microservices for the MVP.

**Reason:** No requirement yet justifies the operational overhead.

---

**Decision:** Avoid Kubernetes.

**Reason:** Nothing to orchestrate yet; adds infrastructure the MVP doesn't
need.

---

**Decision:** Avoid Redis unless a concrete requirement appears.

**Reason:** No caching or pub/sub need has been identified.

---

**Decision:** Avoid complex authentication.

**Reason:** MVP has no user accounts or protected actions yet.

---

**Decision:** Avoid unnecessary infrastructure (e.g. Dockerfile deferred as
of Card 15).

**Reason:** `go run` / `go build` is sufficient for local dev; nothing is
being deployed yet that requires a container.

**Superseded (Card 18):** Dockerfiles + Docker Compose were introduced for
local dev consistency — see the Card 18 decision below. `go run` / `go
build` remain fine outside Docker; this entry no longer reflects current
state.

---

**Decision:** Do not make the final editorial decision automatically.

**Reason:** Per the manifesto, editorial judgment remains human.

---

**Decision:** AI must not become a dependency of the editorial workflow.

**Context/Note:** This does not mean "AI can never be used." AI may assist
development or research. But the Sound Continuum MVP must remain fully
understandable and functional without an AI agent making the final
curation decision.

---

**Decision:** Prefer simple solutions until a real requirement proves
additional complexity necessary.

**Reason:** Guiding principle for all future architectural choices in this
project.

---

**Decision:** Frontend lives at `frontend/` at repo root; the Go backend is
not moved into a `backend/` directory.

**Context:** Card 16 introduced the Vue 3 + Vite frontend. The Go backend
(`cmd/`, `internal/`, `go.mod`) already lived at repo root from Card 15,
un-namespaced.

**Reason:** Wrapping the existing backend in a `backend/` directory purely
for symmetry with `frontend/` would be an unrelated restructuring of
working code, with no functional benefit, done as a side effect of an
unrelated card. `frontend/` alone is enough to make the physical
frontend/backend separation explicit.

**Consequences:** Repo root mixes backend files (`cmd/`, `internal/`,
`go.mod`, `Makefile`) with `frontend/` and `docs/`. If this becomes
confusing later, moving the backend into `backend/` is a small, reversible
follow-up — not a blocker now.

**Superseded (Card 20):** The backend was moved into `backend/` — see the
Card 20 decision below. This was the anticipated follow-up named in this
entry's own Consequences, not an unrelated refactor.

---

**Decision:** Add a Docker Compose development environment; persist SQLite
through a named volume mounted into the backend container, not a separate
database service.

**Context:** Card 18 needs frontend and backend to start together locally
with a consistent, reproducible environment, and to establish where the
future SQLite database will live.

**Reason:** SQLite is an embedded database — it runs inside the process
that uses it. A separate `sqlite` container/service would misrepresent the
architecture and add orchestration complexity (networking, startup
ordering) that an embedded database doesn't need. A named volume mounted at
`/data` in the backend container gives persistence without a database
process of its own. The backend `Dockerfile` lives at repo root (build
context `.`), matching the existing decision to keep the Go module at repo
root rather than under `backend/`.

**Consequences:** `docker compose up --build` starts both services;
`docker compose down` preserves the `sqlite_data` volume, `docker compose
down -v` removes it. No SQLite application/repository code was added — the
volume only reserves `/data` as the future database location. No reverse
proxy, healthcheck orchestration, or production deployment concerns were
introduced.

---

**Decision:** Keep frontend and backend physically separated and use a
minimal, responsibility-based directory structure; move the Go backend
into `backend/`.

**Context:** Card 20 defines the top-level project structure. Card 19 left
the backend at repo root (`cmd/`, `internal/`, `go.mod`, `Makefile`,
`Dockerfile`) while the frontend already lived under `frontend/`, an
asymmetry the Card 15/18 decision explicitly flagged as a legitimate future
follow-up.

**Reason:** Sound Continuum is an experimental MVP and should avoid
speculative architecture. Clear physical separation (`backend/` vs
`frontend/`, no shared/common/packages directory between them) makes
responsibilities understandable without introducing premature abstractions
like a shared API-contract package or code generation.

**Consequences:** Backend tooling (`cmd/`, `internal/`, `go.mod`,
`Makefile`, `Dockerfile`, `.dockerignore`, `.env.example`) now lives under
`backend/`; `docker-compose.yml`'s backend build context is `./backend`;
local backend dev commands run from `backend/` instead of repo root.
Frontend is unchanged — its existing `views/`, `services/`, `router/`,
`stores/` structure already matched this convention. New directories
(`internal/` subpackages, frontend `components/`/`types/`/`assets/`) are
introduced only when actual code requires that responsibility, not ahead of
need.

---

**Decision:** Add a root `Makefile` as a thin delegating task runner over
existing backend/frontend tooling, not a new layer of logic.

**Context:** Card 22 needs a common developer entry point covering both
`backend/` (Go, its own `Makefile`) and `frontend/` (npm scripts), without
replacing either.

**Reason:** The root `Makefile` only wraps commands that already exist and
work (`backend/Makefile`'s targets via `$(MAKE)`, `npm run dev`/`build`) —
it never duplicates build/test logic or invents tooling (no linter, no
frontend test runner, no process supervisor for a combined `dev` target)
that the project hasn't actually adopted yet.

**Consequences:** Future project-level developer tasks should follow the
same pattern: add a root Makefile target that delegates to the real
command, don't grow the Makefile into an orchestration framework. When a
linter or frontend test runner is actually adopted, add the corresponding
`backend-lint`/`frontend-lint`/`frontend-test` targets then, not before.

---

**Decision:** M3 must not be designed around Spotify Audio Features, Audio
Analysis, Recommendations, or Related Artists.

**Context:** Card 23 researched the current Spotify Web API. These four
endpoints were restricted in November 2024 to only those apps that already
had extended-quota access before the cutoff. Sound Continuum is a new app
and has no such prior access, and Spotify has offered no replacement.

**Reason:** Any M5 (musical ranking/bridges) design that assumed
tempo/energy/valence-style audio signals, or any M4 (discovery) design that
assumed Spotify-provided artist similarity, would be building on endpoints
this project cannot call. This is a hard technical constraint, not a
preference.

**Consequences:** Musical bridges (M5) and discovery (M4) must be designed
as editorial-first, optionally supported by Last.fm similarity/tag data —
not as a score computed from Spotify audio data. See
[`docs/spotify-api.md`](../spotify-api.md) for full detail.

---

**Decision:** Design Spotify integration (M3) for permanent Development
Mode limits, not Extended Quota Mode.

**Context:** Card 23 research found Extended Quota Mode now requires an
organizational Partner Application: an established business entity, an
active launched service, and 250,000+ monthly active users.

**Reason:** Sound Continuum is a small editorial MVP with no realistic path
to that threshold in the foreseeable future.

**Consequences:** M3 must work within Development Mode's constraints (5
allowlisted users, app owner needs Spotify Premium, lower rate limits) as
the permanent baseline, not a temporary bootstrapping phase.

---

**Decision:** Last.fm is a candidate complementary data source for M4
(discovery), not part of M3.

**Context:** Card 23 evaluated Last.fm's artist/track similarity and tag
endpoints as a way to compensate for Spotify's Related Artists/
Recommendations restrictions.

**Reason:** M3's scope is catalog resolution and playlist publishing, which
is Spotify-only. Last.fm's value is entirely on the discovery/similarity
side, which belongs to M4. No provider abstraction or Last.fm client exists
yet — introducing one now would be premature.

**Consequences:** M4 design should evaluate Last.fm's `artist.getSimilar`,
`track.getSimilar`, and `tag.*` methods (all usable with an API key, no
user auth) as a supporting discovery signal. M3 implementation should not
add any Last.fm code or dependency.

---

**Decision:** Centralize local development configuration under `dev/`
(`dev/docker-compose.yml`, `dev/.env`, `dev/.secrets.env`), replacing the
root `docker-compose.yml` and `backend/.env.example`.

**Context:** Card 24 needed to define local config boundaries before
Spotify credentials exist, so the restructuring happens while nothing
secret is at stake.

**Reason:** One canonical location for local dev config makes it obvious
where new secrets (Spotify, later Last.fm) belong, and avoids repeating
the root-`docker-compose.yml`-plus-scattered-`.env.example`-files pattern
as more services and credentials are added.

**Consequences:** `docker-compose.yml` moved to `dev/docker-compose.yml`
with build contexts updated to `../backend`/`../frontend`;
`backend/.env.example` removed (superseded — the backend never autoloaded
it anyway); `frontend/.env.example`/`frontend/.env` unchanged (Vite's own
convention, orthogonal to this move); root `Makefile`'s docker targets and
host-dev targets updated to use `dev/`; `.gitignore` covers both env
files.

---

**Decision:** Split local environment configuration into `dev/.env`
(non-secret) and `dev/.secrets.env` (secrets), loaded into Docker
containers via per-service `env_file:` lists rather than the Compose
`--env-file` CLI flag.

**Reason:** `env_file:` accepts a list per service natively, so the
backend service can load both files while the frontend service loads
only `dev/.env` — the frontend container structurally cannot see
`SPOTIFY_CLIENT_SECRET`. The CLI `--env-file` flag only takes one file
and is meant for variable substitution inside the Compose YAML, not
container env injection — the wrong tool here.

**Consequences:** Any future secret follows the same split; any value
genuinely needed by the frontend container goes in `dev/.env`, never
`dev/.secrets.env`.

---

**Decision:** Spotify integration (M3) is entirely backend-owned: the Go
backend holds Spotify OAuth, client credentials, tokens, and all Spotify
API communication. The frontend only calls Sound Continuum's own
`/api/spotify/*` endpoints and never receives Spotify tokens, credentials,
or authorization codes.

**Context:** Card 24 defined the Spotify integration architecture ahead
of implementation (Cards 25+). See
[`docs/spotify-integration.md`](../spotify-integration.md).

**Reason:** Keeps the Spotify client secret and tokens off the browser
entirely — the only place they can leak from is the backend, which is
also the only place that needs them.

**Consequences:** OAuth terminates at the Go backend
(`GET /api/spotify/callback`); the frontend's Spotify-related state is
limited to `connected`/`disconnected`/`authorization_required`.

---

**Decision:** Use the Authorization Code OAuth flow (without PKCE) for
Spotify curator authentication.

**Context:** Card 24 evaluated Spotify's OAuth flows
([`docs/spotify-api.md`](../spotify-api.md#21-authentication)) against
Sound Continuum's architecture: one human curator, a Go backend that can
hold a secret, a browser frontend that never talks to Spotify directly.

**Reason:** PKCE exists to protect public clients that can't hold a
client secret. The Go backend is a confidential client — it holds the
secret and terminates the entire OAuth flow, including the token exchange
— so PKCE would add complexity with no corresponding security benefit
here.

**Consequences:** The backend implements Authorization Code exchange
directly; no PKCE code verifier/challenge handling is needed.

---

**Decision:** Store the Spotify refresh token in SQLite, not
`dev/.secrets.env` and not process memory.

**Context:** Card 24 evaluated where to persist runtime Spotify token
state, given SQLite infrastructure is already provisioned (a named
volume reserved since Card 18) but no application code uses it yet.

**Reason:** A refresh token is runtime application state that changes
over time, not static local configuration — env files are the wrong tool
for it. Process memory would force re-authorization on every backend
restart, which is a poor fit for a low-frequency, human-driven weekly
curation workflow. SQLite requires no new infrastructure and persists
across restarts, fitting the single-curator, single-connection MVP scope.

**Consequences:** No token persistence code is added by Card 24 — this
decision is what a later implementation card builds against.

---

**Decision:** Use `modernc.org/sqlite` (pure Go, no CGO) as the backend's
SQLite driver.

**Context:** Card 25 implements the SQLite-backed token storage Card 24
decided on. The Go standard library has no SQLite driver, so this is the
project's first backend dependency.

**Reason:** `mattn/go-sqlite3` is the more established alternative but
requires CGO and a C toolchain; the backend's Alpine Docker image
(`golang:1.25-alpine`) has no gcc, and adding one would be new build
infrastructure with no benefit over a pure-Go driver for a single-table,
low-throughput MVP store.

**Consequences:** `backend/go.mod` now has real dependencies
(`modernc.org/sqlite` and its transitive pure-Go deps); `backend/go.sum` is
committed for the first time. `backend/Dockerfile` copies `go.sum` and was
bumped to `golang:1.25-alpine` (the driver requires Go 1.25+).

---

**Decision:** Add `user-read-private playlist-read-private` to the OAuth
scope Card 25's `AuthURL` requests, rather than introducing a separate
Client Credentials flow for playlist reads.

**Context:** Card 26 needs `GET /me/playlists` and
`GET /playlists/{id}/items`, both of which require
`playlist-read-private` on the curator's own token
([`docs/spotify-api.md`](../spotify-api.md#27-playlists)). Card 25
requested no scope at all, since `GET /v1/me` needs none.

**Reason:** Client Credentials (app-only, no user context) can't read a
specific user's private playlists at all — it was never a candidate for
this data. The only real choice was adding scope to the existing
Authorization Code token vs. some other mechanism, and there is no other
mechanism: one curator, one connection, one token. Adding scope is the
smallest change that unblocks the actual requirement.

**Consequences:** The curator has to reconnect once (existing reconnect
flow via `GET /api/spotify/auth`, no new mechanism) before playlist reads
work under the new scope. `Client.AuthURL` gained a third `scope`
parameter.

---

**Decision:** Represent Spotify Web API failures as a small typed error
taxonomy (`APIError` + sentinels), not a generic wrapped-string error.

**Context:** Card 26 needs to distinguish auth failure, forbidden,
not-found, rate-limited (with `Retry-After`), and generic failure/
transport/decode errors, and needs HTTP handlers to map each to a
sensible status code without leaking Spotify's raw response.

**Reason:** A plain `fmt.Errorf` (Card 25's existing pattern for one
endpoint) can't carry a status code or `Retry-After` duration for callers
to inspect programmatically. A full custom error-code enum or a
per-endpoint error type would be more machinery than seven fixed failure
categories need. One struct (`APIError`) wrapping one sentinel per
category, checked via `errors.Is`/`errors.As`, is the minimum that
satisfies both "distinguish failure kinds" and "carry status code +
Retry-After."

**Consequences:** `Client.Me` was refactored onto a shared `Client.request`
helper so this error handling isn't duplicated per endpoint; its exported
signature is unchanged. `ErrInvalidGrant` (Card 25, token-endpoint
specific) is left as-is, not folded into `APIError` — it's a different
concern (refresh-token validity, not a Web API response).

---

**Decision:** Use one generic `Paging[T]` type for Spotify's pagination
envelope, instead of a `PlaylistsPage`/`PlaylistItemsPage` pair.

**Context:** `GET /me/playlists` and `GET /playlists/{id}/items` (and, if
used, each object type inside `GET /search`) all return the same
`items`/`total`/`limit`/`offset`/`next`/`previous` envelope shape around a
different item type.

**Reason:** Go's generics (available since the project's Go 1.25 baseline)
make one parameterized type strictly simpler than hand-writing a
near-identical struct per endpoint — same fields, same JSON tags, zero
behavioral difference, only the item type varies.

**Consequences:** `Client.Playlists` returns `Paging[Playlist]`,
`Client.PlaylistItems` returns `Paging[PlaylistItem]`, `SearchResult`'s
fields are `*Paging[Track]`/`*Paging[Artist]`/`*Paging[Playlist]`.

---

**Decision:** Represent a playlist item's payload as a discriminated union
(`ItemType` string + exactly one of `Track`/`Episode` set, `Episode` a new
type) via a custom `UnmarshalJSON`/`MarshalJSON` on `PlaylistItem`, instead
of always decoding into `Track`.

**Context:** Card 27 requires inspecting a playlist item's `type` before
treating it as a track — Spotify documents `track` and `episode` as
current playlist item types, and an item can be `null` (removed/
unavailable content). Card 26's `PlaylistItem.Track Track` silently
decoded every item as a track regardless of type, and would zero-value a
null item rather than reporting it as unavailable.

**Reason:** A generic `interface{}`/`any` payload would push the type
switch onto every caller. A shared `Item` interface with `Track`/`Episode`
implementations is more machinery than two concrete pointer fields need.
Two nilable fields plus one type tag, decoded once via a custom
`UnmarshalJSON`, is the minimum that lets callers safely ignore whichever
type they don't care about (`if item.Track != nil`) without a type switch
or risking a false-positive zero-value `Track`.

**Consequences:** `PlaylistItem` also gained `AddedBy`/`IsLocal`
(previously absent). A matching `MarshalJSON` re-serializes the item as
`{"type", "track"?, "episode"?, ...}` for the dev-facing JSON response,
rather than dropping the payload behind unexported/`json:"-"` fields.
Confirmed live: Spotify can also return `height`/`width` as `null` on a
playlist's `images` (decodes to Go's zero value `0`, no error) — noted
here in case a future card needs to distinguish "unknown dimensions" from
"zero-size image".

---

**Decision:** Do not add Sound Continuum playlist-discovery logic
(finding "the" Sound Continuum playlist by name/ID) in Card 27.

**Context:** Card 27's spec describes a future
`Spotify account → current user's playlists → find Sound Continuum
playlist → inspect → retrieve items` workflow, but explicitly forbids
hardcoding a playlist ID or name and forbids fuzzy matching. No playlist
name or ID configuration exists anywhere in this repo (`dev/.env`,
`dev/.secrets.env`, or elsewhere) as of Card 27.

**Reason:** There is no existing, clean place for this decision to live —
introducing one now would mean inventing both the config mechanism and the
actual name/ID value speculatively, ahead of any card that has decided
what they should be. That's exactly the kind of premature
config/abstraction this project's MVP discipline avoids.

**Consequences:** `GET /api/spotify/playlists` (list), `GET
/api/spotify/playlists/{id}` (this card), and `GET
/api/spotify/playlists/{id}/items` are the complete read-only playlist
surface as of Card 27. Playlist discovery is deferred to a later
application/service layer, once a concrete decision exists about how the
Sound Continuum playlist is identified (config value, naming convention,
or otherwise).

---

**Decision:** Extend the existing `Track`/`Artist` types and add one new
`Album` type for Card 28's `GetTrack`, rather than a parallel "detailed
track" type.

**Context:** Card 28 needs full track metadata (album, explicit flag,
track/disc numbers, external IDs) to support future editorial review of a
track before it's added to a playlist. `Track` already existed (Card 26),
reused by `PlaylistItem.Track` and `SearchResult.Tracks`.

**Reason:** Spotify's fuller track field set is additive and backward-
compatible with every existing decode path — a second "detailed track"
type would duplicate `Track` for no behavioral difference, only more
fields. `Artist` (already shared by `Track.Artists`) also gains fields
(`href`, `external_urls`) needed by the new `Album.Artists`, rather than a
second artist type.

**Consequences:** `PlaylistItem.Track` and `SearchResult.Tracks` pick up
every new field automatically. `Artist`'s decoded shape changes (a
compatible superset) everywhere it's already used. Audio features,
`popularity`, `available_markets`, and `linked_from` remain explicitly
excluded, consistent with the existing M3 audio-features decision above.

---

**Decision:** `Client.Track`/`Service.Track` accept no `market` parameter.

**Context:** Spotify's `GET /tracks/{id}` supports an optional `market`
query parameter. Card 28 asked for a deliberate decision on this rather
than defaulting silently.

**Reason:** `Service.withToken` always supplies a user (Authorization
Code) access token — this client has no Client Credentials path. Spotify
infers market from the authenticated user's account when `market` is
omitted; it's only required for app-only tokens. Threading an unused
parameter through `Client`/`Service`/handler now would be the same kind of
premature configuration this project already avoided once (the Card 27
playlist-discovery deferral above).

**Consequences:** No global market configuration exists. If a concrete
market-mismatch symptom appears later, add the parameter then, not ahead
of need.

---

**Decision:** Reject an empty track ID client-side (`ErrEmptyTrackID`),
mirroring `Client.Search`'s `ErrSearchLimitTooHigh` pattern.

**Context:** `GET /tracks/{id}` with an empty ID would build a malformed
`/v1/tracks/` path — Spotify's bulk tracks endpoint, already established
as unavailable/out of scope for this project.

**Reason:** Reuse the exact validate-before-request pattern
`Client.Search` already established, rather than inventing a different
validation mechanism for one new endpoint: one sentinel, checked before
any HTTP call, special-cased in the handler to `400` before falling
through to `writeSpotifyError`.

**Consequences:** `ErrEmptyTrackID` added to `errors.go`; `TrackHandler`
special-cases it exactly as `SearchHandler` special-cases
`ErrSearchLimitTooHigh`. `writeSpotifyError` itself is unchanged.
