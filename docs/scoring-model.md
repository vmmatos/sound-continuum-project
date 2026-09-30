# Candidate Scoring Model

Card #40 (M5, "Musical Ranking & Bridges"). Implemented in
`backend/internal/scoring`.

## Purpose

Candidate scoring is an **editorial decision-support mechanism**. It is not
an automatic music selection mechanism.

A higher score means: "this candidate currently has stronger signals
according to the Sound Continuum evaluation model." It does **not** mean
"this is objectively a better song," and it does **not** mean "this track
must be selected." Scoring helps answer "which candidates deserve closer
editorial attention first?" — never "which tracks does the system choose
automatically?" The final playlist remains human-curated, per the
[manifesto](manifesto.md) §6.

## Scoring vs. ranking vs. selection

This card defines scoring only. It does not implement ranking (sorting
candidates by score) or selection (choosing which candidates become
playlist picks). `scoring.Calculate` never touches
`candidate.CandidateTrack.Status`, never sorts a candidate pool, never
publishes to Spotify. Ranking and selection are separate, future concerns
that will *consume* a `CandidateScore` as one input among others — not
concerns this package owns.

## The six factors

Each factor is normalized to `[0,1]`, where `0.0` is the weakest possible
contribution and `1.0` is the strongest. Five are positive contributions;
one is a penalty.

| Factor | Question it answers |
|---|---|
| Fit | Does this candidate belong to the Sound Continuum identity? |
| Freshness | Does this bring timely/recent relevance? |
| Discovery Bonus | Does this offer useful discovery value? |
| Diversity | Does this add meaningful variation to the current context? |
| Playlist Fit | Does this make sense in the current musical journey? |
| Repetition Penalty | Are we over-representing something? (0 = no concern, 1 = strongest concern) |

None of the five positive factors' actual calculation algorithms are
implemented by this card — `scoring.Factors` only holds already-calculated
(or not-yet-calculated) values. Later M5 cards implement one factor at a
time. **Fit's algorithm is implemented as of Card #41, and Freshness's as
of Card #42** — see below; Discovery Bonus, Diversity, Repetition Penalty,
and Playlist Fit remain future cards.

### Fit (Card #41, `backend/internal/scoring/fit.go`)

