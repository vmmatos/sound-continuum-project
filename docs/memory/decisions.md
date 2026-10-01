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

---

**Decision:** Extend the existing `Artist` type for Card 29's `GetArtist`
(`type`, `images`, `genres`), rather than a parallel "detailed artist"
type; do not add `followers`/`popularity`; do not implement a bulk artist
endpoint or `/artists/{id}/top-tracks`.

**Context:** Card 29 needs the full single-artist response
(`GET /artists/{id}`) for future editorial workflows inspecting an
artist behind a track or playlist. `Artist` already existed (Card 26) as
a simplified shape shared by `Track.Artists`, `Album.Artists`, and
`SearchResult.Artists` (Card 28 already extended it once with
`href`/`external_urls`).

**Reason:** Same reasoning as Card 28's `Track`/`Album` decision above —
the fuller artist field set is additive and backward-compatible with
every existing decode path, so a second type would only duplicate
`Artist` for no behavioral difference. `followers` and `popularity` were
removed by Spotify from the Artist object for Development Mode (per the
Card 23 API-changes research); inventing or estimating replacement
values would misrepresent Spotify data as something it isn't. The bulk
artist endpoint (`GET /artists?ids=`) and `/artists/{id}/top-tracks` are
both removed for Development Mode — same status as the bulk-tracks
removal Card 28 already worked around — so this card makes one request
per artist and does not wrap the removed top-tracks endpoint.

**Consequences:** `Track.Artists`, `Album.Artists`, and
`SearchResult.Artists` pick up `type`/`images`/`genres` automatically,
since all three already reuse `Artist`. `genres` is decoded as optional,
deprecated Spotify metadata only — no genre normalization or mapping
into Sound Continuum's musical DNA is built on top of it. A future
workflow needing several artists' data must batch explicitly at the
application layer (with awareness of rate limits and partial failures),
not inside this client.

---

**Decision:** One official Sound Continuum Spotify playlist, created once
via `POST /me/playlists`, identified by a locally-stored Spotify playlist
ID — not discovered by name/search.

**Context:** Card 27 deliberately deferred "how do we identify the Sound
Continuum playlist," explicitly forbidding hardcoded IDs and fuzzy
name-matching, until a concrete decision existed. Card 30 needed to
establish the official playlist as persistent infrastructure that every
future weekly edition reuses, per the manifesto's "each week is a
chapter" principle — not a new playlist per edition.

**Reason:** A locally-persisted ID (in a new `official_playlist` SQLite
table, `backend/internal/spotify/store.go`) is the only mechanism that
can guarantee "one project → one playlist" without ambiguity — name
search is inherently fuzzy and could match a renamed or unrelated
playlist. Creating it via `POST /me/playlists` (not the deprecated
`/users/{user_id}/playlists`) follows the current Spotify Web API.

**Consequences:** `Service.InitializeOfficialPlaylist` checks the local
table first and returns the cached record with **no Spotify call at all**
if found — this is also what guarantees idempotency (no duplicate
playlist possible) and what makes a local-record-but-Spotify-can't-
confirm-it scenario a non-issue: an existing local record is never
re-verified against Spotify. If Spotify creation succeeds but the local
save fails, the app does not retry (retrying risks a duplicate) — the
Spotify playlist ID is logged for manual reconciliation. There is no
distributed transaction between the Spotify API call and the SQLite
write, and none is planned; this is an accepted, documented limitation,
not solved by this card.

---

**Decision:** Add `playlist-modify-public` to the OAuth scope Card 25/26
established.

**Context:** Card 30 needs `POST /me/playlists` to create the official
playlist as public.

**Reason:** Same reasoning as Card 26's scope addition — one curator, one
connection, one token; adding scope to the existing Authorization Code
flow is the smallest change that unblocks the requirement. No
`playlist-modify-private` is requested — the official playlist is always
public, and requesting unused scope would be unnecessary.

**Consequences:** The curator must reconnect once (existing
`authorization_required` → `Reconnect Spotify` flow, no new mechanism)
before playlist creation works under the new scope.

---

**Decision:** Introduce `backend/internal/candidate` as a new, flat,
feature-named package (sibling to `spotify`/`health`) for the
`CandidateTrack` domain model, rather than an `internal/domain/...`
layer or a subpackage of `spotify`.

**Context:** Card 31 begins M4 by defining the first Sound Continuum
domain concept that must be structurally independent of Spotify's
response types — a candidate is editorial metadata about a discovered
track, not a copy of `spotify.Track`. This is also the first domain model
in the repo; there was no existing convention to extend.

**Reason:** The existing backend has exactly two packages, both flat and
feature-named (`internal/health`, `internal/spotify`) — no
`internal/domain` layer exists anywhere. Introducing one now, for a single
type, would be new architecture the card doesn't need; a new
feature-named package matches the pattern already established.
`CandidateTrack` cannot live inside `internal/spotify` without
contradicting the card's explicit requirement that the domain model not
depend on Spotify-specific structs.

**Consequences:** `backend/internal/candidate/candidate.go` defines
`Source`, `Category`, `Status` (each `type X string` + a `const` block +
a `Valid() bool` method — the first enum-with-validation convention in
this codebase, since none existed to reuse), `ID` (the candidate's
internal identity, distinct from the external `SpotifyTrackID` field),
and `CandidateTrack` itself. `errors.go` follows `spotify/errors.go`'s
existing sentinel-error pattern (`var ErrX = errors.New("candidate:
...")`). Enum string values match the card's own editorial wording
exactly (`"New Release"`, `"under review"`, etc.) rather than a
normalized wire format, since there is no JSON/API boundary for this type
yet — a future card introducing one can decide serialization then. No ID
generation was added (no UUID or similar dependency); a `candidate.ID` is
supplied by the caller, deferred the same way Card 27 deferred Sound
Continuum playlist discovery — until a concrete persistence/discovery
card creates a real need. No persistence, schema, API endpoint, or CRUD
was added — nothing in the card's Definition of Done requires it.

---

**Decision:** Add `Type` as a second, fully independent enum on
`CandidateTrack`, rather than extending `Category` or deriving `Type`
from `Category`.

**Context:** Card 32 introduces a `CandidateType` concept (`Classic`,
`Current`, `Discovery`) distinct from the editorial `Category` (`Past`,
`Present`, `Emerging`, `New Release`) Card 31 defined. `Category`
describes where a candidate sits editorially; `Type` describes how it
entered the editorial process — e.g. Discovery does not imply Emerging,
and Classic does not imply Past. The card explicitly forbids any
compatibility or inference rule between the two.

**Reason:** `Category` and `Status` already established a convention
(`type X string` + `const` block + `Valid() bool`, field self-named
after its type) for exactly this kind of small, closed domain enum.
Reusing it for `Type` needed no new abstraction — no generic enum
framework, no shared validation helper — since three independent copies
of a five-line pattern is cheaper than building one shared mechanism for
a shape this small. `Type` is required and validated at construction
like `Category` (not defaulted like `Status`), since Card 32 explicitly
rejects a meaningless default such as always defaulting to `Classic`.

**Consequences:** `backend/internal/candidate/candidate.go` gains `Type`
(`TypeClassic`, `TypeCurrent`, `TypeDiscovery`) and a `Type Type` field
on both `CandidateTrack` and `NewCandidateTrackParams`. `errors.go` gains
`ErrInvalidType`. `Validate()` checks `Type.Valid()` independently of
`Category.Valid()` — no cross-field rule was added, so combinations like
`Type: Discovery, Category: Past` or `Type: Classic, Category: New
Release` are valid and unremarkable. No persistence, API, or discovery
logic was added — nothing in the card's Definition of Done requires it.

---

**Decision:** Introduce `backend/internal/discovery` as a new, flat,
feature-named package (sibling to `candidate`/`spotify`) for classic music
discovery, rather than a subpackage of `candidate` or `spotify`, or a
generic multi-source discovery framework.

**Context:** Card #33 builds M4's first discovery workflow: turning Sound
Continuum's 15 "Past" reference artists into `Classic`/`Past`/`discovered`
candidates via Spotify's artist-albums/album-tracks endpoints. The card
explicitly forbids building separate services per candidate type
(Classic/Current/Emerging) or a generic provider/plugin architecture
ahead of need.

**Reason:** Same reasoning as Cards 31/32's `candidate` package decision —
the repo's only convention is flat, feature-named packages; no
`internal/domain` or `internal/discovery/classic` layering exists to
extend. `discovery` depends on both `spotify` (catalogue access) and
`candidate` (the type it produces), so it cannot live inside either
without an import cycle or misplaced ownership.

