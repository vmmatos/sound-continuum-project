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
- **M4 — Discovery Engine** — In progress (the `CandidateTrack` domain
  model is defined; all three planned discovery workflows are implemented:
  classic music discovery from the Past reference artist set, current
  music discovery from the Present reference artist set, and emerging
  artist discovery via a single Last.fm `artist.getsimilar` hop from the
  Emerging reference artist set, resolved through Spotify — all producing
  candidate pools, not a ranking — with `POST /api/discovery/classic`,
  `POST /api/discovery/current`, and `POST /api/discovery/emerging`;
  persistence and musical bridge logic are still open)
- **M5 — Musical Ranking & Bridges** — Planned
- **M6 — Curator Experience** — Planned
- **M7 — Weekly Editorial Workflow** — Planned
- **M8 — Feedback & Evolution** — Planned
