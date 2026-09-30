package scoring

import "errors"

// Sentinels for Weights/Factors validation failures. Use errors.Is against
// these.
var (
	ErrWeightOutOfRange = errors.New("scoring: weight must be in [0,1]")
	ErrWeightSumInvalid = errors.New("scoring: fit, freshness, discovery bonus, diversity, and playlist fit weights must sum to 1.0")
	ErrFactorOutOfRange = errors.New("scoring: factor value must be in [0,1]")
	ErrEmptyCandidateID = errors.New("scoring: candidate id must not be empty")

	// ErrFitComponentWeightSumInvalid is returned when FitWeights'
	// ProjectWeight and WeeklyWeight do not sum to 1.0.
	ErrFitComponentWeightSumInvalid = errors.New("scoring: fit project and weekly weights must sum to 1.0")

	// ErrFitDimensionWeightSumInvalid is returned when FitWeights'
	// Mood, Energy, Texture, and CulturalInfluence do not sum to 1.0.
	ErrFitDimensionWeightSumInvalid = errors.New("scoring: fit mood, energy, texture, and cultural influence weights must sum to 1.0")

	// ErrFreshnessHalfLifeOutOfRange is returned when FreshnessConfig's
	// HalfLifeDays is not a positive number.
	ErrFreshnessHalfLifeOutOfRange = errors.New("scoring: freshness half-life days must be a positive number")
)