**Consequences:** `backend/internal/discovery/reference_artists.go` holds
`PastReferenceArtists`, the first canonical definition of this list
anywhere in the repo (previously undocumented in any runtime form) —
plain curator-editable data, not database-driven. `discovery.go` holds
`Config`, `Service`, `DiscoverClassic`, and `ClassicHandler`. A future
Current/Emerging/Last.fm discovery card should follow the same
"reference data as a package-level var, one focused use case per card"
pattern rather than generalizing this package prematurely — see the
card's explicit "do not create a generic discovery framework" constraint.

---

**Decision:** Give `discovery.Service` a small unexported interface seam
(`spotifyCatalogue`: `Search`/`ArtistAlbums`/`AlbumTracks`) over
`*spotify.Service`, instead of testing against a real `spotify.Service`
pointed at an `httptest.Server` (the pattern every `spotify` package test
uses).

**Context:** Every existing Spotify HTTP-mocking test lives inside
package `spotify` itself and redirects `Service`'s internal `*Client` by
setting its `AuthBaseURL`/`APIBaseURL` fields directly — these are
exported on `Client`, but `Service.client` itself is an unexported field.
`discovery` is a separate package and has no access to it, and `spotify`
exposes no constructor or setter that would let an external package point
a `*spotify.Service` at a fake server.

**Reason:** The alternatives were worse: adding a test-only exported hook
to `spotify.Service` (new production surface serving only one external
package's tests) or seeding a connection through the real OAuth handler
flow (`AuthHandler`/`CallbackHandler`) — which still cannot redirect the
token exchange or catalogue calls away from the real Spotify hosts,
since that redirection is the same unexported-field problem one level
up. A three-method interface, satisfied structurally by `*spotify.Service`
with zero changes to the `spotify` package, is the smallest fix — matches
"introduce only the minimum new code required."

**Consequences:** `discovery.NewService` still takes a concrete
`*spotify.Service` (production callers, `main.go` included, are
unaffected and unaware of the interface). `discovery_test.go` implements
a `fakeCatalogue` satisfying the same interface entirely in memory — no
HTTP server, no SQLite database needed for discovery's own tests. This is
a one-off seam for this package's specific testing gap, not a general
provider-abstraction precedent; it should not be read as license to wrap
every `spotify.Service` consumer in an interface.

---

**Decision:** Reuse the Spotify track ID directly as `candidate.ID`
(`candidate.ID(track.ID)`), rather than generating a separate identifier.

**Context:** `candidate.ID` has been caller-supplied since Card #31, with
no generation mechanism anywhere in the repo — decisions.md flagged this
as deferred "until a concrete persistence/discovery card creates a real
need."

**Reason:** `CandidateTrack.Validate()` has no rule requiring `ID` and
`SpotifyTrackID` to differ. The Spotify track ID is already a stable,
externally unique string fetched as part of discovery — generating a
second identifier (e.g. via `google/uuid`, already an indirect
dependency but not a direct one) would add a dependency and a step for no
behavioral benefit within one discovery run's scope.

**Consequences:** For Source=Spotify candidates, `ID == SpotifyTrackID`
always holds. A future persistence layer or a non-Spotify source (Last.fm,
Manual) will need its own ID strategy — this decision only covers Card
#33's Spotify-sourced candidates, and is not a general ID-generation
policy.

---

**Decision:** `Client.ArtistAlbums` always sends a fixed
`include_groups=album,single`, with no caller-supplied override.

**Context:** Card #33 needs an artist's own catalogue (for Classic
discovery), explicitly warning against building an album-classification
engine or inferring musical quality from metadata.

