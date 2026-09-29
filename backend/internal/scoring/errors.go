package scoring

import "errors"

// Sentinels for Weights/Factors validation failures. Use errors.Is against
// these.
var (
	ErrWeightOutOfRange = errors.New("scoring: weight must be in [0,1]")
	ErrWeightSumInvalid = errors.New("scoring: fit, freshness, discovery bonus, diversity, and playlist fit weights must sum to 1.0")
	ErrFactorOutOfRange = errors.New("scoring: factor value must be in [0,1]")
)
