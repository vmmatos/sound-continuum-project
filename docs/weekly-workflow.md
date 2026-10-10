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

**Curator action:** Click **Generate candidate pool** on the Candidate
Review page (Card #63). Discovery never runs on page load; the curator
starts each run explicitly, and can regenerate at any time.

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

Refresh semantics (Card #63): the pool is not persisted, so a page reload
shows an empty Candidate Review until the curator generates again; Keep/
Maybe/Skip decisions are persisted and re-attach by Spotify track ID on
every run. Regenerating keeps on screen any Kept candidate the new run
didn't rediscover (so Kept tracks, their manual order and a
confirmation are never silently dropped); a failed run leaves the current
pool untouched. Generation never creates or modifies an Edition.

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
reload and a fresh pool-generation run, and is never applied to another
candidate (Card #64).

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

**Status: Partially implemented — and not yet at the "edition" level this
stage describes.** An `Edition` domain model and persistence now exist
(`backend/internal/edition`, Card #139) — each confirmed weekly playlist is
a discrete, identified, persisted record. But nothing queries *past*
editions for comparison yet: there is no "previous edition" lookup, no
archive listing, and no UI surfacing edition history. What exists instead
are two older mechanisms, both driven directly off the live Spotify
playlist's own `added_at` history rather than any notion of discrete
editions:

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

**Status: Implemented** (Card #61, persisted by Card #139).
`WeeklyPlaylistPreview.vue`'s `confirmed` ref locks the UI (hides reorder
controls) and exposes `confirmedPlaylist` via `defineExpose` — a computed
view of the locked Kept-tracks-in-order list, `null` unless confirmed. If a
Keep/Maybe/Skip decision changes elsewhere while confirmed (membership or
sequence would change), confirmation auto-invalidates back to editable
rather than silently drifting out of sync.

Confirming now also persists: `confirmPlaylist()` POSTs the confirmed
entries to `POST /api/editions/confirm`
(`backend/internal/edition.Service.ConfirmFromReview`) and only locks the UI
once that call succeeds — a failure shows an inline error and leaves the
playlist editable, never a false "confirmed" state. The backend persists
the exact ordered snapshot as the active `Edition`'s confirmed track list in
a new `editions` SQLite table (see
[`docs/memory/decisions.md`](memory/decisions.md) for the full model,
lifecycle, and single-active-edition invariant). `confirmedPlaylist`
remains the frontend's own in-session view (lost on reload, as before); the
durable record now lives in the backend `Edition` row, independently of the
browser tab.

**Connects to:** Publish — the confirmed `Edition` row (`ConfirmedTracks`,
in exact order) is the authoritative input Publish must use.

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
frontend, and no code path connecting the confirmed `Edition` to Spotify.

The model and persistence foundation this stage needs now exists (Card
#139): `Edition.Status` transitions `Confirmed → Publishing → Published`
(or back to `Confirmed` on a recoverable failure), and
`Store.StartPublishing`/`RecordPublishSuccess`/`RecordPublishFailure`
already implement and test those transitions — but none of them are wired
to any HTTP route or actual Spotify call yet; they exist only as tested Go
methods with no caller, the same "implemented, not wired" state M5's
scoring factors were in before their production inputs existed. Whoever
builds this should reuse the existing M3 integration (the
`spotify.Client`/`spotify.Service` layering and its typed `APIError`
taxonomy — see [`decisions.md`](memory/decisions.md)) rather than
duplicating it, call `StartPublishing`/`RecordPublishSuccess`/
`RecordPublishFailure` around that call, and persist `SpotifyPlaylistID`/
`SpotifyPlaylistURL` as soon as they're known so a retry can reconcile
against the official playlist's actual remote state instead of blindly
appending duplicate tracks (see Step 7's failure scenario in the card).

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

**Status: Not implemented** — but the model it will rest on now exists.
SQLite holds a fourth table since Card #139, `editions`, recording each
confirmed edition's track list, order, and (once Publish exists)
publication outcome. `Edition.Status` already has an `Archived` value and
`Store.Archive` already transitions `Published → Archived` recording
`ArchivedAt`, rejecting every other source status — but nothing calls it:
there is no archive endpoint, no automatic trigger on a successful publish,
and no archive-listing UI. Archiving should become an automatic
consequence of `RecordPublishSuccess` once Publish exists, not a curator
action.

A failed publication must never be recorded as if it succeeded — this
follows directly from Archive only ever representing a successful Publish
outcome (`Store.Archive` only accepts `StatusPublished`), once Publish
itself exists.

**Connects to:** Compare and Plan Next for the *next* edition — this is the
missing piece that would let stage 4 compare against a real previous
edition instead of raw playlist history.

---

## Current edition → next edition relationship

An `Edition` entity now exists (Card #139) and enforces at most one active
(non-archived) edition at a time, but nothing yet reads *past* editions —
"continuity between editions" is still carried entirely by the live
Spotify playlist's own history (via the Recent Track Filter and Repetition
Penalty) and by the curator's own memory, not by querying archived
`Edition` rows. Generate Pool's next run does not consult the previous
edition in any way. Plan Next (stage 5), once built, is meant to be the
lightweight bridge between editions: notes and directions a curator leaves
for themselves, consumed manually at the start of the next Generate Pool /
Review cycle. It must never mutate the current edition's confirmed
selection or order, regardless of when it happens relative to Publish.

## Known gaps

| Stage | Gap | What a future card needs to decide first |
|---|---|---|
| Compare | An `Edition` entity exists (Card #139), but nothing queries past editions; comparison is still against raw playlist history | How to look up "the previous edition" (needs Archive to produce history first) and what to show against it |
| Plan next | Nothing built | Minimal UI/storage shape (free text vs. `WeeklyDirection`-style structured notes) |
| Publish | No track-write path to Spotify; `Edition` lifecycle/retry fields exist but are unwired | Reuse `spotify.Client`/`APIError` layering; call `StartPublishing`/`RecordPublishSuccess`/`RecordPublishFailure` around it |
| Archive | `Store.Archive` exists and is tested, but has no caller/endpoint | Depends on Publish existing first; wire it as an automatic consequence of `RecordPublishSuccess` |

Publish and Archive are sequentially dependent (Archive needs a real
Publish to record). Compare's gap is independent and could be addressed
before or after either, since it doesn't require a write path to Spotify.
