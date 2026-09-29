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
time.

### Fit

The most important conceptual factor: whether a candidate belongs to Sound
Continuum's editorial identity — musical character, mood, energy, texture,
vocal/melodic/rhythmic qualities, cultural context. Not derived from
genre strings, Spotify popularity, or any other metadata proxy; the actual
analysis is future work.

### Freshness

How timely or recently relevant a candidate is — distinct from
`candidate.Type`/`Category`, which are editorial classification, not a
scoring signal. A `Past` candidate can still have meaningful freshness if
it's being rediscovered; a `New Release` candidate does not automatically
score `Freshness = 1.0` just because it's new.

### Discovery Bonus

The editorial value of surfacing something less obvious to the audience —
*useful* discovery potential, not simply low popularity. "Unknown" does
not automatically mean "valuable," and `Category: Emerging` does not
automatically mean a high score. Obscurity is never rewarded for its own
sake, per the manifesto's "discovery without forced obscurity" principle.

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

Since M5 implements factors one card at a time, every positive factor is
`nil` as of Card #40 — `scoring.Calculate` is fully implemented and
tested, but no production code path constructs real factor values yet.

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