**Reason:** Spotify's `include_groups` (`album,single,compilation,
appears_on`) defaults to all four when omitted. `compilation`/
`appears_on` results are compilations/reissues and guest-appearance
credits — noise that would waste a bounded per-artist album budget rather
than reflect the artist's own historical catalogue. This is one fixed
query parameter value with exactly one caller today, not a
classification system: it excludes two Spotify-defined groups, it does
not rank, score, or interpret album metadata.

**Consequences:** If a future card needs compilations or guest
appearances, `include_groups` should become a parameter then, not ahead
of need — matching the Card #27 (playlist discovery) and Card #28
(`market` parameter) precedent of not threading unused configuration
speculatively.

---

**Decision:** Remove Pinia (`frontend/package.json` dependency,
`createPinia()` bootstrap in `main.ts`, the empty `frontend/src/stores/`
placeholder) until a real store is needed.

**Context:** Card 17 configured Pinia ahead of any actual shared frontend
state, on the expectation it would be needed soon. As of this decision, no
`defineStore` call exists anywhere in the repo — every card since (25/26/
30) that needed frontend state used page-local `ref`s instead, explicitly
noting "no new Pinia store" each time.

**Reason:** A wired-but-unused state-management dependency is exactly the
kind of ahead-of-need complexity this project's MVP discipline otherwise
avoids everywhere else (see the Card 27/28/33 precedents of not adding
config/parameters ahead of a concrete caller). Carrying it costs nothing
today, but it's dead weight in the dependency tree and the `main.ts`
bootstrap, with no code exercising it.

**Consequences:** `frontend/src/main.ts` no longer calls `.use(createPinia())`;
`package.json`/`package-lock.json` no longer list `pinia`;
`frontend/src/stores/` (previously just a `.gitkeep`) is removed. Card 17's
"Configure Pinia" decision is superseded by this one. If a future card
introduces real shared frontend state, reinstate the dependency and the
`stores/` convention then, not before.

---

**Decision:** Remove Vue Router (`frontend/package.json` dependency,
`frontend/src/router/index.ts`, the `.use(router)` bootstrap in `main.ts`)
until a second route is needed; `App.vue` renders `HomeView` directly.

**Context:** Card 16 installed Vue Router for a single `/` route. As of
this cleanup, no second route, no `RouterLink`, and no navigation logic of
any kind exists anywhere in the frontend — every card since has added
content to the same `HomeView.vue`.

**Reason:** A router wired for exactly one route carries no behavior a
plain component render doesn't already provide — the same
wired-but-unearning-its-keep pattern as the Pinia removal above. Rendering
`<HomeView />` directly from `App.vue` is strictly simpler and removes a
dependency with zero routing decisions to make yet.

**Consequences:** `frontend/src/router/` no longer exists.
`frontend/src/App.vue` imports and renders `HomeView` directly instead of
`<RouterView />`. `frontend/src/main.ts` no longer imports or installs a
router. If a future card introduces a second page, reinstate Vue Router
then, not before.

---

**Decision:** Remove `candidate.CategoryNewRelease`.

**Context:** Card 31 introduced it "kept ahead of use deliberately." As of
this cleanup — Cards 32-36 (Classic, Current, Emerging discovery, and the
Candidate Pool orchestrating them) — no code has ever constructed a
candidate with this category; the only references were its own
declaration and two test fixtures.

**Reason:** Card 31's own reasoning for trimming `candidate.go` down to
what workflows actually use (dropping unused `Source`/`Status` values and
editorial-note fields — see that entry above) applies equally here: an
enum value with zero call sites across five subsequent discovery cards is
no longer "ahead of use," it's unused. Superseding that one clause of the
Card 31 decision, not the rest of it.

**Consequences:** `Category` now has three values (`Past`, `Present`,
`Emerging`). The two test cases that referenced `CategoryNewRelease`
(`candidate_test.go`) were repointed at `CategoryPresent` — they were
exercising Spotify-track-ID/Type-independence behavior, not this specific
category. If a future editorial workflow needs a "new release" distinction,
add the constant back then, with a real caller.

---

**Decision:** Add the Recent Track Filter (Card #37) to the existing
`backend/internal/discovery` package (`recent_track_filter.go`), not a new
package.

**Context:** Card #37 needs to split `CandidatePool` candidates into
eligible/recently-used by comparing them against the official Spotify
playlist's `added_at` history. It depends on both `spotify` (playlist
items) and `candidate` (the type it filters).

**Reason:** Same reasoning already recorded for `pool.go` — `discovery` is
the one package that legitimately depends on both `spotify` and
`candidate`, and the filter sits directly after `DiscoverPool` in the same
pipeline. A new package would only duplicate that dependency edge.

**Consequences:** `discovery.Service.FilterRecentTracks` is a new public
method alongside `DiscoverPool`; `CandidatePool` gained one field,
`RecentTrackFilter RecentTrackFilterResult`.

---

**Decision:** Give `discovery.Service` a single unexported clock seam
(`now func() time.Time`, defaulted to `time.Now` in `NewService`,
overridden directly by same-package tests) for `FilterRecentTracks` only.

**Context:** Card #37 explicitly requires that recency-boundary behavior
be tested deterministically, without depending on the real system clock.
Every other `time.Now()` call in this codebase (recency filtering in
`DiscoverCurrent`/`DiscoverEmerging`, token expiry in `spotify`, SQLite
timestamps) is direct, uninjected, and existing tests tolerate real
wall-clock time via relative fixtures (`daysAgo`) — there was no
clock-injection precedent anywhere in the repo before this card.

**Reason:** The card's own requirement overrides the repo's existing
convention for this one feature; a repo-wide clock abstraction would be
solving a problem no other feature has. One unexported field, no clock
interface/package/dependency, is the smallest change that makes an exact
`>=`-boundary test possible.

**Consequences:** This is a one-off exception, not a new repo-wide
pattern — a future card needing the same guarantee elsewhere should add
its own local `now` field the same way, not generalize this one.
`FilterRecentTracks` calls `s.now()` directly with no nil-guard: every
path that constructs a `Service` reaching `FilterRecentTracks`
(`NewService` in production, and every test helper that exercises it)
sets `now`, so a defensive fallback would guard a case that cannot occur.

---

**Decision:** `FilterRecentTracks` walks the entire official playlist with
a dedicated pagination loop, not Card #33's `walkPages[T any]`.

**Context:** `walkPages` bounds a crawl at a caller-supplied `maxItems` —
correct for discovery's bounded catalogue crawls, but Card #37 explicitly
requires inspecting every playlist item, however many there are, since a
missed page could hide a recently-used track and defeat the guardrail.

**Reason:** Reusing `walkPages` would mean picking an arbitrary
`maxItems` cap, which is exactly the "assume the first N items are
enough" behavior the card forbids. A small loop that pages by
`offset`/`Paging.Total` until exhausted is simpler than forcing an
unbounded call through a bounded-by-design helper.

**Consequences:** `recentTrackIndex` (private to `recent_track_filter.go`)
requests 50 items per page (Spotify's current documented max for this
endpoint) and stops when a page returns 0 items or `offset >=
Paging.Total`. `walkPages` itself is unchanged and still used by
Classic/Current/Emerging discovery.

---

**Decision:** Read `RECENT_TRACK_LOOKBACK_DAYS` directly via `os.Getenv`/
`strconv.Atoi` in `cmd/server/main.go` (falling back to
`discovery.DefaultRecentTrackLookbackDays`, 28, on empty/invalid input)
and pass it into `discovery.NewService` as a plain `int`, rather than
following `discovery.Config`/`CurrentConfig`/`EmergingConfig`'s existing
"constructor-injected, no env var" convention — and without wrapping it
in its own config struct, since it's a single field with a single caller.

**Context:** Card #37 explicitly names `RECENT_TRACK_LOOKBACK_DAYS` as a
configuration value and requires it be "configurable through the existing
project configuration mechanism." This repo actually has two: every other
`discovery` bound is a constructor-injected `Config` struct with no env
var (documented reasoning: "no env-var-driven-limit convention, and tests
need small numbers"); every top-level runtime setting (`PORT`,
`SQLITE_PATH`, `SPOTIFY_CLIENT_ID`, `LASTFM_API_KEY`) is a plain
`os.Getenv` read in `main.go`.

**Reason:** The lookback window is a curator-tunable editorial policy
value (like `PORT` is an environment-tunable runtime value), not an
internal crawl bound sized for test convenience (like
`MaxAlbumsPerArtist`). Reading it once in `main.go` and passing it through
the constructor satisfies both conventions at once: the config struct
stays constructor-injected (so tests keep passing small, explicit values
with no env var involved), and the one genuinely environment-tunable
value follows the same `os.Getenv` pattern as every other env var in this
codebase.

**Consequences:** `discovery.Service.recentTrackLookbackDays` is a plain
unexported `int`, set once by `NewService`. An empty or unparseable
`RECENT_TRACK_LOOKBACK_DAYS` falls back to the default, silently logged as
a warning — matching this codebase's existing no-hard-validation style for
`PORT`/`SQLITE_PATH`, not a new fail-fast startup check.

---

**Decision:** Add `spotify.Service.OfficialPlaylist(ctx) (*OfficialPlaylist,
error)`, a thin read-only wrapper over the existing
`Store.GetOfficialPlaylist`, returning the new sentinel
`ErrOfficialPlaylistNotConfigured` when no playlist has been persisted yet.

**Context:** Card #37 needs the official playlist's Spotify ID without
ever creating one. The only existing access paths were
`Store.GetOfficialPlaylist` (unexported `store` field, inaccessible
outside package `spotify`) and `Service.InitializeOfficialPlaylist`
(create-or-return, which would give a read-only filter operation the
ability to accidentally create the playlist and makes "not configured yet"
indistinguishable from "just created").

**Reason:** A read-only accessor that never calls Spotify and never
writes is the smallest new surface that unblocks `FilterRecentTracks`
without overloading `InitializeOfficialPlaylist`'s different (creation)
contract. Returning a sentinel error rather than `(nil, nil)` for "not
found" matches every other Service method's error-based contract in this
package and lets `PoolHandler` map it to a clear "not initialized" error
distinct from a Spotify transport/API failure.

**Consequences:** `discovery.spotifyCatalogue` interface gained
`OfficialPlaylist(ctx) (*spotify.OfficialPlaylist, error)` alongside a new
`PlaylistItems` method (both already satisfied structurally by
`*spotify.Service`, extending the existing one-off test seam rather than
creating a new abstraction). No scope change: the official playlist is
always read through the curator's authenticated token, and
`playlist-read-private` (Card #26) already covers it regardless of the
playlist's public/private flag.

---

**Decision:** A recently-used candidate keeps `Status: discovered` — no
new `candidate.Status` value, no candidate field mutation of any kind.

**Context:** Card #37 explicitly requires that filtering never becomes an
editorial rejection and that every other `CandidateTrack` field
(`Type`/`Category`/`Status`/`Source`/title/artist/Spotify ID) stays
untouched.

**Reason:** Eligible vs. Recently Used is a property of one filtering
operation's output, not a durable state of the candidate itself — a
`RecentlyUsedCandidate` wrapper (`Candidate`, `LastUsedAt`, `Reason`) on
`RecentTrackFilterResult` carries that distinction instead, exactly
mirroring the `discovery.Failure`/`WorkflowError` pattern already used
elsewhere in this package for non-domain, operation-scoped facts.

**Consequences:** `backend/internal/candidate` is completely unchanged by
this card. If a future card introduces an editorial-review workflow with
real status transitions, it can still add new `Status` values then,
independent of this filter.

---

**Known limitation (found during Card #37 live verification, not fixed by
this card):** `Service.InitializeOfficialPlaylist`'s idempotency (Card #30)
only holds within a single SQLite database file. This project has two
separate SQLite files by design — `backend/sound-continuum.db` for host
`make backend-run` (Card 19) and `/data/sound-continuum.db` inside Docker
Compose (Card 18) — each with its own independent, empty `official_playlist`
table on first use. Initializing the official playlist once via each path
creates two real, separate playlists on Spotify, since Card #30's design
deliberately checks only the local row, never Spotify itself (no
name/ID search — see that decision above). Confirmed live: the curator
had done exactly this, ending up with two "Sound Continuum — Weekly
Journey" playlists on the real account, both since deleted.

**Reason not fixed now:** A real fix (e.g. a Spotify-side existence check
as a backstop) is an architectural change to Card #30's playlist-identity
design, out of scope for Card #37 ("implement ONLY Card #37" — see
CLAUDE.md). Logged here per the curator's explicit choice, for a future
card to address.

---

**Decision:** Add `candidate.CandidateMetadata` (`Title`, `Artists`,
`Album`, `DurationMS`, `Explicit`, `SpotifyURL`, `SpotifyURI`) as a new,
nested, additive field (`CandidateTrack.Metadata *CandidateMetadata`) in
package `candidate`, and do the Spotify→metadata mapping/enrichment
(`mapSpotifyTrack`, `Service.EnrichCandidateMetadata`) inside package
`discovery`, not `candidate` or `spotify`.

**Context:** Card #38 needs each eligible candidate to carry structured
Spotify metadata for future ranking/UI/curation, while keeping Sound
Continuum's domain model independent of Spotify's response shape (the same
principle `candidate.go`'s own package doc has stated since Card 31).

**Reason:** The metadata *shape* (title/artists/album/etc.) is Sound
Continuum's own editorial concept, so it belongs in `candidate`, next to
`CandidateTrack`. The *mapping* from a Spotify response into that shape is
provider-specific and belongs where the provider dependency already lives —
`discovery` is the one package with a decided, precedented reason to depend
on both `spotify` and `candidate` (see the Card #37 `recent_track_filter.go`
decision above); putting the mapping in `candidate` would pull `spotify`
into a domain package, and putting it in `spotify` would pull `candidate`
into a provider package. `Metadata` is a pointer, not an embedded value, so
nil unambiguously means "not enriched" — never confusable with "enriched
with all-empty fields."

**Consequences:** `backend/internal/candidate/metadata.go` is a new file,
same package, no new package. `backend/internal/discovery/
metadata_enrichment.go` adds `Track` to the existing `spotifyCatalogue` seam
(already satisfied by `*spotify.Service`'s Card #28 `Track` method — zero
changes to package `spotify`). `CandidatePool.MetadataEnrichment` and
`PoolHandler`'s post-`FilterRecentTracks` enrichment step follow the exact
composition pattern Card #37 established for `RecentTrackFilter`.

---

**Decision:** A per-candidate Spotify metadata lookup failure (not found,
rate limited, unauthorized, malformed response, generic API failure) keeps
that candidate in the response with `Metadata` left nil and the failure
recorded on `EnrichmentResult.Failures`; only a connection-level failure
(`ErrNotConnected`/`ErrInvalidGrant`) aborts the whole
`EnrichCandidateMetadata` call and makes `PoolHandler` return an HTTP error.

**Context:** Card #38 requires that a metadata failure "must not silently
produce an apparently valid candidate" and must not be "silently swallowed,"
without mandating that the whole endpoint fail. Up to ~150 eligible
candidates could need a Spotify `Track` lookup in one `PoolHandler` call.

**Reason:** This mirrors the exact convention Classic/Current/Emerging
discovery already established: a genuine Spotify outage (connection-level)
aborts, while an individual item's failure (Card #36 confirmed this live
with real `ArtistAlbums` 429s) is recorded on a `Failures` list and the run
continues — never treated as a whole-run abort condition. Applying a
stricter, all-or-nothing rule only to metadata enrichment, when every other
per-item Spotify failure in this codebase is tolerated, would be an
inconsistent, un-argued exception. A nil `Metadata` plus a visible
`Failures` entry satisfies "don't fake it" and "don't hide it"
simultaneously without an all-or-nothing endpoint.

**Consequences:** `PoolHandler`'s response can contain an eligible
candidate with `Metadata == nil` (visible in `MetadataEnrichment.Failures`)
alongside fully enriched candidates. A future UI/consumer must handle a nil
`Metadata` on an otherwise-eligible candidate.

---

**Decision:** Provider source (`candidate.Source`) and discovery provenance
(`candidate.Provenance []DiscoveryProvenance`, new in Card #39) are
separate concepts, and `Source` is not extended to represent discovery
mechanisms.

**Context:** Card #39 (M4's final card) needs to record *how* a candidate
entered the pipeline (which discovery method, which seed/reference artist,
which provider was queried) without conflating it with `Source`, which has
meant "which provider supplied the candidate track" since Card #31. Before
this card, `DiscoverEmerging`'s Last.fm-derived discovery context
(`Result.EmergingProvenance`) already existed but lived on the workflow's
`Result`, not the candidate — lost by the time a candidate reached the
Pool, filter, or enrichment stage.

**Reason:** `Source` answers "which provider supplied this track" (today
always `Spotify` — every workflow resolves candidate identity through
Spotify even when Last.fm drove the discovery, per the Card #35 decision).
Provenance answers a different question: "how did this candidate enter
Sound Continuum." Overloading `Source` with discovery-mechanism values
(e.g. `LastFmSimilarArtist`) would make a Spotify-identified,
Last.fm-discovered candidate's `Source` ambiguous between "the track's
provider" and "the discovery signal." Keeping them separate means an
Emerging candidate can correctly carry `Source: Spotify` and provenance
`Provider: Last.fm` at the same time, with neither value overwriting the
other.

**Consequences:** `backend/internal/candidate/provenance.go` adds
`DiscoveryMethod` (`classic_reference_artist`/`current_reference_artist`/
`lastfm_similar_artist`/`manual`), `ProvenanceProvider` (`Spotify`/
`Last.fm`/empty), `SeedArtist` (`Provider`, `ProviderArtistID`, `Name`),
and `DiscoveryProvenance` (`Method`, `Provider`, `Seed`,
`DiscoveredArtist`, `LastFMMatch`) — same package as `Source`/`Type`/
`Category`/`Status`, following their existing `type X string` + `const` +
`Valid()` convention. `CandidateTrack.Source` itself is completely
unchanged: still `SourceSpotify` only, no `Manual`/`Last.fm` value added.
A curator manually adding a real Spotify track keeps `Source: Spotify`;
only its provenance `Method` becomes `manual`, with no `Seed`/`Provider`
fabricated — this avoids reviving the `Manual`/`Last.fm` `Source` values
Card #31 deliberately trimmed for having zero call sites, since Card #39
introduces no workflow that constructs a candidate without going through
Spotify. `Result.EmergingProvenance` (Card #35) is unchanged and still
serves its own purpose (a whole discovery run's artist-level provenance,
independent of any specific candidate); `DiscoveryProvenance` is the
new, per-candidate, pipeline-surviving counterpart.

---

**Decision:** Candidate Pool deduplication merges provenance from every
workflow that found the same Spotify track, instead of discarding the
losing candidate's provenance entirely.

**Context:** `DiscoverPool`'s existing dedup step (Card #36) already
decides which candidate's classification (`Type`/`Category`/`Status`)
survives when two workflows discover the same Spotify track ID, via
`candidateTypePriority`. Before Card #39, the losing candidate's entire
struct — provenance included — was simply discarded.

**Reason:** The card's own principle: "deduplication should remove
duplicate candidates, not useful provenance." A track legitimately found
via both a Classic reference artist and a Last.fm similarity hop, say, has
two genuine discovery paths worth keeping, even though it's edited as one
candidate. Losing the runner-up's provenance would make an editorially
interesting fact (a track was independently surfaced by two different
signals) invisible.

**Consequences:** `pool.go`'s merge loop now computes the surviving
candidate's classification exactly as before, but sets its `Provenance =
candidate.MergeProvenance(existing.Provenance, c.Provenance)` — a new
package-level function in `candidate` that unions two provenance slices,
collapsing exact duplicates (same `Method` + same `Seed`/`DiscoveredArtist`
identity, preferring a stable provider ID over name when available) to
their first occurrence. `MergeProvenance` lives in package `candidate`,
not `discovery`, matching every other provenance-shape decision above.

---

**Decision:** Remove `CandidateTrack.DiscoveryReason` (Card #31's original
free-text field) rather than keep it alongside the new `Provenance` field.

**Context:** `DiscoveryReason` was always exactly one of three fixed
constant strings, one per discovery workflow (e.g. "Discovered from Past
reference artist catalogue."). Card #39 introduced `Provenance[].Method`,
a typed enum (`classic_reference_artist`/`current_reference_artist`/
`lastfm_similar_artist`/`manual`) recording the identical fact for the
identical set of workflows.

**Reason:** Once `Method` existed, `DiscoveryReason` had no information
`Method` didn't already carry, structurally and more usefully (a
consumer can switch on `Method`, not string-match a sentence). No caller
anywhere in the repo read `DiscoveryReason` for anything beyond that one
fact. Keeping both would have been two representations of the same
data with no independent reason to diverge — this project's own
"prefer simple solutions" principle applies as much to a field going
stale as to one added ahead of need.

**Consequences:** `CandidateTrack`/`NewCandidateTrackParams` no longer
have a `DiscoveryReason` field; the three `discoveryReason`/
`currentDiscoveryReason`/`emergingDiscoveryReason` constants in
`discovery.go` are gone. A future UI wanting a human-readable discovery
explanation should render one from `Provenance[].Method` (and `Seed`/
`DiscoveredArtist`/`LastFMMatch`) rather than reintroducing a parallel
free-text field.

---

**Decision:** Introduce `backend/internal/scoring` as a new, flat,
feature-named package (sibling to `candidate`/`discovery`/`spotify`/
`lastfm`) for the Candidate Scoring Model (Card #40, M5's first card),
rather than adding scoring types to package `candidate`.

**Context:** Card #40 needs a `CandidateScore` domain object and its
combination formula, referencing a candidate only by `candidate.ID`. This
is a much lighter dependency than `discovery`'s reason for being split
from `candidate` (needing both `spotify` and `candidate`), so the same
justification doesn't directly transfer.

**Reason:** The real distinction isn't dependency count, it's what kind of
concept each package owns. `candidate`'s existing `metadata.go`/
`provenance.go` (Cards #38/#39) describe *what a candidate is* — its own
identity and enrichment data. Scoring is not that: it's an *evaluation
performed on* a candidate, a downstream pipeline concern analogous to
`discovery` (which *produces* candidates), not to `metadata`/`provenance`
(which *describe* candidates already produced). M5 is also its own named
roadmap milestone, matching this repo's existing one-feature-named-
package-per-concern convention. Practically, the actual per-factor
algorithms future M5 cards will implement (Freshness needs playlist/
edition history, Repetition Penalty needs repetition history, etc.) will
very likely need `discovery`/`spotify`/`lastfm` — landing the model inside
`candidate` now would force an awkward later move out of the foundational
domain package once those dependencies appear.

**Consequences:** `backend/internal/scoring/score.go` defines `Factors`,
`Weights`, `DefaultWeights`, `ModelVersion`, `CandidateScore`, and
`Calculate`; `errors.go` follows the existing `candidate/errors.go`
sentinel-error pattern. `scoring` depends only on `candidate` (for
`candidate.ID`), not on `discovery`/`spotify`/`lastfm`. `CandidateTrack`
itself is completely unchanged — no field was added to it, per the card's
explicit instruction to prefer a separate domain object. No wiring into
`discovery`/`main.go` exists yet: no production code constructs real
`Factors` values, since every per-factor algorithm is still future M5
work; `Calculate` is exercised only by its own tests. See
[`docs/scoring-model.md`](../scoring-model.md) for the full model.

---

**Decision:** Combine the repetition penalty into `scoring.Calculate` as a
multiplicative discount (`FinalScore = BaseScore * (1 - repetitionPenalty
* Weights.RepetitionWeight)`), not a weighted subtraction; and treat a
missing positive factor by excluding it from a weight-renormalized
average, not by substituting `0.0`.

**Context:** Card #40 requires the final score formula to keep
`FinalScore` naturally bounded in `[0,1]` "without awkward normalization,"
explicitly forbidding "arbitrary clamping merely to force the range," and
separately requires that missing/not-yet-calculated factors never read as
an automatic negative signal.

**Reason:** A subtractive penalty (`BaseScore - penalty*weight`) can drive
an already-low `BaseScore` negative, which would need a `math.Max(0, ...)`
clamp to stay in range — exactly the arbitrary clamping the card
forbids. The multiplicative form is bounded by construction: since
`BaseScore ∈ [0,1]` and `repetitionPenalty*RepetitionWeight ∈ [0,1]` (both
enforced by `Weights.Validate()`/`Factors.Validate()`), `FinalScore` is
always in `[0, BaseScore] ⊆ [0,1]` with no clamp anywhere. It is also more
editorially explainable: repetition discounts a candidate's score by at
most `RepetitionWeight` (e.g. 30%), it never makes a strong candidate look
actively bad. Separately, computing `BaseScore` as `sum(value*weight) /
sum(weight)` over only the *available* positive factors (rather than
`sum(value*weight)` over all five, implicitly treating a missing one as
contributing `0`) is what satisfies "missing data must not become a
strong negative signal" — since M5 implements factors one card at a time,
every positive factor is unavailable as of Card #40, so getting this rule
right now is load-bearing for every following M5 card, not a hypothetical
edge case.

**Consequences:** `CandidateScore.AvailableWeight` exposes how much of the
positive-factor weight was actually used, so a future caller can judge a
given score's completeness. If zero positive factors are available,
`FinalScore` is `nil` rather than a fabricated `0.0`. A missing
`RepetitionPenalty` is treated as `0.0` (no discount) — silence about
repetition history is not evidence of repetition. See
[`docs/scoring-model.md`](../scoring-model.md) for the worked numeric
example and the full missing-factor strategy.

---

**Decision:** Introduce `backend/internal/musicaldna` as a new package
holding one shared `Profile` type (mood/energy/texture/cultural influence,
each optional) and two editorial contexts built from it — `ProjectDNA`
(stable, project-wide) and `WeeklyDirection` (contextual, per edition) —
and implement the Fit factor (`scoring.CalculateFit`, Card #41) against
them, rather than deriving Fit from `candidate.CandidateTrack` fields or
inventing a second Musical DNA representation.

**Context:** Card #41 requires Fit to evaluate "compatibility with Sound
Continuum DNA" and "compatibility with the current week's direction,"
explicitly forbidding genre matching, Spotify Audio Features/Analysis,
popularity, and fabricated precision. Before this card, "musical DNA" was
only an informal phrase (this file, the Card #29 entry above) and no
weekly/edition-direction model existed anywhere in the repo.

**Reason — package placement:** `Profile`/`ProjectDNA`/`WeeklyDirection`
are domain concepts in their own right (an editorial vocabulary and two
contexts it describes), not an evaluation *of* a candidate — the same
distinction that put `scoring` outside `candidate` (see the Card #40
entry above) argues against putting these inside `scoring` too. They also
don't belong in `candidate`: a candidate's own musical tagging is
supplied as a plain `musicaldna.Profile` argument to `CalculateFit`, not
stored on `CandidateTrack` (see below), so `candidate` never needs to
import this concept. `scoring` imports `musicaldna`, never the reverse —
no cycle.

**Reason — one `Profile` type, three roles:** A candidate's tagged
characteristics, `ProjectDNA`'s baseline, and `WeeklyDirection`'s target
are the same kind of thing — a small set of musical-character dimensions —
so they share one type. Introducing a second, differently-shaped
representation for any of the three would be exactly the "competing
definition of Musical DNA" the card forbids.

**Reason — `ProjectDNA.Profile` is empty by default:** `docs/manifesto.md`
and Sound Continuum's M1 decisions describe editorial *process and
philosophy* — continuum, musical bridges, discovery without forced
obscurity, human judgment — not concrete mood/energy/texture/cultural
values. Mapping each `Profile` dimension against the manifesto:

| Dimension | Manifesto/M1 support for a concrete value | Value set |
|---|---|---|
| Mood | None — manifesto principle 1 ("music is a continuum") explicitly rejects a fixed identity here | unset |
| Energy | None | unset |
| Texture | None | unset |
| CulturalInfluence | None — manifesto never addresses geography/culture; `project-context.md`'s "bridges matter more than rigid genre classification" argues against fixing one | unset |

Setting any of these would assert a permanent musical-identity claim the
manifesto does not make, and would contradict the continuum principle
`ProjectDNA` exists to encode. `DefaultProjectDNA()` returns an empty
`Profile` today, not a placeholder awaiting this package's own future
work — it is the accurate reflection of what the manifesto currently
specifies. A field should be set here only once a durable, project-wide
trait is actually documented in the manifesto or this file.

**Reason — `WeeklyDirection` carries more weight than `ProjectDNA`:**
`scoring.DefaultFitWeights()` splits the two components 0.75/0.25 in
`WeeklyDirection`'s favor, per the card's requirement and the manifesto's
"each week is a chapter" principle. Because `ProjectDNA` is empty by
construction today, this split is also realized structurally: the project
component contributes nothing to `CalculateFit` until a real dimension
value exists, so Fit is driven entirely by `WeeklyDirection` in practice,
not merely by a documented ratio.

**Reason — exact string match, no confidence, no override:** No
fuzzy/embedding similarity is available without the AI/ML this project
excludes; a partial-credit heuristic between arbitrary editorial strings
would itself be fabricated precision, so `compareProfiles` uses
case-insensitive, trimmed exact match only. A confidence score (Fit value
vs. confidence in that value) was considered and left out — it would
either be an unearned statistical claim or unnecessary complexity right
now; open question for a future card. No editorial-override mechanism was
built: none exists in Card #40 to preserve, and `Factors.Fit` is already a
plain settable `*float64` a human or future workflow can overwrite
directly — a low or missing Fit value never means "rejected."

**Reason — candidate profile is a parameter, not a `CandidateTrack`
field:** No automatic signal exists today for a candidate's own musical
character (no Spotify Audio Features, genre excluded as a proxy), so a
`musicaldna.Profile` for a candidate can currently only come from explicit
editorial tagging. `CalculateFit(candidateProfile, project, direction,
weights)` takes it as a plain argument rather than adding a field to
`candidate.CandidateTrack` — Card #38's struct stays completely unchanged,
and no persistence is introduced for candidate-level tagging (out of
scope per the MVP constraints). A future card can decide how/where such
tagging is captured and stored.

**Consequences:** `backend/internal/musicaldna/{profile,project_dna,
weekly_direction,errors}.go` define the domain types; `backend/internal/
scoring/fit.go` adds `FitDimension`, `FitComponent`, `FitWeights`,
`DefaultFitWeights`, `FitDimensionResult`, `FitResult`, and
`CalculateFit`. Missing dimensions/components are renormalized at both
levels — the same idiom `Calculate` already uses for `Factors` — so
`Fit` is `nil`, never `0.0`, when nothing is comparable.
`FitResult.Dimensions` carries all 8 (dimension × component) entries for
explainability regardless of whether `Value` is set. `CalculateFit` never
takes a `candidate.CandidateTrack`, so it is independent of
`CandidateType`/`Category`/release date/discovery provenance/playlist
sequence by construction, not by convention — see
`backend/internal/scoring/fit_test.go`'s explicit independence tests. No
change to `candidate.CandidateTrack`, `scoring.Factors`, `scoring.Weights`,
or `scoring.Calculate`. See [`docs/scoring-model.md`](../scoring-model.md)
for the full Fit model.

---

**Decision:** Implement the Freshness factor (Card #42) as a half-life
recovery curve, `Freshness(t) = 1 − 0.5^(t / HalfLifeDays)` with a default
60-day half-life, rather than a linear ramp or stepped buckets.

**Context:** Card #42 requires Freshness to measure time since a
candidate's most recent appearance in the official playlist, give a
gradient beyond the 28-day Recent Track Filter's hard boundary (Card #37),
and have no discontinuity at that boundary — explicitly ruling out "0 if
recently used else 1."

**Reason — half-life over the alternatives:** A linear ramp needs an
arbitrary saturation horizon (at what day does it reach 1.0?) with no
natural justification, and clamps discontinuously once reached. Stepped
buckets reintroduce exactly the kind of sharp jump the card forbids at
whichever bucket edge sits nearest 28 days. A half-life decay is
continuous and monotonic everywhere by construction, asymptotic (never
exactly reaching `1.0` for a used track, which keeps it visibly distinct
from a genuinely never-used candidate's exact `1.0`), and has exactly one
tunable constant — the simplest model satisfying the card's "avoid
arbitrary complexity" instruction.

**Reason — 60-day half-life:** The 28-day filter window is a hard
eligibility cutoff, not a freshness saturation point — the card explicitly
warns against assuming the filter's 28 days should become "the entire
Freshness model." Doubling it to 60 days keeps Freshness's horizon
anchored to a value the codebase already treats as editorially meaningful,
while giving 29–35-day-old candidates (just past the filter) a visibly
low Freshness (~0.28–0.33) and 120–180-day-old candidates a visibly high
one (~0.78–0.88), a clear gradient without inventing an unrelated
constant. At the boundary itself, 27/28/29 days apart differ by only
~0.017 — no discontinuity. See
[`docs/scoring-model.md`](../scoring-model.md) for the full value table.

**Reason — asymmetric handling of "never used" vs. "used long ago":**
`lastUsedAt == nil` (never appeared in the playlist) returns exactly
`1.0`; any used candidate, however old, approaches but never reaches
`1.0`. This preserves a real, if narrow, distinction the card requires:
"never used" and "used a very long time ago" are different facts about a
candidate, even though both should score high.

**Reason — clock-skew clamping:** If `now` is somehow before
`lastUsedAt`, `t` is clamped to `0` rather than producing a negative
duration or erroring. This is defensive handling for a condition the
model's own guarantees don't need to distinguish (any `t ≤ 0` means "just
used"), not a new failure mode requiring its own error path.

**Consequences:** `backend/internal/scoring/freshness.go` adds
`FreshnessConfig`, `DefaultFreshnessConfig`/`DefaultFreshnessHalfLifeDays`
(60), `FreshnessResult`, and `CalculateFreshness(lastUsedAt *time.Time,
now time.Time, config FreshnessConfig) (FreshnessResult, error)`. `now` and
`lastUsedAt` are both explicit parameters — never read from the system
clock inside the function — so Freshness is deterministic for a given
candidate/history/evaluation time, the same discipline Card #41 already
established for `CalculateFit`'s inputs.

---

**Decision:** Export `discovery.Service.PlaylistTrackHistory` as a thin
wrapper around Card #37's existing `recentTrackIndex`, and have
`scoring.CalculateFreshness` accept a plain `map[string]time.Time` via a
small `scoring.FreshnessLastUsedAt` lookup function, rather than having
`scoring` import `discovery` or building a second playlist-history
retrieval mechanism.

**Context:** Card #42 explicitly forbids a second playlist-history
mechanism — Freshness must reuse Card #37's retrieval, pagination, and
episode/duplicate-handling logic exactly. That logic
(`discovery.Service.recentTrackIndex`) was unexported and private to
`discovery.Service`, called only from `FilterRecentTracks`.

**Reason — export from `discovery`, don't duplicate into `scoring`:**
`discovery` already owns the only code that talks to
`spotify.Service.OfficialPlaylist`/`PlaylistItems` and walks pagination in
full. Exporting `PlaylistTrackHistory` (`OfficialPlaylist` lookup +
`recentTrackIndex`) as one new method, and refactoring
`FilterRecentTracks` to call it instead of duplicating the two calls
inline, collapses the codebase back down to exactly one retrieval
mechanism instead of introducing a second. This was a small, in-place
refactor of Card #37's own file, not a rewrite — `FilterRecentTracks`'s
behavior and its existing tests are unchanged.

**Reason — `scoring.CalculateFreshness` stays pure, `scoring` never
imports `discovery`:** `CalculateFit` (Card #41) established the pattern
that a scoring factor's calculator is a pure function over plain values,
with no I/O and no knowledge of where its inputs came from — kept
independent and trivially testable without mocks. Having `scoring` import
`discovery` (or vice versa) to fetch playlist history directly would
couple a pure calculation package to a retrieval/orchestration package for
no benefit; the two packages have no import relationship today and this
card doesn't need to create one. Instead, `PlaylistTrackHistory` returns a
plain `map[string]time.Time` (Spotify track ID → most recent `added_at`),
and `scoring.FreshnessLastUsedAt(history, spotifyTrackID)` — living in
`scoring`, operating only on that plain map — bridges the two without
either package needing to know about the other's types. Nothing wires
`CalculateFit` into production code yet either, so Freshness is
consistent in staying an equally unwired, pure library function until a
future ranking/orchestration card connects them.

**Reason — the reuse proof lives in `discovery`, not `scoring`:** A Go
test file's package can import anything its non-test code doesn't, so
`backend/internal/discovery/freshness_integration_test.go` (package
`discovery`, test-only) imports `scoring` to exercise the true end-to-end
path — real `PlaylistTrackHistory` output flowing into
`FreshnessLastUsedAt` and `CalculateFreshness` — without creating a
non-test import cycle or requiring `scoring` to depend on `discovery`.

**Consequences:** `backend/internal/discovery/recent_track_filter.go`
gains `PlaylistTrackHistory(ctx) (map[string]time.Time, error)`;
`FilterRecentTracks` now calls it instead of the inline
`OfficialPlaylist`+`recentTrackIndex` pair. `backend/internal/
scoring/freshness.go` gains `FreshnessLastUsedAt`. `scoring` still imports
only `candidate`/`musicaldna`; `discovery` still imports only
`spotify`/`candidate` in its non-test code — no new production import
edges either direction.

---

**Decision:** `scoring.CalculateDiscoveryBonus(category candidate.Category,
editorialDiscoveryValue *float64) (DiscoveryBonusResult, error)` takes an
explicit editorial discovery value as a plain function parameter, never a
`candidate.CandidateTrack` field, and never derives it from discovery
provenance, Last.fm similarity, `CandidateType`, or popularity.

**Context:** Card #43 defines the Discovery Bonus factor: it must reward
`CategoryEmerging` candidates only when there is genuine, editorially
demonstrated discovery value — never simply because a candidate is unknown,
was found via Last.fm, or has a high Last.fm similarity score. The card is
explicit that no automatic signal exists (or should exist) for "how
valuable is this discovery" — that judgment is inherently human.

**Reason — parameter, not a `CandidateTrack` field:** This mirrors Card
#41's `musicaldna.Profile` decision exactly: no automatic signal exists
today for a candidate's genuine discovery value, so it can currently only
come from explicit editorial assessment. `CalculateDiscoveryBonus` takes it
as a plain argument rather than adding a field to `candidate.CandidateTrack`
— that struct stays completely unchanged by this card, consistent with the
project's standing discipline of not adding scoring-only state to the
domain model ahead of a real caller (see the Card #38 `CandidateMetadata`
precedent for the *opposite* case — Spotify-sourced metadata genuinely
belongs on `CandidateTrack` because it's a fact about the track, not an
editorial judgment call made per scoring run).

**Reason — discovery provenance and Last.fm similarity are excluded from
the formula:** `candidate.DiscoveryProvenance.LastFMMatch` has been
documented since Card #39 as "discovery metadata only... never a ranking
signal." Reading it into Discovery Bonus would directly contradict that
existing architectural boundary and would conflate "how this candidate was
found" with "how valuable it is to surface" — two different questions. A
candidate discovered through an unusual or unremarkable path can still have
strong or weak discovery value independently; provenance remains available
on `CandidateTrack.Provenance` for a caller to display *alongside* a
`DiscoveryBonusResult`, never combined into the calculation itself.

**Reason — `CategoryEmerging` is necessary but not sufficient:** The
manifesto's "discovery without forced obscurity" principle (M1) means an
emerging/unknown artist is not automatically valuable — obscurity itself is
never a positive signal. Gating on `category == candidate.CategoryEmerging`
alone, with no explicit value supplied, would silently reward the wrong
thing (`if Category == Emerging: DiscoveryBonus = 1.0`, the exact shortcut
the card forbids). Reusing `candidate.CategoryEmerging` — the existing
editorial classification, not a new "EmergingArtist" flag — for eligibility
keeps this factor consistent with the rest of the scoring model's use of
`Category`, and keeps it independent of `CandidateType` by construction: a
`Discovery`-type candidate is not necessarily `Emerging`, and this function
never even accepts `CandidateType` as input.

**Reason — missing vs. explicit zero must stay distinguishable:** Per the
scoring model's existing missing-factor convention (Card #40), an
unavailable positive factor must never read as a negative signal. This
matters more for Discovery Bonus than any other factor: if "no editorial
assessment yet" silently became `0.0`, every newly-discovered Emerging
candidate would default to "worthless" until a human got around to
assessing it — the opposite of the intended behavior. `DiscoveryBonusResult`
exposes `Eligible`/`Supplied` independently of `Value` so "not eligible,"
"eligible but unassessed," and "eligible and scored" are all distinguishable
states, and an explicit `EditorialDiscoveryValue = 0.0` ("assessed as no
discovery value") is preserved exactly, never collapsed into the same `nil`
that represents "not yet assessed."

**Reason — no weights/config struct:** Unlike `FitWeights` (multiple
dimension weights) or `FreshnessConfig` (a half-life constant), v1's
formula is the identity function on the supplied value — there is no
editorial knob to name. Adding a struct with no configurable field would be
speculative machinery ahead of a real v2 need (e.g. a future curve or
multi-signal blend), which Card #43 explicitly defers ("the first version
should intentionally be this simple").

**Reason — Spotify popularity/followers exclusion is structural, not
tested:** These fields were already removed from this project's Spotify
model entirely for Development Mode (Card #29's decision above) — there is
no field anywhere in the codebase that could leak into Discovery Bonus even
by mistake, so no synthetic test fixture was built to "prove" the
exclusion.

**Consequences:** `backend/internal/scoring/discovery_bonus.go` adds
`DiscoveryBonusResult` and `CalculateDiscoveryBonus`; `errors.go` gains
`ErrDiscoveryBonusValueOutOfRange`. `score.go` needed zero changes — `
Factors.DiscoveryBonus`, `Weights.DiscoveryBonus = 0.15`, and `Calculate`'s
renormalization already handled it generically since Card #40. Like
`CalculateFit`/`CalculateFreshness`, `CalculateDiscoveryBonus` is not wired
into `pool.go`/`main.go`/any handler — it remains a pure, tested library
function until a future ranking/orchestration card connects it, consistent
with M5's "implement one factor at a time" precedent.

---

**Decision:** Implement Diversity (Card #44) entirely inside the existing
`backend/internal/scoring` package (`diversity.go`), not a new package and
not a new persisted Edition entity.

**Context:** Card #44 needs Diversity to be evaluated against "the current
edition being assembled" — context distinct from `candidate.CandidateTrack`,
the candidate pool, and the official historical Spotify playlist — while
explicitly forbidding persistence or a new Edition domain model at this
stage.

**Reason:** Same reasoning already recorded for Fit/Freshness/Discovery
Bonus: `scoring` is the one package whose job is defining how editorial
signals combine into a score, and Diversity needs nothing `scoring` doesn't
already have access to (`musicaldna.Profile` is already an import via
`fit.go`). A `CurrentEditionContext{Tracks []EditionTrack}` struct,
constructed by the caller for one scoring call, satisfies "the smallest
appropriate function-level/context-level representation" the card asks
for — introducing a database table or a `candidate`/`discovery`-level
Edition type now would be exactly the ahead-of-need architecture this
project's MVP discipline avoids everywhere else.

**Consequences:** `EditionTrack` carries only what Diversity needs
(`ArtistSpotifyIDs []string`, `Era *string`, `Sound musicaldna.Profile`) —
not a copy of `CandidateTrack`. A future persistence/Edition-assembly card
can build real Edition state and map it into `CurrentEditionContext` at the
call site without this package needing to change.

---

**Decision:** Reuse `musicaldna.Profile` (Card #41's shared vocabulary) for
Diversity's Sound dimension, but compute Sound Diversity with an
independent formula from `CalculateFit`'s `compareProfiles`, rather than
inventing a separate `DiversityProfile` type or calling into Fit.

**Context:** Card #44 explicitly requires Sound Diversity to use an
explicit, editorially-supplied sound representation — never genre, never
Spotify Audio Features — and explicitly forbids reusing Fit's calculated
score as Diversity, while allowing reuse of an existing representation "if
it can be reused without coupling Diversity to Fit."

**Reason:** `musicaldna.Profile` already is exactly the representation the
card describes — explicit, deterministic, editorially supplied,
independent of genre/Audio Features — so defining a second,
structurally-identical `DiversityProfile` type would be the "second artist
identity model" kind of duplication the card warns against elsewhere,
applied to sound instead. Sharing the *type* is safe because Diversity and
Fit use it for structurally different comparisons: Fit's
`compareProfiles` scores one candidate against one fixed target
(ProjectDNA or WeeklyDirection); Diversity's `soundDiversity` scores one
candidate's concentration against a population of edition tracks' profiles
using the `1/(1+occurrences)` formula below. Neither function calls the
other, and `CalculateDiversity` never receives a `FitResult` or `Fit`
value as input.

**Consequences:** Adding a fifth `musicaldna.Profile` dimension in a future
card updates both Fit's and Diversity's dimension lists automatically with
no cross-package coordination needed beyond the shared type.

---

**Decision:** Use `contribution(occurrences) = 1 / (1 + occurrences)` as
the one deterministic diminishing-concentration formula for all three
Diversity dimensions (and each Sound sub-dimension), instead of hardcoded
per-occurrence thresholds (e.g. "first appearance = 1.0, second = 0.5,
third = 0.0").

**Reason:** Card #44 explicitly requires "a simple deterministic
diminishing-concentration calculation" and explicitly forbids arbitrary
hardcoded thresholds. `1/(1+n)` is the simplest function satisfying
"0 occurrences is strongest, monotonically decreasing, bounded `(0,1]`,
never treats one additional track by an artist as automatically invalid" —
one formula, reused for Artist, Era, and each Sound sub-dimension, rather
than three separate hand-tuned curves.

**Consequences:** The curve cannot reach exactly `0`, by design — "no
meaningful diversity" and "this candidate hasn't contributed any diversity
yet" are different claims than "there is literally zero value"; a future
card revisiting this formula should preserve the bounded-`(0,1]`,
monotonically-decreasing, no-hard-threshold properties the card requires.

---

**Decision:** An empty `CurrentEditionContext` (non-nil, zero tracks) and
a `nil` `CurrentEditionContext` both yield `Diversity = nil`, distinguished
via `DiversityResult.ContextProvided`/`EditionEmpty` — neither
automatically returns `1.0`.

**Context:** Card #44 explicitly warns that "there is no concentration
yet" is not the same claim as "this candidate has been evaluated as
maximally diverse," and separately requires missing context to yield
`nil`, never a fabricated maximum or minimum.

**Reason:** The two cases have genuinely different meanings (no context
supplied vs. an edition that's legitimately just getting started) and the
card asks for both to be documented explicitly — collapsing them into one
unexplained `nil` would lose that distinction, so both fields are carried
on `DiversityResult` rather than only a bare `nil` `Value`.

**Consequences:** A caller that wants to special-case "first track of a
new edition" differently from "no context was wired up yet" can do so from
`DiversityResult` alone, without re-deriving the distinction.

---

**Decision:** No `DiversityWeights` config struct; the three aspect
weights (Artist/Era/Sound, 1/3 each) are fixed constants in
`CalculateDiversity`, and `CalculateDiversity` returns no `error`.

**Context:** Card #44 explicitly says "do not invent a complex hierarchy
of diversity weights" and specifies equal 1/3 weighting as the first
version. Unlike `FitWeights`/`FreshnessConfig`, nothing about Diversity's
inputs is a caller-supplied float or struct that could be out of range.

**Reason:** Same reasoning already recorded for Discovery Bonus's lack of
a weights struct: a config type for a value with no real caller and no
planned variation is exactly the ahead-of-need complexity this project's
MVP discipline avoids. Since there is no `Weights`/`Config` to validate and
no caller-supplied numeric value needing a range check (occurrences are
computed internally from edition data, never passed in directly),
`CalculateDiversity` has no error path to report — unlike
`CalculateFit`/`CalculateFreshness`/`CalculateDiscoveryBonus`, which all
validate a caller-supplied `Weights`/`Config`/editorial value.

**Consequences:** If a future card needs configurable Artist/Era/Sound
weighting, add a `DiversityWeights` struct (and the matching `error`
return) then, following `FitWeights`'s `Validate()` pattern — not ahead of
a real need.

---

**Decision:** Implement Repetition Penalty (Card #45, M5's final factor) as
`max(TrackRepetitionSeverity, ArtistRepetitionSeverity)`, with each
dimension a linear decay to zero at a 90-day horizon
(`DefaultRepetitionHorizonDays`), rather than summing the two dimensions or
reusing the Recent Track Filter's 28-day window or Freshness's 60-day
half-life as the horizon.

**Context:** Card #45 needs a soft scoring signal for recent track/artist
reuse, distinct from the Card #37 hard 28-day eligibility filter and the
Card #42 Freshness factor, following the M1 repetition philosophy that
repetition is never an absolute prohibition.

**Reason — max over sum:** A track repetition event is already an artist
repetition event (the track has an artist). Summing both severities would
double-count the same historical fact and could push a candidate's penalty
higher than either dimension alone justifies; `max` reflects "the
strongest single reason this is a repetition concern" and keeps the result
bounded in `[0,1]` by construction, with no clamping needed.

**Reason — linear decay, not half-life:** The card explicitly recommends a
simple deterministic linear decay over a complex decay function for a
first version. Unlike Freshness, which deliberately never reaches exactly
0/1 for a used track (to stay visibly distinct from "never used"),
Repetition Penalty needs a horizon where the penalty reaches exactly zero
— an asymptotic curve structurally cannot do that, so half-life was not
reused here even though it was the right choice for Freshness.

**Reason — a horizon independent of both existing recency constants:**
Reusing the 28-day hard-filter window as the full penalty model would make
Repetition Penalty redundant with the Recent Track Filter — a track used
29 days ago is eligible but would carry almost no soft memory if the
horizon were also ~28 days. Reusing Freshness's 60-day half-life would
conflate two different curves for two different purposes (asymptotic
recovery vs. a bounded penalty). 90 days is its own simple, explainable
constant: long enough to give repetition memory real reach past the hard
filter (a 45-day-old use still carries a real, weaker-than-day-10
penalty), short enough that reuse is never penalized indefinitely.

**Consequences:** `backend/internal/scoring/repetition_penalty.go` adds
`RepetitionPenaltyConfig`, `DefaultRepetitionPenaltyConfig`/
`DefaultRepetitionHorizonDays` (90), `RepetitionComponentResult`,
`RepetitionPenaltyResult`, `CalculateRepetitionPenalty`, and
`RepetitionArtistLastUsedAt`. `errors.go` gains
`ErrRepetitionHorizonOutOfRange`. `CalculateRepetitionPenalty` takes only
`*time.Time`/`time.Time`/config — never a `candidate.CandidateTrack` — so
it is independent of `CandidateType`/`Category`/the other five factors by
construction, the same guarantee Fit/Freshness/Discovery Bonus/Diversity
already give. Like those four, it is not wired into
`pool.go`/`main.go`/any handler — a pure, tested library function until a
future ranking/orchestration card connects it. No change to
`scoring.Factors`, `scoring.Weights` (`RepetitionWeight` stays `0.30`), or
`scoring.Calculate`.

---

**Decision:** Extend `discovery`'s existing playlist-history walk
(`recentTrackIndex`, Card #37) to also build a Spotify-artist-ID history
index in the same pagination pass, exposed as a new
`Service.PlaylistArtistHistory`, rather than writing a second
playlist-retrieval mechanism for Repetition Penalty's artist dimension.

**Context:** Card #45 explicitly forbids a second playlist-history source
and requires artist repetition to use a stable Spotify artist identity.
`recentTrackIndex`'s existing walk already iterates every playlist item's
decoded `spotify.Track`, which already carries `Artists []spotify.Artist`
(each with `ID`) — the data needed for an artist index was already being
read and discarded on every call.

**Reason:** This is the same category of change Card #42 already made to
this file (extracting `PlaylistTrackHistory` out of the private walk
without altering Card #37's behavior or tests) — a small, in-place
extension, not a rewrite. `recentTrackIndex` is refactored into a thin
wrapper over a new `recentTrackAndArtistIndex`, which builds both maps
(track ID -> max `added_at`, artist ID -> max `added_at` across any of
that artist's tracks) from one pagination pass instead of two, so
`PlaylistArtistHistory` costs no extra Spotify calls beyond what
`PlaylistTrackHistory` already makes. `recentTrackIndex`'s own return
value and every existing Card #37/#42 test are unchanged — this is proven
by a new test
(`TestPlaylistTrackHistoryUnaffectedByArtistExtension`), not just
asserted.

**Consequences:** `backend/internal/discovery/recent_track_filter.go`
gains `Service.PlaylistArtistHistory(ctx) (map[string]time.Time, error)`,
`playlistHistoryIndexes`, and `recentTrackAndArtistIndex` (private).
Episode/unavailable-item skipping and duplicate-entry max-`added_at`
handling are identical for both dimensions, inherited from the single
shared walk. `FilterRecentTracks`'s signature, `RecentTrackFilterResult`,
and `DefaultRecentTrackLookbackDays` are all untouched.

---

**Decision:** Reuse `candidate.CandidateMetadata.Artists[].SpotifyArtistID`
for Repetition Penalty's artist identity — the same field Diversity (Card
#44) already reuses for its Artist dimension — rather than introducing a
second artist-identity representation.

**Context:** Card #45 requires artist repetition to use a stable Spotify
artist ID when available, not an artist name, and explicitly forbids a
second artist-identity model.

**Reason:** `CandidateArtist.SpotifyArtistID` is already this project's one
stable artist identity, established by Card #38 and already reused by
Diversity. A candidate can carry more than one artist, so
`RepetitionArtistLastUsedAt(artistHistory, spotifyArtistIDs []string)`
takes every one of a candidate's artist IDs and returns the single most
recent occurrence across all of them — not just the first — so a
multi-artist candidate's repetition reflects whichever of its artists was
actually used most recently.

**Consequences:** No new artist-identity field or type was added anywhere.
A future caller wiring Repetition Penalty into a real pipeline passes
`candidate.Metadata.Artists`' `SpotifyArtistID` values directly, exactly as
a future Diversity caller would.
