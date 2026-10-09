# Weekly Curation Workflow

Card #62. This document defines the end-to-end weekly editorial lifecycle
that the M3-M6 cards built piecemeal, and states plainly which parts of it
exist in the codebase today versus which are intended future work. It does
not introduce or change any application behavior.

## Purpose

Per the [manifesto](manifesto.md), Sound Continuum is not a sequence of
disconnected playlists: music is a continuum (§1) and each weekly edition is
a chapter in that continuum, carrying forward what came before and setting
up what comes next (§2). This document gives that idea a concrete process —
the stages a curator moves through each week, what each stage needs, what a
curator does there, and what has to be true before moving to the next one.

It is a **process definition**, not a product spec for unbuilt features.
Where a stage is not implemented, this document says so and describes the
*intended* behavior only — it does not claim the behavior exists.

## The lifecycle

```
Generate pool → Review → Build journey → Compare → Plan next →
Confirm final playlist → Publish → Archive
```

Plan Next is a side activity, not a strict sequencing step: it can happen
before or after Publish, as long as it never mutates the current edition's
confirmed selection or order (see [Plan next](#5-plan-next)).

Confirm final playlist is the fixed boundary between editable curation and
publishing, established by Card #61. Everything before it is freely
revisable; nothing after it may change the confirmed track list or order.

---

## 1. Generate pool

**Why it exists:** Produces the raw material for editorial judgment. Per
the manifesto, discovery is a goal, but it is never an automatic selection
mechanism — this stage only surfaces candidates.

**Input:** Sound Continuum's fixed reference-artist lists (Past, Present,
Emerging) and the official Spotify playlist's own track history.

**Curator action:** None yet — this stage runs automatically on request. The
curator's first real action happens at Review.

**Output:** A pool of enriched, deduplicated candidates split into eligible
and recently-used, each carrying discovery provenance.

**Transition condition:** The pool (even if partially degraded) is
available for review.

**Status: Implemented.**
`discovery.Service.DiscoverPool` (`backend/internal/discovery/pool.go`) runs
the three discovery workflows — Classic (Past), Current (Present), Emerging
(Last.fm-seeded) — merges them, and deduplicates by Spotify track ID.
`review.Service.ReviewPool` (`backend/internal/review/review.go`) then
applies the Recent Track Filter (a hard 28-day gate against the official
playlist's `added_at` history) and metadata enrichment. Exposed via `POST
/api/candidates/pool` and `GET /api/candidates/review`.

Known limitation: Spotify Development Mode rate-limits `GET
/artists/{id}/albums`, which can degrade or empty a discovery run. This is
never silent — failures are recorded on `Result.Failures` /
`CandidatePool.WorkflowErrors` and the frontend shows a distinct "degraded"
state (Card #126) rather than an indistinguishable empty pool.
`SPOTIFY_MOCK_MODE` exists so this stage (and everything downstream) can be
exercised offline during development.

**Connects to:** Review — every eligible candidate from this stage is what
the curator sees next.

---

## 2. Review

**Why it exists:** This is where editorial judgment actually happens.
Scores and discovery rankings are decision support, never a replacement for
human judgment (manifesto §6) — nothing in this stage can select a track by
itself.

**Input:** The eligible candidate pool from stage 1, each with metadata
(title, artists, album, artwork), discovery provenance, and whatever scoring
factors have real data behind them today.

**Curator action:** Inspect each candidate's metadata, score, explanation,
and provenance; Keep, Maybe, or Skip it.

**Output:** Every candidate carries an explicit decision, persisted
independently of any single pool-generation run.

**Transition condition:** At least one candidate is Kept before Build
Journey becomes meaningful (an empty Kept set produces an empty playlist
preview).

**Status: Implemented**, with a real caveat on scoring completeness.
`GET /api/candidates/review` returns ranked, scored, explained candidates.
Of the six scoring factors defined in
[`docs/scoring-model.md`](scoring-model.md), only **Freshness** and
**Repetition Penalty** are wired with real production inputs
(`review.go`'s pipeline). Fit, Discovery Bonus, Diversity, and Playlist Fit
are fully implemented and tested (`backend/internal/scoring/`, M5) but stay
`nil` in production, because their inputs — per-candidate
`musicaldna.Profile` tags, a `CurrentEditionContext` for the edition being
assembled, an explicit editorial discovery-value — are not collected by any
production workflow yet. The UI reflects this honestly (a "Partial · N%
signal" caption, never a fabricated full score).

Keep/Maybe/Skip (Cards #56-#58) are mutually exclusive, idempotent, and
persisted in SQLite (`candidate_selection` table) via `POST
/api/candidates/{id}/{keep,maybe,skip,clear}` — a decision survives a page
reload and a fresh pool-generation run.

**Connects to:** Build Journey — only Kept candidates appear there.

---

## 3. Build journey

**Why it exists:** This is the project's central editorial craft (manifesto
§4): the connections between tracks matter more than any single track's
individual merit. A rigid formula or a score-descending order is explicitly
not sufficient — the curator must be able to prioritize a meaningful
transition over score or discovery rank.

**Input:** The set of Kept candidates from Review, initially in
`scoring.Rank`'s score-descending order.

**Curator action:** Reorder tracks manually to build mood, energy, and
stylistic transitions; consider the playlist's overall arc (opening,
development, ending).

**Output:** A manually ordered sequence of Kept tracks.

**Transition condition:** The curator is satisfied with the sequence and
moves to Confirm.

**Status: Implemented**, frontend-only, in-session.
`WeeklyPlaylistPreview.vue` (Card #59) lists Kept candidates in rank order;
Card #60 adds manual reordering via native HTML5 drag-and-drop plus
Up/Down keyboard controls, fully independent of score. Order is **not**
persisted — a page reload reverts to rank order. Reordering never touches
`CandidateTrack.Status`; selection (Review) and ordering (Build Journey)
stay independent concepts, as designed.

**Connects to:** Compare — the curator should look at this sequence against
the project's recent history before locking it in, though today that
comparison is manual (see next stage).

---

## 4. Compare

**Why it exists:** Per the manifesto, each edition should be a distinctive
chapter, not a repeat of the last one — but repetition is never absolutely
forbidden (§5, §7: "think ahead" implies looking back too). This stage
exists to give the curator that context before confirming.

**Input (intended):** The current proposed sequence, the previous
edition(s), and the project's recent track/artist history.

**Curator action (intended):** Identify unnecessary repetition, understand
continuity, spot recurring artists worth attention, judge whether the
edition is distinctive — while preserving deliberate exceptions, since
there is no blanket "never repeat an artist" rule.

**Output (intended):** An editorially reviewed sequence, with any
repetition or continuity concerns consciously accepted or addressed.

**Status: Partially implemented — and not at the "edition" level this
stage describes.** There is no `Edition` concept anywhere in this codebase:
no edition identifier, no edition boundary, no stored record of past
editions to compare against. What exists instead are two mechanisms, both
driven directly off the live Spotify playlist's own `added_at` history
rather than any notion of discrete editions:

- The **Recent Track Filter** (Card #37) — a hard 28-day gate, already
  applied during Generate Pool, that keeps a recently-used track out of the
  candidate pool entirely.
- The **Repetition Penalty** scoring factor (Card #45) — a soft,
  per-candidate signal (track and artist decay curves over a 90-day
  horizon, `max`-combined) visible in Review's score breakdown, never an
  automatic rejection.

Both already satisfy the manifesto's "no blanket exclusion" requirement by
construction. What's genuinely missing: there is no curator-facing view
that compares the current proposed sequence against a *specific previous
edition*, and no way to say "this was last week's edition" at all, since
editions aren't a persisted concept. A curator today does this comparison
manually, by memory or by looking at the live Spotify playlist.

**Connects to:** Plan Next and Confirm — whatever continuity judgment the
curator makes here is currently undocumented by the system; it lives only
in the curator's head.

---

## 5. Plan next

**Why it exists:** Per the manifesto (§7), editorial planning should
consider not just the current chapter but one or two editions ahead.

**Input (intended):** The current edition's candidates — both selected and
not — including promising discoveries that didn't make the cut, musical
bridges that were only partially developed, and genres/regions/artists
worth exploring further.

**Curator action (intended):** Jot down lightweight directions for the next
edition — artists to revisit, bridges to develop, a point of contrast or
continuity to aim for.

**Output (intended):** A small set of editorial notes/directions, not a
pre-approved selection.

**Transition condition (intended):** None — this is a non-blocking side
activity. Per the lifecycle note above, it must never mutate the current
confirmed edition, and it is not a prerequisite for Confirm or Publish.

**Status: Not implemented.** No mechanism exists anywhere in the codebase —
frontend or backend — for recording or surfacing next-edition notes.
`musicaldna.WeeklyDirection` (Card #41) is the closest structural fit (it
already models "an edition's explicit direction" as `EditionID` + `Profile`
+ free-text `Notes`), but it has no UI, no persistence, and no caller; it
exists purely as a scoring-input type today. Building this stage should
stay proportionate: a free-text note or a short structured list is
sufficient, not a planning system.

**Connects to:** Nothing automatically — it's informational for whoever
curates the following edition's Generate Pool / Review stages.

---

## 6. Confirm final playlist

**Why it exists:** Establishes an explicit, human-made boundary between
"still editable" and "about to be published" — per the manifesto, the final
call belongs to a human editor (§6), and that call should be a deliberate
act, not an implicit side effect of reordering.

**Input:** The Build Journey sequence (Kept tracks, in curator-set order).

**Curator action:** Explicitly confirm the sequence is final.

**Output:** A locked selection and order, unavailable for further
reordering until explicitly reopened.

**Transition condition:** At least one Kept track exists. Confirming is
only actionable with ≥1 Kept track.

**Status: Implemented** (Card #61), frontend-only, in-session.
`WeeklyPlaylistPreview.vue`'s `confirmed` ref locks the UI (hides reorder
controls) and exposes `confirmedPlaylist` via `defineExpose` — a computed
view of the locked Kept-tracks-in-order list, `null` unless confirmed. If a
Keep/Maybe/Skip decision changes elsewhere while confirmed (membership or
sequence would change), confirmation auto-invalidates back to editable
rather than silently drifting out of sync. `confirmedPlaylist` is the
integration point a future Publish flow would read via a template ref — **no
such flow exists yet**; nothing in this repo consumes it.

**Connects to:** Publish — `confirmedPlaylist` is the authoritative input
Publish must use.

---

## 7. Publish

**Why it exists:** Delivers the curated edition to the actual Spotify
playlist listeners follow. The confirmed selection and order are
authoritative — publishing must never substitute a score- or
discovery-rank order for the curator's confirmed order.

**Input (intended):** `confirmedPlaylist` from stage 6 — the exact Kept
tracks, in the exact confirmed order.

**Curator action (intended):** Trigger publish once confirmed.

**Output (intended):** A successfully published Spotify playlist matching
the confirmed track list and order exactly, or a clearly surfaced failure
that can be retried or resolved without silently falling back to a
different ordering.

**Preconditions (intended, per this card's own requirements):**
- The playlist contains at least one Kept track.
- The curator has explicitly confirmed the final playlist (stage 6).
- The published order exactly matches the confirmed order.

**Status: Not implemented.** The only Spotify write endpoint in this repo
is `POST /api/spotify/playlist` (`Service.InitializeOfficialPlaylist`,
`backend/internal/spotify/handlers.go`), which creates the official,
empty playlist once and is idempotent — it has no notion of tracks at all.
There is no add-tracks-to-playlist call anywhere in the codebase (Spotify's
own `POST /playlists/{id}/tracks` is unused), no publish button in the
frontend, and no code path connecting `confirmedPlaylist` to Spotify.

Building this stage should reuse the existing M3 integration (the
`spotify.Client`/`spotify.Service` layering and its typed `APIError`
taxonomy — see [`decisions.md`](memory/decisions.md)) rather than
duplicating it, and should decide upfront whether a failed publish attempt
is retried against the same confirmed state or requires the curator to
re-confirm.

Operational note for whoever builds this: the official playlist has been
deleted outside the app multiple times in this project's history (Cards
#30/#37/#39/#126), each requiring manual reconciliation. Publish will
depend on that playlist's continued existence on Spotify's side — this
isn't something to fix here, but it's a real failure mode to design for.

**Connects to:** Archive — only a successful publish should ever be
archived.

---

## 8. Archive

**Why it exists:** Preserves what was actually published, so future
Compare and Plan Next stages have real history to work from instead of
relying on the curator's memory or the live Spotify playlist alone.

**Input (intended):** The outcome of a successful Publish — not an earlier
editable preview, and never a failed attempt.

**Curator action (intended):** None — archiving should be an automatic
consequence of a successful publish.

**Output (intended):** A historical record of the edition: identifier or
publication date, final track selection, exact published order, Spotify
playlist/URL reference, publication status, and editorial notes if
available.

**Status: Not implemented.** SQLite currently holds exactly three tables:
`spotify_connection`, `official_playlist` (both singletons), and
`candidate_selection` (per-candidate Keep/Maybe/Skip, not edition-scoped).
Nothing records a published edition's track list, order, or outcome.
`confirmedPlaylist` exists only in frontend memory for the browser
session — it does not survive a page reload, let alone a publish.

A failed publication must never be recorded as if it succeeded — this
follows directly from Archive only ever representing a successful Publish
outcome, once Publish itself exists.

**Connects to:** Compare and Plan Next for the *next* edition — this is the
missing piece that would let stage 4 compare against a real previous
edition instead of raw playlist history.

---

## Current edition → next edition relationship

Today, "continuity between editions" is carried entirely by the live
Spotify playlist's own history (via the Recent Track Filter and Repetition
Penalty) and by the curator's own memory — there is no `Edition` entity
connecting one week's confirmed playlist to the next week's Generate Pool
run. Plan Next (stage 5), once built, is meant to be the lightweight bridge
between editions: notes and directions a curator leaves for themselves,
consumed manually at the start of the next Generate Pool / Review cycle. It
must never mutate the current edition's confirmed selection or order,
regardless of when it happens relative to Publish.

## Known gaps

| Stage | Gap | What a future card needs to decide first |
|---|---|---|
| Compare | No `Edition` entity; comparison is against raw playlist history, not a discrete previous edition | Whether an Edition needs to be modeled at all, or whether richer playlist-history queries are enough |
| Plan next | Nothing built | Minimal UI/storage shape (free text vs. `WeeklyDirection`-style structured notes) |
| Publish | No track-write path to Spotify | Reuse `spotify.Client`/`APIError` layering; retry semantics on partial failure |
| Archive | No persistence of published editions | Depends on Publish existing first; storage shape listed in stage 8 above |

Publish and Archive are sequentially dependent (Archive needs a real
Publish to record). Compare's gap is independent and could be addressed
before or after either, since it doesn't require a write path to Spotify.
