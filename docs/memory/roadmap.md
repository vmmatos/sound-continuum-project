# Roadmap

High-level direction, not a contract. Do not treat this as a detailed spec
of future implementation requirements.

- **M1 — Define Sound Continuum** — Completed
- **M2 — Project Foundation** — In progress
- **M3 — Spotify Integration** — In progress (OAuth/token lifecycle and a
  read-only API client — profile, playlists (list + single), playlist
  items with track/episode/unavailable discrimination, single-track
  retrieval, single-artist retrieval, search — are implemented; playlist
  creation/management and Sound Continuum playlist discovery still open)
- **M4 — Discovery Engine** — Feature-complete (the `CandidateTrack`
  domain model is defined; all three planned discovery workflows are
  implemented: classic music discovery from the Past reference artist
  set, current music discovery from the Present reference artist set, and
  emerging artist discovery via a single Last.fm `artist.getsimilar` hop
  from the Emerging reference artist set, resolved through Spotify — all
  producing candidate pools, not a ranking — with
  `POST /api/discovery/classic`, `POST /api/discovery/current`, and
  `POST /api/discovery/emerging`. A Candidate Pool orchestrates all three
  into one merged, deduplicated result, and a Recent Track Filter sits
  immediately after it, splitting the pool into eligible and
  recently-used candidates against the official Spotify playlist's own
  `added_at` history (28-day default lookback,
  `RECENT_TRACK_LOOKBACK_DAYS`-configurable) — a temporary editorial
  guardrail, not a permanent blacklist. Every eligible candidate is then
  enriched with structured Spotify metadata (title, artists, album,
  release date + precision, duration, explicit flag, artwork, Spotify
  URL/URI) and carries structured discovery provenance (which discovery
  method, which seed/reference artist, which provider — distinct from the
  provider-oriented `Source` field) that survives pool deduplication,
  filtering, and enrichment — no ranking or scoring is implemented
  anywhere yet. All of this is exposed through `POST
  /api/candidates/pool`; persistence and musical bridge logic are open
  for M5+)
- **M5 — Musical Ranking & Bridges** — In progress (the Candidate Scoring
  Model is defined: six factors, initial weights, and the combination
  formula; the Fit factor's algorithm is implemented (Card #41,
  `musicaldna` package + `scoring.CalculateFit`), the Freshness factor's
  algorithm is implemented (Card #42, `scoring.CalculateFreshness`,
  reusing Card #37's playlist-history retrieval), and the Discovery Bonus
  factor's algorithm is implemented (Card #43,
  `scoring.CalculateDiscoveryBonus` — eligible only for `CategoryEmerging`
  candidates, and only when an explicit editorial discovery value is
  supplied; never derived from discovery provenance, Last.fm similarity,
  or popularity), the Diversity factor's algorithm is implemented
  (Card #44, `scoring.CalculateDiversity` — Artist/Era/Sound concentration
  evaluated against an explicit, transient `CurrentEditionContext`, never
  the official playlist or candidate pool), the Repetition Penalty
  factor's algorithm is implemented (Card #45,
  `scoring.CalculateRepetitionPenalty` — track and artist repetition
  against official playlist history, combined by `max`, a soft signal
  distinct from the Card #37 hard Recent Track Filter), and the Playlist
  Fit factor's algorithm is implemented (Card #46,
  `scoring.CalculatePlaylistFit` — a sequence-aware transition score
  between the current edition's immediately-preceding track and the
  candidate, using Mood/Energy/Texture/Cultural Influence with a
  match-or-baseline and ordinal-energy scoring model, independent of
  every other factor) — see
  [`docs/scoring-model.md`](../scoring-model.md); all six scoring factors
  are now implemented, M5 now only has ranking and automatic selection
  remaining open)
- **M6 — Curator Experience** — Planned
- **M7 — Weekly Editorial Workflow** — Planned
- **M8 — Feedback & Evolution** — Planned
