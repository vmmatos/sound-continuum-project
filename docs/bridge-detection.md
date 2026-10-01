# Potential Bridge Detection

Card #48, `backend/internal/scoring/bridge.go`.

## What a potential bridge means

The [manifesto](manifesto.md) treats musical bridges — the connections
between tracks — as the heart of Sound Continuum's editorial craft, more
important than any single track's individual merit. `DetectPotentialBridge`
is an evidence-gathering mechanism for editorial review of a candidate
pair (Track A, Track B): it produces a deterministic signal,
`PotentialBridge = true/false`, plus the structured evidence behind it.

It is **not**:

- a similarity score,
- a claim that two tracks are musically similar,
- a ranking or automatic playlist-selection mechanism,
- part of `scoring.Factors`/`scoring.Calculate` — it is never combined into
  `CandidateScore` and carries no weight.

`PotentialBridge: true` means "this pair deserves editorial consideration
as a bridge," never "this pair is a confirmed bridge." The final call
remains human, per the manifesto's community/editorial-judgment principle.

## Signals used

Built on [Card #47](research/musical-similarity.md)'s investigation of what
is actually available today (no Spotify Audio Features, Audio Analysis,
Recommendations, or Related Artists exist for this app — see
[`decisions.md`](memory/decisions.md)), plus the explicit editorial
dimensions Fit (Card #41) and Playlist Fit (Card #46) already established.

**Four explicit dimensions** (`musicaldna.Profile`: Mood, Energy, Texture,
Cultural Influence), compared with the exact comparators Playlist Fit
already defined — reused directly, not duplicated:

- Mood / Texture / Cultural Influence: exact, case-insensitive match = 1.0;
  a known mismatch = a flat 0.5 baseline (contrast is allowed and never
  scored as "wrong"); a missing value on either side is excluded
  (`Available: false`), never scored.
- Energy: the same local five-level ordinal vocabulary (`very low` … `very
  high`), scored `1 - |levelDiff|/4` — same level = 1.0, adjacent = 0.75
  ("progression"), two apart = 0.5, further = below 0.5. An unparseable
  value on either side falls back to the same match-or-baseline rule as the
  categorical dimensions (the established safe default).

**Three contextual signals**, evidence only, never proof on their own:

- **Shared artist** — do both tracks share a Spotify artist ID
  (`BridgeTrack.ArtistSpotifyIDs`)? Identity only, never treated as a
  similarity claim.
- **Last.fm artist similarity** — was a Last.fm `artist.getsimilar` match
  supplied by the caller (`lastFMArtistMatch`) and is it positive?
  `DetectPotentialBridge` makes no Last.fm call itself; the caller supplies
  this from an existing lookup (e.g. `lastfm.Client.SimilarArtists`, or
  `candidate.DiscoveryProvenance.LastFMMatch`). This is listening
  co-occurrence evidence, never a claim of track-level musical similarity.
- **Release-era relationship** — do both tracks fall in the same coarse
  decade, via the existing `scoring.DiversityEra`? Contextual framing only;
  says nothing about musical character.

## What is deliberately excluded

- **Genre overlap.** `DetectPotentialBridge`'s signature has no genre
  parameter at all — genre cannot contribute evidence by construction, not
  by a runtime check. Card #47 found Spotify genres unreliable (artist-level
  only, optional, frequently `null`, no taxonomy), and the card explicitly
  forbids treating genre overlap as sufficient bridge evidence.
- **Spotify Audio Features/Audio Analysis/Recommendations/Related
  Artists** — confirmed unavailable for this app (see `decisions.md`).
- **Last.fm track-level similarity (`track.getSimilar`) and tag overlap
  (`tag.*`)** — Card #47 found these exist at the provider but have no code
  in this repository; building either is new integration surface, not
  reuse, and is out of scope here.
- **Any weighted numeric score.** There is no `Mood × 0.25 + Energy × 0.25 +
  …` formula. The only threshold is `DefaultMinimumBridgeEvidence` (a count
  of independent signals, not a weighted combination).
- **Popularity, followers, streams, listener/playcount counts.**

## Why PotentialBridge ≠ confirmed bridge

No signal available to this codebase today can observe rhythm, melody,
instrumentation, vocal character, or production — the qualities the
manifesto actually lists as what a bridge connects. `DetectPotentialBridge`
can only ever surface *circumstantial* evidence (explicit editorial tags,
shared identity, listening co-occurrence, release era) for a human editor
to weigh. Treating its `true` result as a confirmed bridge would overstate
what the evidence supports — exactly what the AI/automation guardrail in
`decisions.md` ("AI must not become a dependency of the editorial
workflow") exists to prevent.

## The evidence-count rule

Each of the four dimensions and three contextual signals independently
reports whether it counts as a "meaningful relationship": a dimension
counts when it is available and its score is above the 0.5 baseline (an
exact match, or an adjacent-energy progression); a contextual signal counts
when it is available and present. `PotentialBridge` is true when the total
count reaches `DefaultMinimumBridgeEvidence` (2).

This single constant — not a weighted formula — is the whole threshold. The
card explicitly forbids treating any one signal as proof (e.g. Last.fm
similarity alone must never independently force a bridge); requiring at
least two independent, corroborating signals is the smallest rule that
satisfies that constraint. It is deterministic, documented here, and
trivially changed in one place (`backend/internal/scoring/bridge.go`) if a
future card finds the threshold wrong.
