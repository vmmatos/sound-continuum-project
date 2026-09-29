// Package scoring defines Sound Continuum's candidate scoring model — the
// structure by which a discovered candidate's editorial signals combine
// into one explainable score. It is a decision-support mechanism, not an
// automatic music selection mechanism: nothing in this package ranks,
// sorts, filters, selects, or rejects a candidate, and nothing here reads
// or writes candidate.CandidateTrack.Status. The human curator remains
// responsible for the final call — see docs/scoring-model.md for the full
// editorial reasoning.
//
// A score is built from six factors, each normalized to [0,1]:
//
//   - Fit: does this candidate belong to the Sound Continuum identity?
//   - Freshness: does this bring timely/recent relevance?
//   - DiscoveryBonus: does this offer useful discovery value?
//   - Diversity: does this add meaningful variation to the current context?
//   - PlaylistFit: does this make sense in the current musical journey?
//   - RepetitionPenalty: are we over-representing something? (0 = no
//     concern, 1 = strongest concern — the only factor that is a penalty,
//     not a positive contribution)
//
// This package does not calculate any of these six values — that is left
// to later M5 cards, one factor at a time. It only defines how already-
// calculated (or not-yet-calculated) factor values combine into one
// CandidateScore. See Calculate.
package scoring

import (
	"math"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
)

// ModelVersion identifies this scoring model definition — its factor set,
// combination formula, and default weights. Bump it whenever any of those
// change in a way that would make scores computed under different
// versions incomparable. There is no score history or persistence; this
// is a label only, carried on every CandidateScore for future
// traceability.
const ModelVersion = "v1"

// Factors holds each scoring factor's normalized value in [0,1], or nil
// when that factor cannot yet be calculated. A single nil state
// deliberately collapses "not yet implemented" / "not applicable in this
// context" / "temporarily unknown" into one concept — Calculate only
// needs to know whether a value is usable, not why it's missing. See
// Calculate's doc comment for how a nil factor affects the final score.
type Factors struct {
	Fit               *float64
	Freshness         *float64
	DiscoveryBonus    *float64
	Diversity         *float64
	PlaylistFit       *float64
	RepetitionPenalty *float64 // 0 = no repetition concern, 1 = strongest
}

// Validate checks that every non-nil factor value lies in [0,1] and is not
// NaN. A nil value is always valid — "not yet available" is never an
// error.
func (f Factors) Validate() error {
	for _, v := range []*float64{
		f.Fit, f.Freshness, f.DiscoveryBonus, f.Diversity,
		f.PlaylistFit, f.RepetitionPenalty,
	} {
		if v != nil && (math.IsNaN(*v) || *v < 0 || *v > 1) {
			return ErrFactorOutOfRange
		}
	}
	return nil
}

// Weights are Sound Continuum's initial editorial assumptions about how
// much each factor should influence a score — not scientifically
// validated values, and expected to change as M5 factor implementations
// mature. Fit, Freshness, DiscoveryBonus, Diversity, and PlaylistFit are
// the five positive contributions and must sum to 1.0 (see Validate).
// RepetitionWeight is independent of that sum: it caps how much the
// repetition penalty can discount an otherwise-strong score (see
// Calculate), it does not compete with the positive factors for share of
// the total.
type Weights struct {
	Fit              float64
	Freshness        float64
	DiscoveryBonus   float64
	Diversity        float64
	PlaylistFit      float64
	RepetitionWeight float64
}

// weightSumTolerance absorbs float64 summation error (on the order of
// 1e-16 for five terms like these) while still catching a real mis-sum.
const weightSumTolerance = 1e-9

// DefaultWeights returns Sound Continuum's initial scoring weights.
//
// Fit (0.35) and PlaylistFit (0.25) together make up 60% of the score.
// The manifesto names musical bridges "the connections between tracks...
// the most important editorial craft in the project," and Card #40
// itself names Fit the single most important factor — together they
// require the model to prioritize musical/editorial fit and playlist
// continuity above everything else.
//
// DiscoveryBonus and Diversity are weighted equally (0.15 each): both
// serve the manifesto's "discovery without forced obscurity" principle,
// present in the score but never allowed to dominate it — obscurity is
// never a goal in itself.
//
// Freshness is weighted lowest (0.10): Card #40 explicitly warns that
// freshness or novelty must never overwhelm musical fit, so recency can
// only ever be a minor signal, never a substitute for belonging.
//
// RepetitionWeight (0.30) caps how much the repetition penalty can
// discount an otherwise-strong candidate's score (see Calculate) — high
// enough to matter editorially, low enough that a single strong signal
// is never zeroed out by repetition alone. The score remains
// decision-support, never an automatic reject.
func DefaultWeights() Weights {
	return Weights{
		Fit:              0.35,
		PlaylistFit:      0.25,
		DiscoveryBonus:   0.15,
		Diversity:        0.15,
		Freshness:        0.10,
		RepetitionWeight: 0.30,
	}
}