Fit answers one question: **"How naturally does this candidate belong to
the musical identity and direction Sound Continuum is currently building?"**
It is not a popularity score, a genre-matching score, a recommendation
score, or a playlist-transition score (that's Playlist Fit, below).

**Two contextual layers.** Sound Continuum's musical identity has two
distinct parts, both defined in `backend/internal/musicaldna`:

- **`musicaldna.ProjectDNA`** — the stable, project-wide identity ("what is
  Sound Continuum?"), sourced only from [`docs/manifesto.md`](manifesto.md)
  and Sound Continuum's M1 decisions.
- **`musicaldna.WeeklyDirection`** — the current edition's explicit
  direction ("what is this edition becoming?"), set by a human curator, one
  per edition (`EditionID`). Card #41 represents this at the domain level
  only — no UI, no HTTP endpoint, no persistence; it must always be an
  explicit input, never inferred from whichever candidates a discovery
  workflow happens to produce.

Both are expressed with the same `musicaldna.Profile` type (also used for a
candidate's own tagged characteristics), so there is one dimension
vocabulary, not three competing representations.

**Why `ProjectDNA` is empty today.** The manifesto describes editorial
*process and philosophy* — music as a continuum, musical bridges over
individual merit, discovery without forced obscurity, human editorial
judgment — not concrete mood/energy/texture/cultural values. Setting a
`Profile` field on `ProjectDNA` would assert "Sound Continuum's identity
always has this musical quality," which the manifesto does not claim and
which would contradict its continuum principle (§1). So
`musicaldna.DefaultProjectDNA()` leaves every dimension unset, and Fit is
driven entirely by `WeeklyDirection` until a real, durable, project-wide
trait is documented in the manifesto or `docs/memory/decisions.md` and a
field is set here to match — see the Card #41 entry in
`docs/memory/decisions.md` for the explicit per-dimension mapping and
reasoning.

**Dimensions.** Four, drawn from the qualities this document already named
for Fit before Card #41 (mood, energy, texture, cultural context) rather
than invented independently: `Mood`, `Energy`, `Texture`,
`CulturalInfluence`. Qualities that would require unavailable audio-level
analysis to observe reliably (tempo, detailed rhythmic/melodic structure,
vocal character) are left out — Spotify's Audio Features/Audio Analysis
endpoints are unavailable to this app (see
`docs/memory/decisions.md`'s M3 Audio Features decision), and fabricating
these values from genre strings or nothing at all is explicitly forbidden.
Each dimension is an optional `*string`, supplied only through explicit
editorial input (never derived from `CandidateMetadata`, genre, or
popularity) — nil means "not supplied," never "poor fit."

**Comparison strategy.** Case-insensitive, trimmed exact string match per
dimension. No fuzzy or embedding-based similarity is used — that would
require the AI/ML this project explicitly excludes, and any partial-credit
heuristic between arbitrary strings would be fabricated precision.

**Weights** (`scoring.DefaultFitWeights()`), two independent groups, each
summing to 1.0:

| Weight | Value | Group |
|---|---|---|
| Weekly Direction | 0.75 | component |
| Project DNA | 0.25 | component |
| Mood | 0.35 | dimension |
| Energy | 0.25 | dimension |
| Texture | 0.25 | dimension |
| Cultural Influence | 0.15 | dimension |

Weekly Direction (0.75) far outweighs Project DNA (0.25), per the card's
requirement that the current week's direction carry "greater contextual
importance than a generic project-wide similarity" — and, because
`ProjectDNA` is empty by construction today, this split is realized
structurally, not just by number. Mood is weighted highest among
dimensions as the most direct expression of "does this belong to the story
we're telling right now"; Cultural Influence is weighted lowest since Card
#41 explicitly warns against genre/cultural matching becoming the
definition of Fit.

**Missing dimensions.** Renormalized, at two levels, using the same idiom
`scoring.Calculate` already uses for missing factors: within one comparison
(Project or Weekly), a dimension missing on either side is excluded and the
match ratio is computed over the dimensions actually available; across the
two comparisons, a component with nothing available contributes nothing and
weight is renormalized over whichever component did produce a value. `Fit`
is `nil` — never a fabricated `0.0` — only when neither comparison had
anything to compare. This is deliberate: missing musical-character data
must never read as "poor fit," which would systematically penalize
emerging or less-documented artists.

**No confidence score.** Considered and left out: it would either be a
fake statistical calibration or unnecessary complexity at this stage. Open
question for a future card.

**No editorial override mechanism.** None exists in Card #40 to preserve.
`Factors.Fit` is already a plain settable `*float64`, so a human or future
workflow can already override a Fit value by direct assignment — a low Fit
score never means "candidate rejected," only "the current model sees
weaker fit according to available signals." Building a full override
workflow is out of scope for this card.

**Explainability.** `scoring.FitResult.Dimensions` carries one
`FitDimensionResult` per (dimension, component) pair — 8 total — recording
whether it was available and whether it matched, so a `nil` or low `Fit`
value is always traceable to specific missing or mismatched dimensions,
never an opaque number.

**Independence from the other five factors.** `CalculateFit` takes only
`musicaldna.Profile`/`ProjectDNA`/`WeeklyDirection` values — never a
`candidate.CandidateTrack` — so it cannot see, and cannot be affected by,
`CandidateType`, `Category`, release date (Freshness's domain), discovery
provenance (Discovery Bonus's domain), or playlist sequence (Playlist
Fit's domain). See Card #41's tests in
`backend/internal/scoring/fit_test.go` for explicit regression coverage of
each independence claim.

### Freshness (Card #42, `backend/internal/scoring/freshness.go`)

Freshness answers one question: **"How long has it been since this
candidate last appeared in the official Sound Continuum playlist?"** It is
**playlist-history freshness, not release-date freshness** — a 1985 track
that has never appeared in Sound Continuum has `Freshness = 1.0`, and a
track released yesterday that was already used recently has low Freshness.
`scoring.CalculateFreshness` does not accept a release date, popularity
figure, `CandidateType`, `Category`, or discovery provenance as input, so
none of them can affect Freshness by construction — see
`backend/internal/scoring/freshness_test.go`'s independence test.

**Distinct from Fit.** Fit never reads playlist history, and Freshness
never reads musical character — a very fresh candidate can have poor Fit,
and a perfectly-fitting candidate can be stale. `CalculateFit` takes no
playlist-history input, and `CalculateFreshness` takes no
`musicaldna.Profile` input, by construction.

**Distinct from the Recent Track Filter (Card #37).** `discovery.
FilterRecentTracks` is a **hard eligibility rule**: "was this track used
within the last `RecentTrackLookbackDays` (28) days? If so, exclude it
before scoring." Freshness is a **soft scoring gradient** for candidates
that already passed that filter: "of the tracks that *are* eligible, how
long has it actually been?" A track excluded by the filter never reaches
Freshness at all in the normal pipeline; Freshness exists to distinguish,
among eligible tracks, a 35-day-old appearance from a 180-day-old one —
the filter alone cannot do that, since both are simply "not recent."

**Distinct from Repetition Penalty** (future card): Repetition Penalty is
a scoring *penalty* about broader repetition patterns (e.g. the same
artist or musical territory recurring); Freshness is a *positive* factor
about one specific track's own playlist-appearance recency. The two are
independent signals, not two implementations of the same idea.

**Identity and source of truth.** Freshness uses Spotify track ID as the
identity key and reuses Card #37's playlist-history retrieval — there is
exactly one mechanism that walks the official playlist
(`discovery.Service.PlaylistTrackHistory`, which both `FilterRecentTracks`
and Freshness's wiring build on; see `docs/memory/decisions.md`'s Card #42
entry for why this is exported rather than duplicated). If the same track
appears multiple times in the playlist, the most recent valid `added_at`
is used. Episodes and unavailable/null items are ignored, identically to
Card #37. `scoring.CalculateFreshness` itself never calls Spotify — it is
a pure function; a small glue function, `scoring.FreshnessLastUsedAt(
history map[string]time.Time, spotifyTrackID string) *time.Time`, looks up
one candidate's most recent appearance from the history map
`PlaylistTrackHistory` returns.

**Never-used candidates.** A candidate whose Spotify track ID has never
appeared in the official playlist gets the maximum Freshness, exactly
`1.0` — regardless of how old the underlying track is.

**The curve.** For a used candidate, Freshness follows a half-life
recovery curve:

```
Freshness(t) = 1 − 0.5^(t / HalfLifeDays)
```

where `t` is the number of days since the track's most recent appearance
(`now − lastUsedAt`, both explicit inputs — `now` is always supplied by
the caller, never read from the system clock inside the function, so
Freshness is deterministic for a given candidate/history/evaluation time).
Default `HalfLifeDays = 60` (`scoring.DefaultFreshnessHalfLifeDays`, in
`scoring.FreshnessConfig`):

| Days since last use | Freshness |
|---|---|
| 0 | 0.00 |
| 28 (the #37 filter boundary) | 0.28 |
| 35 | 0.33 |
| 60 (one half-life) | 0.50 |
| 180 | 0.88 |
| 730 (2 years) | 0.9998 |

This curve is continuous and monotonically increasing in `t` — more time
since last use never produces a *lower* Freshness — and is deliberately
asymptotic: a used candidate's Freshness approaches but never reaches
exactly `1.0`, keeping it visibly distinct from a genuinely never-used
candidate's exact `1.0`. Critically, it has **no discontinuity at the
28-day Recent Track Filter boundary**: 27, 28, and 29 days apart differ by
only a few hundredths, not a jump from near-zero to maximum — Freshness is
a gradient *beyond* the hard filter, not a re-implementation of it.

Half-life was chosen over a linear ramp (which needs an arbitrary
saturation horizon with no natural justification) and over stepped buckets
(which would reintroduce exactly the discontinuity the card forbids) as
the simplest curve with one explainable constant. 60 days — roughly double
the 28-day filter window — was chosen so a track just outside the filter
window (29–35 days) reads as clearly low-but-not-zero, while a track
several months old (120–180 days) reads as clearly high, giving the
gradient real spread without an oversized configuration surface. See
`docs/memory/decisions.md`'s Card #42 entry for the full reasoning.

**Clock skew.** If `lastUsedAt` is somehow after `now`, `t` is clamped to
`0` rather than producing a negative duration or an error — defensive
handling, not a new failure mode.

**Missing/unavailable history.** If the official playlist cannot be
retrieved (not configured, authentication failure, API error, network
error), `PlaylistTrackHistory`/`FreshnessLastUsedAt` never fabricate an
empty result — the error propagates, exactly as Card #37 requires, so a
temporary Spotify outage can never silently read as "every candidate is
maximally fresh." A **successfully retrieved but genuinely empty**
official playlist is different: every candidate is legitimately
never-used, so Freshness is `1.0` for all of them.

**Explainability.** `scoring.FreshnessResult` carries `Value`,
`LastUsedAt` (nil for never-used), and `TimeSinceLastUse`, so a low or
maximal Freshness is always traceable to a concrete last-appearance time
rather than an opaque number.

### Discovery Bonus

The editorial value of surfacing something less obvious to the audience —
*useful* discovery potential, not simply low popularity. "Unknown" does
not automatically mean "valuable," and `Category: Emerging` does not
automatically mean a high score. Obscurity is never rewarded for its own
sake, per the manifesto's "discovery without forced obscurity" principle.

Distinct from **Fit**: Discovery Bonus reads discovery provenance (how a
candidate was found); Fit never does. A candidate discovered through an
unusual path can still have strong or weak Fit independently — provenance
is not itself a Fit input (see Card #41 tests).

### Diversity

How much a candidate contributes something different from what's already
represented in the current playlist/candidate context. Not the same as
randomness — Sound Continuum is built on musical bridges, so diversity
should eventually reward variation that stays musically connected.
Diversity is **contextual**: the same candidate can score differently
depending on the current playlist state, so it cannot be calculated from
the candidate in isolation.

### Playlist Fit

Distinct from **Fit**: Fit asks "does this candidate belong to Sound
Continuum's identity at all?" (a general, largely context-independent
question); Playlist Fit asks "does this candidate make sense in the
*current* chapter/journey?" (relationship to recently selected material,
narrative continuity, transition opportunities). A candidate can have high
general Fit and low Playlist Fit in a given edition, or vice versa — the
two factors are independent and neither is derived from the other. No
musical-similarity algorithm, Audio Features, or Spotify Recommendations
are used for this factor, now or in any planned future card (see
`docs/memory/decisions.md`'s M3 Audio Features decision).

### Repetition Penalty vs. the Recent Track Filter (Card #37)

These answer different questions and must not be conflated:

- **Recent Track Filter** (`discovery.FilterRecentTracks`, Card #37) — a
  **hard eligibility guardrail** that runs *before* scoring, on the whole
  Candidate Pool. It answers: "was this track used recently enough that it
  should currently be excluded?" A recently-used candidate never reaches
  scoring at all in the normal pipeline.
- **Repetition Penalty** — a **soft scoring signal** for candidates that
  already passed the hard filter. It answers a different question: "even
  though this candidate is eligible, does its broader repetition history
  (e.g. the same artist, album, or musical territory appearing often) make
  it less valuable right now?"

The Recent Track Filter is never replaced by the Repetition Penalty, and
scoring never reintroduces a recently-used track into eligibility — the
two mechanisms are independent and both stay in the pipeline.

## Weights

Weights are explicit, configurable, and documented as **initial editorial
assumptions, not scientifically validated values**. They live in one
struct (`scoring.Weights`), constructor-supplied via `scoring.
DefaultWeights()` — never hardcoded inline elsewhere.

| Weight | Value | Applies to |
|---|---|---|
| Fit | 0.35 | positive factor |
| Playlist Fit | 0.25 | positive factor |
| Discovery Bonus | 0.15 | positive factor |
| Diversity | 0.15 | positive factor |
| Freshness | 0.10 | positive factor |
| **Sum (positive factors)** | **1.00** | — |
| Repetition Weight | 0.30 | penalty cap, independent of the sum above |

**Reasoning, grounded in the manifesto:**

- **Fit (0.35) + Playlist Fit (0.25) = 60%** of the score. The manifesto
  names musical bridges "the connections between tracks... the most
  important editorial craft in the project" (§4), and Card #40 itself
  names Fit the single most important factor. Together, fit and continuity
  with the current journey must dominate the model.
- **Discovery Bonus and Diversity are equal at 0.15 each.** Both serve
  "discovery without forced obscurity" (§5) — present in the score, never
  allowed to dominate it.
- **Freshness is lowest, at 0.10.** Card #40 explicitly warns that
  freshness or novelty must never overwhelm musical fit; recency is a
  minor signal, never a substitute for belonging.
- **Repetition Weight (0.30)** caps how much repetition can discount an
  otherwise-strong score. High enough to matter editorially, low enough
  that no single signal can zero out a strong candidate outright — the
  score stays decision-support, never an automatic reject.

## Formula

```
BaseScore    = Σ(factor × weight) / Σ(weight)   — over available positive factors only
FinalScore   = BaseScore × (1 − RepetitionPenalty × RepetitionWeight)
```

**Worked example**, using `DefaultWeights()` and every factor available:

```
Fit=0.82, Freshness=0.65, DiscoveryBonus=0.90, Diversity=0.70, PlaylistFit=0.88, RepetitionPenalty=0.10

BaseScore  = 0.82×0.35 + 0.65×0.10 + 0.90×0.15 + 0.70×0.15 + 0.88×0.25
           = 0.287 + 0.065 + 0.135 + 0.105 + 0.220
           = 0.812
FinalScore = 0.812 × (1 − 0.10×0.30)
           = 0.812 × 0.97
           = 0.7876
```

Repetition is combined **multiplicatively**, not by direct subtraction. A
subtractive penalty (`BaseScore − penalty×weight`) could drive an
already-low `BaseScore` negative, which would then need arbitrary
clamping to stay in range — exactly what this model avoids. Because
`BaseScore ∈ [0,1]` and `RepetitionPenalty × RepetitionWeight ∈ [0,1]`,
the multiplicative form guarantees `FinalScore ∈ [0, BaseScore] ⊆ [0,1]`
by construction, with no clamping anywhere.

## Missing factors

Every factor value is `*float64` — nil means "not yet available." This
single nil state deliberately collapses "not yet implemented," "not
applicable in this context," and "temporarily unknown" into one concept:
the model only needs to know whether a value is usable, not why it's
missing.

A missing positive factor is **excluded**, not substituted with `0.0` —
`BaseScore` is a weight-renormalized average over only the factors that
are actually present. This is the model's central missing-data rule: an
unimplemented factor never silently drags a candidate's score down as if
it had scored the worst possible value. `CandidateScore.AvailableWeight`
exposes how much of the positive-factor weight was actually used, so a
consumer can judge how complete a given score currently is.

If **no present factor contributes any weight** — every positive factor is
missing, or (an edge case only reachable with a non-default `Weights`) the
only present factors carry a curator-assigned weight of `0` — `FinalScore`
is `nil` — there is nothing yet to report, never a fabricated `0.0` that
would misleadingly look like "weakest possible candidate."

A missing `RepetitionPenalty` is treated as `0.0` (no discount): silence
about repetition history is not evidence of repetition.

Since M5 implements factors one card at a time, Fit and Freshness are the
only positive factors with an implemented algorithm as of Card #42 —
`scoring.Calculate` handles all six factors, but Discovery Bonus,
Diversity, Repetition Penalty, and Playlist Fit have no production code
path constructing real values yet.

## Explainability

`CandidateScore` exposes every input and intermediate value as an
independently inspectable field — `CandidateID`, `ModelVersion`,
`Weights`, `Factors`, `FinalScore`, `AvailableWeight` — never only a
single opaque number. A future API/UI can render something like:

```
Candidate: Example Track
Fit: 0.82
Freshness: 0.65
Discovery Bonus: 0.90
Diversity: 0.70
Playlist Fit: 0.88
Repetition Penalty: 0.10
Final Score: 0.79
```

That rendering is not implemented by this card — only the underlying
struct that makes it possible.

## Model version

`scoring.ModelVersion` ("v1") is a plain string label carried on every
`CandidateScore`, for future traceability if factor definitions, weights,
or the formula change in a way that makes scores across versions
incomparable. There is no score history, persistence, or version
migration — this is a label, not an infrastructure feature.

## What this is not

- Not ranking: nothing here sorts a candidate pool.
- Not selection: nothing here chooses, accepts, or rejects a candidate.
- Not a `CandidateTrack.Status` mutation of any kind.
- Not a Spotify playlist mutation of any kind.
- Not popularity-based: no Spotify popularity/followers, no Last.fm
  popularity, no social/follower counts feed any factor.
- Not threshold-based: this card defines no "score ≥ X means select"
  rule, and none should be added without a dedicated future decision.
- Not machine learning: every factor and the combination formula are
  deterministic, explainable, and hand-authored.
