package scoring

import (
	"math"
	"strings"

	"github.com/vmmatos/sound-continuum-project/internal/musicaldna"
)

// FitDimension identifies one musical-character dimension considered when
// evaluating Fit. Drawn from musicaldna.Profile — see that package for why
// these four and not others.
type FitDimension string

const (
	FitDimensionMood              FitDimension = "mood"
	FitDimensionEnergy            FitDimension = "energy"
	FitDimensionTexture           FitDimension = "texture"
	FitDimensionCulturalInfluence FitDimension = "cultural_influence"
)

// FitComponent identifies which of Fit's two contextual comparisons a
// FitDimensionResult belongs to.
type FitComponent string

const (
	FitComponentProjectDNA      FitComponent = "project_dna"
	FitComponentWeeklyDirection FitComponent = "weekly_direction"
)

// FitWeights holds Fit's editorial weights, as two independent groups that
// must each sum to 1.0 (see Validate):
//
//   - ProjectWeight/WeeklyWeight decide how much ProjectDNA vs
//     WeeklyDirection contribute to Fit.
//   - Mood/Energy/Texture/CulturalInfluence decide how much each dimension
//     contributes within a single comparison (against either ProjectDNA or
//     WeeklyDirection).
type FitWeights struct {
	ProjectWeight float64
	WeeklyWeight  float64

	Mood              float64
	Energy            float64
	Texture           float64
	CulturalInfluence float64
}

// DefaultFitWeights returns Sound Continuum's initial Fit weights.
//
// WeeklyWeight (0.75) far outweighs ProjectWeight (0.25): the manifesto
// treats each edition as its own chapter, and Card #41 requires the current
// week's direction to carry "greater contextual importance than a generic
// project-wide similarity." In practice this split is realized structurally,
// not just by number — musicaldna.DefaultProjectDNA leaves every dimension
// unset, so the project component contributes nothing until a real,
// documented project-wide trait exists, and Fit is driven entirely by
// WeeklyDirection until then.
//
// Mood (0.35) is weighted highest: it is the most direct expression of "does
// this belong to the story we're telling right now." Energy and Texture
// (0.25 each) are weighted equally as the next most legible qualities from
// editorial listening. CulturalInfluence is weighted lowest (0.15): useful
// context, but Card #41 explicitly warns against genre/cultural matching
// becoming the definition of Fit.
func DefaultFitWeights() FitWeights {
	return FitWeights{
		ProjectWeight: 0.25,
		WeeklyWeight:  0.75,

		Mood:              0.35,
		Energy:            0.25,
		Texture:           0.25,
		CulturalInfluence: 0.15,
	}
}

// Validate checks that every weight lies in [0,1] (and is not NaN), that
// ProjectWeight+WeeklyWeight sum to 1.0, and that
// Mood+Energy+Texture+CulturalInfluence sum to 1.0 — both within
// weightSumTolerance.
func (w FitWeights) Validate() error {
	for _, v := range []float64{
		w.ProjectWeight, w.WeeklyWeight,
		w.Mood, w.Energy, w.Texture, w.CulturalInfluence,
	} {
		if math.IsNaN(v) || v < 0 || v > 1 {
			return ErrWeightOutOfRange
		}
	}
	if diff := (w.ProjectWeight + w.WeeklyWeight) - 1.0; diff < -weightSumTolerance || diff > weightSumTolerance {
		return ErrFitComponentWeightSumInvalid
	}
	dimSum := w.Mood + w.Energy + w.Texture + w.CulturalInfluence
	if diff := dimSum - 1.0; diff < -weightSumTolerance || diff > weightSumTolerance {
		return ErrFitDimensionWeightSumInvalid
	}
	return nil
}

// FitDimensionResult explains how one dimension contributed to one
// comparison. Available is false when either side lacks this dimension —
// Match is only meaningful when Available is true.
type FitDimensionResult struct {
	Component FitComponent
	Dimension FitDimension
	Available bool
	Match     bool
}

// FitResult is the outcome of CalculateFit: a normalized [0,1] Value (nil
// when nothing was comparable — never a fabricated 0.0) plus every
// dimension's contribution, so a low or missing Fit is always explainable
// rather than an opaque number.
type FitResult struct {
	Value      *float64
	Dimensions []FitDimensionResult
}

