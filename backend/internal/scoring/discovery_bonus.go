package scoring

import (
	"math"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
)

// DiscoveryBonusResult is the outcome of CalculateDiscoveryBonus. Value is
// nil unless Category is candidate.CategoryEmerging AND an explicit,
// in-range editorial discovery value was supplied — an eligible-but-
// unassessed candidate stays nil, never a fabricated 0.0, and an explicit
// 0.0 is preserved, never collapsed into nil. Eligible and Supplied are
// independently inspectable so an explanation can distinguish "not
// eligible" from "eligible, not yet assessed" from "eligible and scored."
type DiscoveryBonusResult struct {
	Value    *float64
	Category candidate.Category
	Eligible bool
	Supplied bool
}

// CalculateDiscoveryBonus computes the Discovery Bonus factor: the editorial
// value of surfacing this candidate as a genuine discovery, not how unknown
// or obscure it is. category is the candidate's editorial Category
// (candidate.CandidateTrack.Category) — candidate.CategoryEmerging is the
// only eligibility gate, and eligibility alone is never sufficient.
// editorialDiscoveryValue is an explicit, editorially-supplied assessment in
// [0,1]; nil means "not yet assessed," which must not be treated as 0.0.
//
// v1's formula is intentionally trivial: DiscoveryBonus =
// editorialDiscoveryValue, when the candidate is eligible and a value was
// supplied. There is no curve, no popularity inversion, no multi-source
// confidence model — Card #43 defines the editorial signal, not a
// music-intelligence system.
//
// CalculateDiscoveryBonus deliberately takes no candidate.CandidateTrack,
// CandidateType, candidate.DiscoveryProvenance, Last.fm similarity/match
// value, Spotify popularity/followers (removed from this project's Spotify
// model entirely — see spotify/types.go), release date, Fit, Freshness,
// Diversity, PlaylistFit, or RepetitionPenalty — none of those are formula
// inputs, by construction. A candidate's discovery provenance remains
// available directly on CandidateTrack.Provenance for editorial explanation
// alongside this result; it is context, never a scoring input, and is never
// threaded through this calculation.
//
// editorialDiscoveryValue is validated first: NaN or outside [0,1] returns a
// zero-value DiscoveryBonusResult and ErrDiscoveryBonusValueOutOfRange, with
// no partial computation — mirroring CalculateFit/CalculateFreshness.
func CalculateDiscoveryBonus(category candidate.Category, editorialDiscoveryValue *float64) (DiscoveryBonusResult, error) {
	if editorialDiscoveryValue != nil {
		v := *editorialDiscoveryValue
		if math.IsNaN(v) || v < 0 || v > 1 {
			return DiscoveryBonusResult{}, ErrDiscoveryBonusValueOutOfRange
		}
	}

	result := DiscoveryBonusResult{
		Category: category,
		Eligible: category == candidate.CategoryEmerging,
		Supplied: editorialDiscoveryValue != nil,
	}

	if result.Eligible && result.Supplied {
		v := *editorialDiscoveryValue
		result.Value = &v
	}

	return result, nil
}
