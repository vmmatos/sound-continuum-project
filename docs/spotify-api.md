# External Music APIs — Spotify & Last.fm Research (Card 23)

Despite the filename, this document covers external music data source
research broadly: the current Spotify Web API and the Last.fm API, as
candidate sources for Sound Continuum's catalog, discovery, and musical
bridge needs. **Research only — no integration exists yet.**

Research date: September 2026. All claims below are sourced from official
documentation fetched during this research pass (see [Sources](#13-sources));
where docs were ambiguous, that is stated explicitly rather than guessed.

---

## 1. Executive Summary

Spotify's Web API changed substantially between November 2024 and 2026, and
those changes directly break the assumptions Sound Continuum's roadmap was
built on:

- **Audio Features, Audio Analysis, Recommendations, and Related Artists are
  all deprecated for apps without pre-existing extended-quota access.**
  Sound Continuum has no such legacy access (it's a new app), so none of
  these are available. This removes the entire technical basis originally
  imagined for "musical bridge" signals (tempo, energy, danceability,
  valence, artist similarity graphs).
- **Extended Quota Mode is now gated behind an organizational application**
  (established business, 250k+ monthly active users, six-week review). Sound
  Continuum, as a solo/small-editorial MVP, will realistically **stay in
  Development Mode indefinitely** — capped at 5 allowlisted users, lower rate
  limits, and the app owner must hold Spotify Premium.
- **Search got smaller, not bigger**: max `limit` is now 10 (default 5), and
  several browse/discovery endpoints (`new-releases`, `categories`,
  `artists/{id}/top-tracks`, `markets`) were removed outright in the
  February 2026 migration.
- **Playlist track management moved from `/tracks` to `/items`**, and
  playlist item contents are now returned only for playlists the requesting
  user owns or has explicit access to.
- **Last.fm's similarity/tag endpoints require no authentication** (API key
  only) and remain a genuinely useful, currently-available source for artist
  similarity, track similarity, and contextual tags — none of which Spotify
  offers anymore to a new app.

Net effect: Spotify is still the right system for catalog identity and
playlist publishing, but it can no longer be assumed to provide discovery or
musical-bridge signals. Last.fm can plausibly fill part of that gap for
similarity/tags, but nothing currently available (from either source)
replaces audio-content signals like tempo or energy — those remain editorial
judgment. See [Section 5](#5-sound-continuum-data-strategy) and
[Section 6](#6-musical-bridge-implications).

---

## 2. Spotify Web API

### 2.1 Authentication

Three active OAuth 2.0 flows ([Authorization docs][spotify-auth]):

| Flow | User resource access | Server secret required | Refresh |
|---|---|---|---|
| Authorization Code | Yes | Yes | Yes |
| Authorization Code + PKCE | Yes | No | Yes |
| Client Credentials | No (app-only) | Yes | No |

Implicit Grant is deprecated, not recommended.

- **Access tokens** last 1 hour (3600s) ([Access Token concept][spotify-access-token]).
- **Refresh tokens** now expire after **6 months** from initial user
  authorization — a new policy, effective immediately for new apps and from
  **2026-07-20** for existing apps. Refreshing an access token does **not**
  reset this 6-month clock. Expired refresh tokens return `400
  invalid_grant`; the user must re-authorize from scratch
  ([refresh token expiration blog][spotify-refresh-expiry]).
- **Development Mode**: as of the February 2026 migration, the app owner
  must hold an active Spotify **Premium** subscription, or the app stops
  working; up to 5 allowlisted users; 1 Client ID per developer for new apps
  (raised to 25 per developer account in July 2026, with dev-mode quota now
  counted **per developer account**, not per Client ID)
  ([Feb 2026 migration guide][spotify-feb2026], [July 2026 changelog][spotify-jul2026]).
- **Extended Quota Mode**: unlimited users, higher rate limits, but as of
  May 2025 the application process requires an **organization** — an
  established business entity, an active launched service, **250,000+
  monthly active users**, key-market availability, and commercial viability
  — reviewed over up to six weeks via a Partner Application form
  ([Quota Modes concept][spotify-quota-modes]).

**Implication for curator/playlist auth:** Authorization Code (with PKCE if
the curator-facing client is a browser/SPA) is the right flow for a human
curator authenticating to manage playlists on their own Spotify account.
Client Credentials is only useful for catalog lookups (search, public track
metadata) that don't need a specific user's identity — it cannot create or
modify playlists. Given Sound Continuum's small, editorial nature, Extended
Quota Mode is very unlikely to be attainable soon; **design for Development
Mode constraints (5 users, Premium-gated) as the baseline, not an exception.**

### 2.2 Search

`GET /search` ([reference][spotify-search]):

- Required: `q` (query, supports field filters `album`, `artist`, `track`,
  `year`, `upc`, `isrc`, `genre`, `tag:new`, `tag:hipster`) and `type`
  (comma-separated: `album`, `artist`, `playlist`, `track`, `show`,
  `episode`, `audiobook` — audiobook limited to a handful of English-speaking
  markets).
- Optional: `market`, `include_external`, **`limit` (default 5, max 10 — down
  from a historical max of 50)**, `offset` (default 0, max 1000).
- `tag:new` = albums released in the last two weeks; `tag:hipster` = albums
  with the lowest popularity in that genre — the closest thing Spotify has
  to an "emerging" signal, and it's a popularity-inverse proxy, not an
  editorial concept (see [Section 7](#7-emerging-artist-discovery)).
- Genre and year filters do support genre-based and year-based discovery
  queries; there is no dedicated "emerging artist" filter or endpoint.
- **The max-10-per-page limit is a real constraint**: any workflow that
  wants to browse candidates broadly now needs many more paginated requests
  than before.

### 2.3 Artists

- **Artist metadata, genres, popularity, followers, images, albums**: still
  available via `GET /artists/{id}` and `GET /artists/{id}/albums`, using
  Client Credentials or user auth.
- **Artist Top Tracks** (`GET /artists/{id}/top-tracks`): **removed** in the
  February 2026 migration ([migration guide][spotify-feb2026]).
- **Related Artists** (`GET /artists/{id}/related-artists`): marked
  **Deprecated**, restricted to apps with pre-existing extended-quota access
  from before 2024-11-27 ([reference][spotify-related-artists],
  [Nov 2024 announcement][spotify-nov2024]).

### 2.4 Tracks

Track objects still expose ID, URI, name, artists, album, duration,
`explicit`, `preview_url` (single-track responses; batch multi-get preview
URLs were restricted in the Nov 2024 change). `popularity` and
`available_markets` were marked removed in the February 2026 migration guide
but the `external_ids` field removal was **reverted** in March 2026 — a
concrete example of Spotify walking back a breaking change within weeks.
Given that reversal, treat any single changelog entry as provisional until
cross-checked against the live endpoint at implementation time.

### 2.5 Audio Features & Audio Analysis — CRITICAL

**Both endpoints (`GET /audio-features/{id}` and
`GET /audio-analysis/{id}`) are marked Deprecated and gated behind
pre-2024-11-27 extended-quota access** ([Nov 2024 announcement][spotify-nov2024],
confirmed on the [Audio Features reference page][spotify-audio-features]).

- **New applications cannot access** danceability, energy, valence, tempo,
  key, mode, loudness, acousticness, instrumentalness, liveness, or
  speechiness. Sound Continuum has never had access to these (it's a new
  app), so this isn't a regression to plan around — it's a door that was
  never open.
- **Apps that already had extended-quota access before 2024-11-27 keep
  access.** Development Mode status doesn't matter for this — what matters
  is whether the app existed and had that access pre-cutoff. There is no
  path for a new app to acquire it; Spotify has stated no intent to reopen
  it and offered **no replacement endpoint**.
- Extended Quota Mode (2026 rules) does not restore this — the 2024
  restriction is a separate, harder gate than the general quota system.
- **Implication for musical bridges:** the entire tempo/energy/valence-based
  bridge model the manifesto's technical imagination once implied is not
  buildable on Spotify data, full stop. See
  [Section 6](#6-musical-bridge-implications).

### 2.6 Recommendations & Related Artists

Both `GET /recommendations` and `GET /artists/{id}/related-artists` are
marked **Deprecated** and subject to the same 2024-11-27 extended-quota gate
as Audio Features. No new app can call either meaningfully — even though
the endpoints haven't been fully removed from the docs, calls from a new app
return `403`. **Do not design Sound Continuum's discovery engine around
either.**

### 2.7 Playlists

- **Create Playlist**: current path is `POST /me/playlists` (creates for the
  current user); a separate `create-playlist-for-user` endpoint still
  appears in the reference at time of research, but the February 2026
  migration guide lists `POST /users/{user_id}/playlists` among removed
  endpoints — this is an inconsistency between two docs pages fetched in
  this session; **treat the legacy user-scoped path as unreliable and plan
  around `POST /me/playlists`.** Flagged in [Open Questions](#12-open-questions).
- **Playlist items**: `GET/POST/DELETE /playlists/{id}/tracks` are
  deprecated in favor of `/playlists/{id}/items` (same verbs, `tracks`
  request/response field renamed to `items`). Required scope for reading:
  `playlist-read-private`. Writing needs `playlist-modify-public` and/or
  `playlist-modify-private`.
- Reordering and replacing playlist contents work through the same `/items`
  endpoint family (partial `PUT`/`POST`/`DELETE` semantics carried over from
  the old `/tracks` endpoints).
- **February 2026 migration guide states playlist item contents are now only
  returned for playlists the requesting user owns or collaborates on** — the
  live `/items` reference page fetched in this session didn't explicitly
  restate that restriction, another cross-doc inconsistency; verify directly
  against a live call before relying on it for M3 design.
- Playlist metadata update (name, description, public/private) and cover
  image upload are separate, unaffected endpoints.

### 2.8 User / Library APIs

Relevant to MVP scope only:

- **Current user profile** (`GET /me`): removed fields per the Feb 2026
  guide include `country`, `email`, `explicit_content`, `product`; a new
  `account_id` (stable, pseudoanonymous) field was added in May 2026 and
  should be preferred over `id` for any external linking.
- **Saved tracks / followed artists / library "contains" checks**: the
  February 2026 migration **consolidated** seven entity-specific
  save/remove/contains endpoints into three generic ones —
  `PUT /me/library`, `DELETE /me/library`, `GET /me/library/contains` —
  all keyed by Spotify URI instead of per-type IDs.
- **User's playlists / top tracks/artists / recently played**: not needed
  for the MVP's playlist-publishing use case; not investigated further per
  the card's "only what's relevant" instruction.
- `GET /users/{id}` and `GET /users/{id}/playlists` were removed in the
  February 2026 migration — don't design around looking up other users by ID.

### 2.9 Rate Limits & Quotas

- Enforced on a **rolling 30-second window**; exceeding the app tier's
  threshold returns `429` with a `Retry-After` header in seconds
  ([Rate Limits concept][spotify-rate-limits]).
- Development Mode has a materially lower ceiling than Extended Quota Mode.
- As of July 2026, dev-mode quota is tracked **per developer account**
  across all of that developer's Client IDs (previously per Client ID) —
  relevant if Sound Continuum ever splits environments (e.g. staging vs.
  prod app registrations) since they'd now share one bucket.
- The 429 body format was made more structured in July 2026:
  `{"status": 429, "message": "Too many requests", "reason":
  "QUOTA_EXCEEDED"}`.
- **Practical implication for a weekly curation workflow:** a human curator
  making occasional search/playlist calls once a week is nowhere near
  Development Mode's ceiling. Simple exponential backoff on 429/`Retry-After`
  is more than sufficient — **do not build a rate-limiting framework for
  this traffic pattern.**

### 2.10 Spotify Deprecations & Breaking Changes Timeline

- **2024-11-27** — Related Artists, Recommendations, Audio Features, Audio
  Analysis, Get Featured Playlists, Get Category's Playlists, and
  multi-get preview URLs restricted to apps with pre-existing extended-quota
  access. Stated reason: "creating a more secure platform"; no replacement
  offered ([announcement][spotify-nov2024]).
- **2026-02** — Major migration: batch/multi-get endpoints removed (fetch
  items individually now); `/browse/new-releases`, `/browse/categories`,
  `/artists/{id}/top-tracks`, `/markets`, `/users/{id}`,
  `/users/{id}/playlists` removed; library save/remove/contains endpoints
  consolidated to URI-based `/me/library*`; playlist `/tracks` → `/items`
  rename; search `limit` max cut from 50 to 10; several object fields
  removed; Development Mode apps now require the owner to hold Premium
  ([migration guide][spotify-feb2026]).
- **2026-03** — `external_ids` removal (on albums and tracks) from the Feb
  2026 change was **reverted** ([March 2026 changelog][spotify-mar2026]).
- **2026-05** — `account_id` field added to `GET /me`
  ([May 2026 changelog][spotify-may2026]).
- **2026-05** (policy, not changelog) — Extended Quota Mode applications
  restricted to organizations with 250k+ MAU ([Quota Modes][spotify-quota-modes]).
- **2026-06/07** — Refresh tokens gained a 6-month expiry; rollout to
  existing apps completed 2026-07-20 ([blog][spotify-refresh-expiry]).
- **2026-07** — Client ID cap per developer raised 1 → 25; dev-mode quota
  now counted per developer account, not per Client ID; structured 429
  error body ([July 2026 changelog][spotify-jul2026]).

#### Spotify "Do Not Build Against" list

- ❌ Audio Features / Audio Analysis (any of the 11 attributes)
- ❌ `GET /recommendations`
- ❌ `GET /artists/{id}/related-artists`
- ❌ `GET /artists/{id}/top-tracks`
- ❌ `GET /browse/new-releases`, `/browse/categories*`
- ❌ `GET /markets`, `GET /users/{id}`, `GET /users/{id}/playlists`
- ❌ Any multi-get/batch endpoint (`/tracks?ids=`, `/albums?ids=`, etc.) — one
  item per call now
- ❌ Assuming Extended Quota Mode is attainable for a small/solo editorial
  project
- ❌ Assuming a refresh token, once obtained, is valid indefinitely
- ⚠️ Treat `/users/{user_id}/playlists` (create-for-user) as unreliable —
  conflicting docs found this session

---

## 3. Last.fm API

### 3.1 Authentication

Per the [API Intro][lastfm-intro]: **all write services require full
authentication (signed session)**; the methods relevant to Sound Continuum's
discovery use case (`artist.getInfo`, `artist.getSimilar`, `track.getSimilar`,
`tag.*`) are confirmed, method-by-method, to **not require authentication —
an API key alone is sufficient** ([artist.getSimilar][lastfm-artist-getsimilar],
[track.getSimilar][lastfm-track-getsimilar], [tag.getSimilar][lastfm-tag-getsimilar],
[artist.getInfo][lastfm-artist-getinfo]). Sound Continuum's MVP has no need
for authenticated/write Last.fm calls (no scrobbling, no user accounts) — an
API key is the entire authentication requirement.

### 3.2 Artist Discovery

- **`artist.getInfo`**: bio (truncated to 300 chars), tags, similar artists,
  listener count, total playcount, images, MBID, Last.fm URL. No auth
  needed.
- **`artist.getSimilar`**: list of similar artists with a `match` score
  (0–1, "similarity value"). No auth needed.
- **`artist.getTopTags`, `artist.getTopTracks`, `artist.getTopAlbums`**:
  listed in Last.fm's docs alongside the above; not individually fetched
  this session but follow the same unauthenticated, API-key-only pattern per
  the Intro page's stated model — verify each at implementation time rather
  than assuming.

### 3.3 Track Discovery

- **`track.getSimilar`**: similar tracks with match percentages, artist
  details, URLs, cover art. No auth needed.
- **`track.getInfo`, `track.getTopTags`**: not fetched this session; same
  caveat as above.
- Last.fm's similarity data comes from **listening/scrobble co-occurrence**,
  not audio content analysis — it's a "people who played X also played Y"
  signal, not a tempo/key/timbre signal. Useful for discovery adjacency, not
  a substitute for audio-feature-based bridges.

### 3.4 Tags

- **`tag.getSimilar`**: tags ranked by similarity, "based on listening
  data." No auth needed.
- **`tag.getTopArtists`, `tag.getTopTracks`, `tag.getTopAlbums`**: referenced
  in Last.fm's docs navigation as companion methods; not individually
  verified this session.
- Tags are **user-submitted folksonomy**, not objective musical properties —
  a tag like "afrobeat" or "chill" reflects how listeners categorized a
  track/artist, not a measured audio characteristic. Treat tags as cultural/
  contextual signal, never as a stand-in for audio analysis.

### 3.5 Last.fm Listening / Popularity Signals

`artist.getInfo` and (per pattern) track-level equivalents expose
`listeners` and `playcount`. These are raw Last.fm-scrobble-based popularity
signals — useful for distinguishing "well-known on Last.fm" from "obscure,"
but **not a proxy for editorial quality or for Sound Continuum's "emerging"
concept** (see [Section 7](#7-emerging-artist-discovery)). Charts and
tag-ranking endpoints exist per Last.fm's docs structure but weren't
individually verified this session.

### 3.6 Last.fm API Limits & Terms

- **Rate limit**: search results (not the primary ToS page) cite "5 requests
  per originating IP per second, averaged over a 5-minute period" without
  prior written consent — this figure comes from a secondary/community
  source, not confirmed verbatim on the ToS page fetched this session.
- **ToS-stated "Reasonable Usage Cap"**: "a maximum of 100 MB" (of data),
  "as may be updated by Last.fm from time to time" — ambiguous whether this
  is per-day, per-key, or otherwise; flagged as an [Open Question](#12-open-questions).
- **Caching**: developers "must implement suitable caching in accordance
  with the HTTP headers sent with web service responses" — Last.fm expects
  callers to cache and not re-fetch the same data repeatedly
  ([ToS][lastfm-tos]).
- **Attribution is mandatory**: must credit Last.fm and link to the Last.fm
  site when displaying Last.fm Data — "powered by AudioScrobbler" button,
  and artist/album/track pages must link to the corresponding
  `last.fm/music/<artist>` catalog page.
- **Commercial use requires a separate written agreement** with Last.fm
  (contact `partners@last.fm`); non-commercial use is permitted by default
  under the standard ToS.
- No personal identification of Last.fm users; no sub-licensing; no DRM
  applied to the data; audio/audiovisual content itself is explicitly
  excluded from the agreement (only metadata, not playback).

These are **technical possibility + literal ToS text as fetched**, not legal
advice — a real go/no-commercial-use decision should get an actual legal
read before Sound Continuum monetizes anything.

---

## 4. Spotify vs Last.fm

| Capability | Spotify | Last.fm | Sound Continuum relevance |
|---|---|---|---|
| Search artists | Available, OAuth or Client Credentials | Not a dedicated search; use `artist.getInfo`/`getSimilar` by name | Spotify for resolving to a playable/publishable ID |
| Search tracks | Available, `limit` capped at 10 | Not a dedicated search; `track.getInfo` by artist+track | Spotify for catalog resolution |
| Artist metadata | Available (genres, popularity, images) | Available (bio, tags, listeners, playcount) | Complementary — Spotify for identity, Last.fm for context |
| Track metadata | Available (title, album, duration, etc.) | Available (tags, playcount) | Spotify for identity, Last.fm for context |
| Similar artists | **Deprecated / restricted** (new apps: no) | **Available**, no auth, match score 0–1 | Last.fm is the only current source |
| Similar tracks | **Removed with Recommendations** | **Available**, no auth, match % | Last.fm is the only current source |
| Tags | Not exposed as a concept | **Available** (`tag.*` family) | Last.fm-only; folksonomy, not objective |
| Popularity/listening signals | `popularity` field (partially removed/restored across 2026 changes) | `listeners`, `playcount` | Both, but neither is an editorial signal |
| Audio Features | **Removed for new apps** | Not offered by Last.fm either | **Nobody provides this now** |
| Audio Analysis | **Removed for new apps** | Not offered | **Nobody provides this now** |
| Recommendations | **Deprecated / restricted** | No direct equivalent (similarity ≠ recommendation engine) | Neither is a drop-in discovery engine |
| Playlist creation | **Available** (`POST /me/playlists`) | Not applicable — Last.fm has no playlist/publishing concept | Spotify only |
| Playlist management | **Available** (`/items` endpoints) | Not applicable | Spotify only |
| Emerging artist discovery | No dedicated concept; `tag:hipster`/`tag:new` are weak popularity-inverse proxies | `listeners`/`playcount` low-end + tag browsing, still just popularity proxies | Neither source defines "emerging"; requires editorial research either way |
| Musical bridge signals | **None available** (Audio Features/Analysis gone) | Similarity + tags (cultural/contextual, not acoustic) | Editorial judgment remains the primary bridge mechanism |

---

## 5. Sound Continuum Data Strategy

### A. What should Spotify be responsible for?

- Catalog identity: resolving artist/track names to canonical Spotify IDs
  and URIs.
- Track/artist lookup for display (title, artist, album art, duration,
  `preview_url` where present).
- Playlist creation and management — publishing the finished weekly chapter.

Confirmed by research: search, artist/track lookup, and playlist
creation/management endpoints are all currently available. Nothing else
(recommendations, related artists, audio features) is confirmed available
for a new app, so nothing else should be assumed.

### B. What could Last.fm be responsible for?

- Similar artists / similar tracks (confirmed available, no auth).
- Tags as contextual/cultural signal (confirmed available, no auth).
- Listening/popularity context (`listeners`, `playcount`) as one input among
  several — never the deciding factor.

### C. What data is still missing?

Confirmed missing from **both** APIs, not just assumed:

- Tempo, energy, valence, key, mode, danceability, acousticness,
  instrumentalness, liveness, speechiness, loudness — no source currently
  provides these to a new application.
- Production characteristics (mix, texture, arrangement) — never offered by
  either API historically; still absent.
- A defined "emerging artist" signal — neither API has this concept; both
  only offer popularity-adjacent proxies.

### D. Can we still build musical bridges?

**Yes, but not the way the project's early technical imagination assumed.**
The manifesto's bridge dimensions (rhythm, tempo, melody, instrumentation,
texture, vocals, energy, production, mood, cultural context, genre
relationships) map to sources as follows:

1. **Spotify**: genre tags on artists, release-date/era context, and basic
   catalog facts (album, duration). Nothing about rhythm, tempo, or texture.
2. **Last.fm**: cultural/contextual signal via tags and artist/track
   similarity — useful for "these are perceived as related by listeners,"
   not for "these share a tempo or key."
3. **Other metadata**: none currently in scope; no third source identified
   by this research.
4. **Editorial reasoning**: this is where rhythm, tempo, melody,
   instrumentation, texture, vocals, energy, production, and mood-based
   bridging must live now — a human curator listening and judging, not an
   API call. This is consistent with the manifesto's own position (bridges
   are "the most important editorial craft," principle 4) — the API
   landscape change doesn't contradict the manifesto, it just removes a
   technical shortcut that was never guaranteed and was never supposed to
   replace editorial judgment in the first place.

### E. Emerging Artist Discovery

Neither API defines "emerging." Available signals, and what they actually
mean:

- Spotify `tag:hipster` (lowest popularity within a genre) and `tag:new`
  (released in the last two weeks): **low popularity ≠ emerging**, and
  **recently released ≠ emerging** — the card's own instruction not to
  conflate these with "emerging" is directly borne out by what these filters
  literally measure.
- Last.fm `listeners`/`playcount` low end: same caveat — low scrobble count
  measures obscurity on one platform, not artistic emergence, career stage,
  or momentum.
- **What's genuinely missing**: nothing in either API captures "artist is
  early in their career and rising" vs. "artist is niche and has been niche
  for a decade." That distinction requires editorial research — checking
  release history, following context outside these two APIs (news, other
  platforms, word of mouth) — which is out of scope for this card and
  remains a human/editorial task for M6/M7-era workflow, not something M4's
  discovery engine can automate from Spotify or Last.fm data alone.

---

## 6. Musical Bridge Implications

Confirms [Section 5D](#d-can-we-still-build-musical-bridges) at the
architecture level: any M5 design that assumes an Audio-Features-style
numeric bridge score is building on a foundation that doesn't exist for this
app. The only automatable bridge *inputs* available are:

- Last.fm artist/track similarity scores (listening co-occurrence)
- Last.fm/Spotify tags and genres (folksonomy/taxonomy, not acoustic)
- Spotify catalog facts (era/release date, album grouping)

None of these are acoustic-property bridges (tempo/key/energy matching).
**M5 cannot be "compute a bridge score from API data" — it has to be
structured as editorial-first, optionally informed by similarity/tag
signals as supporting context**, not the reverse.

---

## 7. Emerging Artist Discovery

(Full reasoning in [Section 5E](#e-emerging-artist-discovery).) Summary: both
APIs only offer popularity-inverse proxies. Genuine emerging-artist
identification remains editorial research; the APIs can at most narrow a
candidate list (e.g., "here are similar artists with low Last.fm playcount")
for a human to then evaluate — they cannot make the emerging-vs-niche call
themselves.

---

## 8. Rate Limits & API Constraints

- **Spotify**: rolling 30-second window, `429` + `Retry-After` on breach;
  Development Mode ceiling is well above what a weekly, human-driven
  curation workflow generates. No custom rate-limiting infrastructure
  justified for the MVP — simple backoff on `429` is enough.
- **Last.fm**: ToS cites a "100 MB" reasonable-usage cap (period
  unspecified — see Open Questions) and expects response-header-driven
  caching. A weekly batch of similarity/tag lookups for a curated set of
  tracks is trivially small next to either limit.
- Neither service's constraints justify a queue, cache layer (beyond
  respecting HTTP cache headers), or rate-limiter framework at MVP scale —
  consistent with the project's existing "avoid Redis/complexity unless a
  concrete need appears" decision.

---

## 9. Terms & Policy Constraints

- **Spotify**: standard developer terms apply; the Audio Features
  deprecation notice cited security rationale, not a terms change per se.
  One explicit AI-relevant restriction found on the Audio Features reference
  page: **"Spotify content may not be used to train machine learning or AI
  models."** Relevant if Sound Continuum ever considers any ML-assisted
  workflow using Spotify-sourced data — that would need separate legal
  review regardless of the MVP-constraints decision already in place against
  AI-driven editorial decisions.
- **Last.fm**: mandatory attribution (link + "powered by AudioScrobbler"
  button + per-entity `last.fm/music/...` links) whenever Last.fm Data is
  displayed; non-commercial use permitted by default; commercial use needs a
  separate written agreement; no redistribution/sub-licensing of the data;
  audio itself excluded from the agreement (metadata only).
- Both sets of terms are quoted/linked above as fetched; no legal
  interpretation beyond the literal text is offered here, per the card's
  instruction.

---

## 10. Architectural Implications

**M3 — Spotify Integration**: Should be scoped around what's actually
available now — Authorization Code (+PKCE if a browser client is involved)
for curator auth, Client Credentials for anonymous catalog lookups, Search,
Artist/Track metadata, and Playlist create/manage via `/me/playlists` and
`/items`. Should explicitly *not* assume Related Artists, Recommendations,
Audio Features/Analysis, or Extended Quota Mode will ever be available.

**M4 — Discovery Engine**: Discovery cannot be Spotify-alone in the way
originally implied (no Related Artists/Recommendations). Last.fm similarity
and tags would materially improve it — they're the only currently-available
"similar artist/track" signal from either API. Open question: how much
editorial value Last.fm's listening-co-occurrence similarity actually adds
vs. a human curator's own knowledge — that's a design question for M4, not
answered by this research.

**M5 — Musical Ranking & Bridges**: Cannot be built on Spotify Audio
Features — that door is closed for a new app. Last.fm similarity/tags can
supply *supporting* signals (cultural/contextual adjacency), but tempo/key/
energy/production/texture-based bridging has no API source at all and
remains fully editorial. M5's design should treat any API signal as an aid
to the curator, never as the bridge-scoring mechanism itself.

---

## 11. Recommended M3 Direction

Smallest useful M3, justified by this research:

1. Spotify app registration (Development Mode; owner needs Premium).
2. Client Credentials flow for anonymous catalog lookups (search, artist/
   track metadata) — no user auth needed for this half.
3. Authorization Code (+ PKCE if the curator UI is browser-based) for the
   curator's own account, scoped to `playlist-modify-public`,
   `playlist-modify-private`, `playlist-read-private`.
4. A minimal Spotify client wrapping: search, artist/track lookup, create
   playlist (`/me/playlists`), and playlist item management (`/items`
   endpoints).
5. Explicit non-goals for M3: no Related Artists, no Recommendations, no
   Audio Features/Analysis calls (they'll 403 anyway), no assumption of
   Extended Quota Mode.

**Last.fm: research further before M4, do not include in M3.** M3's job is
catalog + publishing, which is Spotify-only; Last.fm's value is entirely on
the discovery/similarity side, which is M4's problem. Pulling it into M3
would mix concerns the card itself separates (Part 5's provider model) and
add scope M3 doesn't need. The provider-model split proposed in Part 5 of
the card — Spotify for catalog/identity/publishing, Last.fm for
relationships/tags/discovery, Sound Continuum for editorial reasoning — **is
appropriate as a conceptual boundary** and matches what this research found
each API can and can't do. It should inform M3/M4 scoping decisions, but per
the card's explicit scope protection, no interfaces or provider
abstractions should be created in code yet — that's premature until M4
actually needs to call Last.fm.

---

## 12. Open Questions

- **Create Playlist for user**: does `POST /users/{user_id}/playlists`
  still function, or was it fully removed in February 2026? The current
  reference page and the migration guide disagree (see
  [Section 2.7](#27-playlists)). Verify with a live test call before M3
  implementation.
- **Playlist item visibility restriction**: does `GET /playlists/{id}/items`
  actually restrict results to owned/collaborated playlists as the Feb 2026
  migration guide states, or does the scope-based model (`playlist-read-private`)
  allow broader access? The two docs pages fetched this session don't fully
  agree; verify against a live call.
- **Last.fm "100 MB" reasonable usage cap**: unit of time (per day? per key,
  ever?) is not stated in the ToS text fetched. Needs clarification directly
  from Last.fm or a support thread before relying on it for capacity
  planning.
- **Last.fm rate limit ("5 req/sec over 5 min")**: this figure came from a
  community/support source, not the primary ToS page fetched in this
  session — worth confirming directly against `last.fm/api/tos` at
  implementation time.
- **Extended Quota Mode realism**: given the 250k-MAU organizational
  requirement, is there any scenario where Sound Continuum would ever
  qualify, or should the project permanently design for Development Mode
  limits? Not a research question so much as a product-strategy one —
  flagged here rather than decided.
- **`external_ids` volatility**: Spotify reverted a February 2026 field
  removal in March 2026. This suggests the changelog is not fully stable
  even month-to-month — worth re-verifying field availability directly
  against the live API immediately before M3 implementation, not relying
  solely on this document by then.
- **Does Last.fm's similarity data add editorial value over a curator's own
  judgment?** Not answerable from API docs alone — would need actual use
  during M4 design/prototyping.

---

## 13. Sources

- [Authorization | Spotify for Developers][spotify-auth]
- [Access Token | Spotify for Developers][spotify-access-token]
- [Introducing refresh token expiration | Spotify for Developers][spotify-refresh-expiry]
- [Quota Modes | Spotify for Developers][spotify-quota-modes]
- [Rate Limits | Spotify for Developers][spotify-rate-limits]
- [Search | Spotify for Developers][spotify-search]
- [Get Playlist's Tracks (deprecated) | Spotify for Developers][spotify-playlist-tracks-deprecated]
- [Create Playlist | Spotify for Developers][spotify-create-playlist]
- [Get Track's Audio Features | Spotify for Developers][spotify-audio-features]
- [Get Recommendations | Spotify for Developers][spotify-recommendations]
- [Get Artist's Related Artists | Spotify for Developers][spotify-related-artists]
- [Introducing some changes to our Web API (Nov 27, 2024) | Spotify for Developers][spotify-nov2024]
- [Web API Changelog — February 2026 (migration guide) | Spotify for Developers][spotify-feb2026]
- [Web API Changelog — March 2026 | Spotify for Developers][spotify-mar2026]
- [Web API Changelog — May 2026 | Spotify for Developers][spotify-may2026]
- [Web API Changelog — July 2026 | Spotify for Developers][spotify-jul2026]
- [Last.fm API Introduction][lastfm-intro]
- [Last.fm API Terms of Service][lastfm-tos]
- [artist.getInfo | Last.fm API][lastfm-artist-getinfo]
- [artist.getSimilar | Last.fm API][lastfm-artist-getsimilar]
- [track.getSimilar | Last.fm API][lastfm-track-getsimilar]
- [tag.getSimilar | Last.fm API][lastfm-tag-getsimilar]

[spotify-auth]: https://developer.spotify.com/documentation/web-api/concepts/authorization
[spotify-access-token]: https://developer.spotify.com/documentation/web-api/concepts/access-token
[spotify-refresh-expiry]: https://developer.spotify.com/blog/2026-06-18-refresh-token-expiration
[spotify-quota-modes]: https://developer.spotify.com/documentation/web-api/concepts/quota-modes
[spotify-rate-limits]: https://developer.spotify.com/documentation/web-api/concepts/rate-limits
[spotify-search]: https://developer.spotify.com/documentation/web-api/reference/search
[spotify-playlist-tracks-deprecated]: https://developer.spotify.com/documentation/web-api/reference/get-playlists-tracks
[spotify-create-playlist]: https://developer.spotify.com/documentation/web-api/reference/create-playlist
[spotify-audio-features]: https://developer.spotify.com/documentation/web-api/reference/get-audio-features
[spotify-recommendations]: https://developer.spotify.com/documentation/web-api/reference/get-recommendations
[spotify-related-artists]: https://developer.spotify.com/documentation/web-api/reference/get-an-artists-related-artists
[spotify-nov2024]: https://developer.spotify.com/blog/2024-11-27-changes-to-the-web-api
[spotify-feb2026]: https://developer.spotify.com/documentation/web-api/tutorials/february-2026-migration-guide
[spotify-mar2026]: https://developer.spotify.com/documentation/web-api/references/changes/march-2026
[spotify-may2026]: https://developer.spotify.com/documentation/web-api/references/changes/may-2026
[spotify-jul2026]: https://developer.spotify.com/documentation/web-api/references/changes/july-2026
[lastfm-intro]: https://www.last.fm/api/intro
[lastfm-tos]: https://www.last.fm/api/tos
[lastfm-artist-getinfo]: https://www.last.fm/api/show/artist.getInfo
[lastfm-artist-getsimilar]: https://www.last.fm/api/show/artist.getSimilar
[lastfm-track-getsimilar]: https://www.last.fm/api/show/track.getSimilar
[lastfm-tag-getsimilar]: https://www.last.fm/api/show/tag.getSimilar