// CalculateFit computes the Fit factor: how naturally candidateProfile
// belongs to Sound Continuum's identity (project) and the current edition's
// direction (weekly). Both comparisons use the same exact-match strategy —
// case-insensitive, trimmed string equality per dimension — because no
// fuzzy/embedding similarity is available without the AI/ML this card
// explicitly excludes, and a partial-credit heuristic between arbitrary
// strings would be fabricated precision Card #41 forbids.
//
// A missing dimension (nil on either side of a comparison) is excluded from
// that comparison rather than counted as a mismatch: weights are
// renormalized over whichever dimensions are actually present, the same
// missing-data idiom Calculate uses for Factors. If a comparison has no
// dimension available at all, it contributes nothing — not a fabricated 0 —
// and the same renormalization happens one level up between the project and
// weekly components. Value is nil only when neither comparison had anything
// available.
//
// CalculateFit takes musicaldna.Profile/ProjectDNA/WeeklyDirection values
// only — never a candidate.CandidateTrack — so it cannot see, and cannot be
// affected by, CandidateType, Category, release date, discovery provenance,
// or playlist sequence, keeping Fit independent of Freshness, Discovery
// Bonus, and Playlist Fit by construction.
func CalculateFit(candidateProfile musicaldna.Profile, project musicaldna.ProjectDNA, direction musicaldna.WeeklyDirection, weights FitWeights) (FitResult, error) {
	if err := weights.Validate(); err != nil {
		return FitResult{}, err
	}

	projectValue, projectDims := compareProfiles(FitComponentProjectDNA, candidateProfile, project.Profile, weights)
	weeklyValue, weeklyDims := compareProfiles(FitComponentWeeklyDirection, candidateProfile, direction.Profile, weights)

	dims := make([]FitDimensionResult, 0, len(projectDims)+len(weeklyDims))
	dims = append(dims, projectDims...)
	dims = append(dims, weeklyDims...)

	components := []struct {
		value  *float64
		weight float64
	}{
		{projectValue, weights.ProjectWeight},
		{weeklyValue, weights.WeeklyWeight},
	}

	var weightedSum, availableWeight float64
	for _, c := range components {
		if c.value != nil {
			weightedSum += *c.value * c.weight
			availableWeight += c.weight
		}
	}

	if availableWeight == 0 {
		return FitResult{Dimensions: dims}, nil
	}
	value := weightedSum / availableWeight
	return FitResult{Value: &value, Dimensions: dims}, nil
}

// compareProfiles compares candidate against target across all four
// dimensions, returning a weight-renormalized match ratio in [0,1] (nil if
// no dimension was comparable) plus one FitDimensionResult per dimension.
func compareProfiles(component FitComponent, candidate, target musicaldna.Profile, weights FitWeights) (*float64, []FitDimensionResult) {
	dims := []struct {
		name   FitDimension
		weight float64
		a, b   *string
	}{
		{FitDimensionMood, weights.Mood, candidate.Mood, target.Mood},
		{FitDimensionEnergy, weights.Energy, candidate.Energy, target.Energy},
		{FitDimensionTexture, weights.Texture, candidate.Texture, target.Texture},
		{FitDimensionCulturalInfluence, weights.CulturalInfluence, candidate.CulturalInfluence, target.CulturalInfluence},
	}

	results := make([]FitDimensionResult, 0, len(dims))
	var totalWeight, matchedWeight float64
	for _, d := range dims {
		if d.a == nil || d.b == nil {
			results = append(results, FitDimensionResult{Component: component, Dimension: d.name, Available: false})
			continue
		}
		totalWeight += d.weight
		match := strings.EqualFold(strings.TrimSpace(*d.a), strings.TrimSpace(*d.b))
		if match {
			matchedWeight += d.weight
		}
		results = append(results, FitDimensionResult{Component: component, Dimension: d.name, Available: true, Match: match})
	}

	if totalWeight == 0 {
		return nil, results
	}
	value := matchedWeight / totalWeight
	return &value, results
}