// Validate checks that every weight lies in [0,1] (and is not NaN) and
// that the five positive factor weights (Fit, Freshness, DiscoveryBonus,
// Diversity, PlaylistFit) sum to 1.0 within weightSumTolerance.
func (w Weights) Validate() error {
	for _, v := range []float64{
		w.Fit, w.Freshness, w.DiscoveryBonus, w.Diversity,
		w.PlaylistFit, w.RepetitionWeight,
	} {
		if math.IsNaN(v) || v < 0 || v > 1 {
			return ErrWeightOutOfRange
		}
	}
	sum := w.Fit + w.Freshness + w.DiscoveryBonus + w.Diversity + w.PlaylistFit
	diff := sum - 1.0
	if diff < 0 {
		diff = -diff
	}
	if diff > weightSumTolerance {
		return ErrWeightSumInvalid
	}
	return nil
}

// CandidateScore is the result of scoring one candidate under one set of
// Weights. Every factor value, the weights that produced it, and the
// final number are independently inspectable fields — this struct IS the
// explainability mechanism the model requires; there is no single opaque
// number anywhere in this package.
//
// CandidateScore is purely informational. Nothing reads it to rank, sort,
// filter, select, or reject a candidate — that is future ranking work,
// deliberately out of scope for this package.
type CandidateScore struct {
	CandidateID  candidate.ID
	ModelVersion string
	Weights      Weights
	Factors      Factors

	// FinalScore is nil only when AvailableWeight is 0 — no present
	// factor contributes any weight to the score — and there is nothing
	// yet to report; nil is preferred over a fabricated 0.0 that would
	// misleadingly look like "weakest possible candidate."
	FinalScore *float64

	// AvailableWeight is the sum of the weights of positive factors that
	// are both present and actually contribute to FinalScore (see
	// Calculate) — 1.0 when every positive factor is present under
	// DefaultWeights, lower while some are still unimplemented. A present
	// factor whose own weight is 0 contributes nothing and is not counted
	// here, since a Weights.Validate-satisfying weight of 0 means the
	// curator has already excluded that factor from the score entirely.
	// AvailableWeight lets a caller judge how complete a given FinalScore
	// currently is; it is not itself part of the score.
	AvailableWeight float64
}

// Calculate computes a CandidateScore for one candidate's Factors under
// the given Weights. id, w, and f are validated first — an empty id or an
// invalid w/f returns the matching sentinel error and a zero-value
// CandidateScore, with no partial computation performed.
//
// Missing factors (nil in Factors) are treated as "not yet known," never
// as zero: the five positive factors are combined as a weight-
// renormalized average over only the factors that are actually present —
// sum(value*weight) / sum(weight) across the available ones — so an
// unimplemented factor is excluded rather than silently dragging the
// score down as if it scored 0. If no present factor contributes any
// weight (every positive factor is missing, or every present factor's
// own weight is 0), FinalScore is nil.
//
// RepetitionPenalty is combined multiplicatively, not subtracted:
// FinalScore = BaseScore * (1 - repetitionPenalty*Weights.RepetitionWeight).
// A nil RepetitionPenalty is treated as 0.0 (no discount) — silence about
// repetition history is not evidence of repetition. Because BaseScore and
// repetitionPenalty*RepetitionWeight are both in [0,1], FinalScore is
// always in [0, BaseScore] ⊆ [0,1] with no clamping: a subtractive
// penalty could drive an already-low BaseScore negative, forcing exactly
// the kind of arbitrary clamping this model avoids by construction.
func Calculate(id candidate.ID, f Factors, w Weights) (CandidateScore, error) {
	if id == "" {
		return CandidateScore{}, ErrEmptyCandidateID
	}
	if err := w.Validate(); err != nil {
		return CandidateScore{}, err
	}
	if err := f.Validate(); err != nil {
		return CandidateScore{}, err
	}

	type weighted struct {
		value  *float64
		weight float64
	}
	positive := []weighted{
		{f.Fit, w.Fit},
		{f.Freshness, w.Freshness},
		{f.DiscoveryBonus, w.DiscoveryBonus},
		{f.Diversity, w.Diversity},
		{f.PlaylistFit, w.PlaylistFit},
	}

	var weightedSum, availableWeight float64
	for _, p := range positive {
		if p.value != nil {
			weightedSum += *p.value * p.weight
			availableWeight += p.weight
		}
	}

	score := CandidateScore{
		CandidateID:     id,
		ModelVersion:    ModelVersion,
		Weights:         w,
		Factors:         f,
		AvailableWeight: availableWeight,
	}

	if availableWeight > 0 {
		base := weightedSum / availableWeight
		penalty := 0.0
		if f.RepetitionPenalty != nil {
			penalty = *f.RepetitionPenalty
		}
		final := base * (1 - penalty*w.RepetitionWeight)
		score.FinalScore = &final
	}

	return score, nil
}
