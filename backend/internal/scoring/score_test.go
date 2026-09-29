package scoring

import (
	"errors"
	"testing"
)

func float64Ptr(v float64) *float64 { return &v }

func fullFactors() Factors {
	return Factors{
		Fit:               float64Ptr(0.8),
		Freshness:         float64Ptr(0.6),
		DiscoveryBonus:    float64Ptr(0.9),
		Diversity:         float64Ptr(0.7),
		PlaylistFit:       float64Ptr(0.5),
		RepetitionPenalty: float64Ptr(0.2),
	}
}

// --- Weights ---

func TestDefaultWeightsValid(t *testing.T) {
	if err := DefaultWeights().Validate(); err != nil {
		t.Fatalf("DefaultWeights().Validate() = %v, want nil", err)
	}
}

func TestWeightsOutOfRange(t *testing.T) {
	base := DefaultWeights()
	setters := map[string]func(v float64) Weights{
		"Fit":              func(v float64) Weights { w := base; w.Fit = v; return w },
		"Freshness":        func(v float64) Weights { w := base; w.Freshness = v; return w },
		"DiscoveryBonus":   func(v float64) Weights { w := base; w.DiscoveryBonus = v; return w },
		"Diversity":        func(v float64) Weights { w := base; w.Diversity = v; return w },
		"PlaylistFit":      func(v float64) Weights { w := base; w.PlaylistFit = v; return w },
		"RepetitionWeight": func(v float64) Weights { w := base; w.RepetitionWeight = v; return w },
	}
	for name, set := range setters {
		for _, v := range []float64{-0.1, 1.1} {
			w := set(v)
			if err := w.Validate(); !errors.Is(err, ErrWeightOutOfRange) {
				t.Errorf("%s=%v: err = %v, want ErrWeightOutOfRange", name, v, err)
			}
		}
	}
}

func TestWeightsSumInvalid(t *testing.T) {
	cases := []Weights{
		// sums to 0.9, every individual weight within [0,1]
		{Fit: 0.3, Freshness: 0.2, DiscoveryBonus: 0.2, Diversity: 0.1, PlaylistFit: 0.1, RepetitionWeight: 0.3},
		// sums to 1.1, every individual weight within [0,1]
		{Fit: 0.4, Freshness: 0.3, DiscoveryBonus: 0.2, Diversity: 0.1, PlaylistFit: 0.1, RepetitionWeight: 0.3},
	}
	for _, w := range cases {
		if err := w.Validate(); !errors.Is(err, ErrWeightSumInvalid) {
			t.Errorf("weights %+v: err = %v, want ErrWeightSumInvalid", w, err)
		}
	}
}

func TestWeightsSumToleranceBoundary(t *testing.T) {
	within := Weights{
		Fit: 0.35 + 1e-10, Freshness: 0.10, DiscoveryBonus: 0.15,
		Diversity: 0.15, PlaylistFit: 0.25, RepetitionWeight: 0.3,
	}
	if err := within.Validate(); err != nil {
		t.Errorf("sum off by 1e-10: err = %v, want nil", err)
	}

	outside := Weights{
		Fit: 0.35 + 1e-8, Freshness: 0.10, DiscoveryBonus: 0.15,
		Diversity: 0.15, PlaylistFit: 0.25, RepetitionWeight: 0.3,
	}
	if err := outside.Validate(); !errors.Is(err, ErrWeightSumInvalid) {
		t.Errorf("sum off by 1e-8: err = %v, want ErrWeightSumInvalid", err)
	}
}

// --- Factors ---

func TestFactorsAllNilValid(t *testing.T) {
	if err := (Factors{}).Validate(); err != nil {
		t.Fatalf("empty Factors.Validate() = %v, want nil", err)
	}
}

func TestFactorsOutOfRange(t *testing.T) {
	setters := map[string]func(v float64) Factors{
		"Fit":               func(v float64) Factors { return Factors{Fit: &v} },
		"Freshness":         func(v float64) Factors { return Factors{Freshness: &v} },
		"DiscoveryBonus":    func(v float64) Factors { return Factors{DiscoveryBonus: &v} },
		"Diversity":         func(v float64) Factors { return Factors{Diversity: &v} },
		"PlaylistFit":       func(v float64) Factors { return Factors{PlaylistFit: &v} },
		"RepetitionPenalty": func(v float64) Factors { return Factors{RepetitionPenalty: &v} },
	}
	for name, set := range setters {
		for _, v := range []float64{-0.1, 1.1} {
			f := set(v)
			if err := f.Validate(); !errors.Is(err, ErrFactorOutOfRange) {
				t.Errorf("%s=%v: err = %v, want ErrFactorOutOfRange", name, v, err)
			}
		}
		for _, v := range []float64{0.0, 1.0} {
			f := set(v)
			if err := f.Validate(); err != nil {
				t.Errorf("%s=%v (boundary): err = %v, want nil", name, v, err)
			}
		}
	}
}

// --- Calculate: missing-factor behavior ---

func TestCalculateAllFactorsPresentIsDeterministic(t *testing.T) {
	w := DefaultWeights()
	f := fullFactors()

	base := 0.8*w.Fit + 0.6*w.Freshness + 0.9*w.DiscoveryBonus + 0.7*w.Diversity + 0.5*w.PlaylistFit
	want := base * (1 - 0.2*w.RepetitionWeight)

	got, err := Calculate("cand-1", f, w)
	if err != nil {
		t.Fatalf("Calculate() err = %v, want nil", err)
	}
	if got.FinalScore == nil {
		t.Fatalf("FinalScore = nil, want %v", want)
	}
	if diff := *got.FinalScore - want; diff < -1e-12 || diff > 1e-12 {
		t.Errorf("FinalScore = %v, want %v", *got.FinalScore, want)
	}
	if got.AvailableWeight != 1.0 {
		t.Errorf("AvailableWeight = %v, want 1.0", got.AvailableWeight)
	}
}

func TestCalculateOneFactorMissingRenormalizes(t *testing.T) {
	w := DefaultWeights()
	cases := []struct {
		name       string
		mutate     func(f *Factors)
		wantWeight float64
		wantSum    float64
	}{
		{"Fit", func(f *Factors) { f.Fit = nil }, 1 - w.Fit, 0.6*w.Freshness + 0.9*w.DiscoveryBonus + 0.7*w.Diversity + 0.5*w.PlaylistFit},
		{"Freshness", func(f *Factors) { f.Freshness = nil }, 1 - w.Freshness, 0.8*w.Fit + 0.9*w.DiscoveryBonus + 0.7*w.Diversity + 0.5*w.PlaylistFit},
		{"DiscoveryBonus", func(f *Factors) { f.DiscoveryBonus = nil }, 1 - w.DiscoveryBonus, 0.8*w.Fit + 0.6*w.Freshness + 0.7*w.Diversity + 0.5*w.PlaylistFit},
		{"Diversity", func(f *Factors) { f.Diversity = nil }, 1 - w.Diversity, 0.8*w.Fit + 0.6*w.Freshness + 0.9*w.DiscoveryBonus + 0.5*w.PlaylistFit},
		{"PlaylistFit", func(f *Factors) { f.PlaylistFit = nil }, 1 - w.PlaylistFit, 0.8*w.Fit + 0.6*w.Freshness + 0.9*w.DiscoveryBonus + 0.7*w.Diversity},
	}
	for _, tc := range cases {
		f := fullFactors()
		tc.mutate(&f)

		got, err := Calculate("cand-1", f, w)
		if err != nil {
			t.Fatalf("%s: Calculate() err = %v, want nil", tc.name, err)
		}
		if diff := got.AvailableWeight - tc.wantWeight; diff < -1e-12 || diff > 1e-12 {
			t.Errorf("%s: AvailableWeight = %v, want %v", tc.name, got.AvailableWeight, tc.wantWeight)
		}
		wantBase := tc.wantSum / tc.wantWeight
		wantFinal := wantBase * (1 - 0.2*w.RepetitionWeight)
		if got.FinalScore == nil {
			t.Fatalf("%s: FinalScore = nil, want %v", tc.name, wantFinal)
		}
		if diff := *got.FinalScore - wantFinal; diff < -1e-9 || diff > 1e-9 {
			t.Errorf("%s: FinalScore = %v, want %v", tc.name, *got.FinalScore, wantFinal)
		}
	}
}

func TestCalculateAllPositiveFactorsMissingYieldsNilScore(t *testing.T) {
	f := Factors{RepetitionPenalty: float64Ptr(0.8)}
	got, err := Calculate("cand-1", f, DefaultWeights())
	if err != nil {
		t.Fatalf("Calculate() err = %v, want nil", err)
	}
	if got.FinalScore != nil {
		t.Errorf("FinalScore = %v, want nil", *got.FinalScore)
	}
	if got.AvailableWeight != 0 {
		t.Errorf("AvailableWeight = %v, want 0", got.AvailableWeight)
	}
}

func TestCalculateMissingRepetitionPenaltyAppliesNoDiscount(t *testing.T) {
	w := DefaultWeights()
	f := fullFactors()
	f.RepetitionPenalty = nil

	got, err := Calculate("cand-1", f, w)
	if err != nil {
		t.Fatalf("Calculate() err = %v, want nil", err)
	}
	base := 0.8*w.Fit + 0.6*w.Freshness + 0.9*w.DiscoveryBonus + 0.7*w.Diversity + 0.5*w.PlaylistFit
	if got.FinalScore == nil {
		t.Fatalf("FinalScore = nil, want %v", base)
	}
	if diff := *got.FinalScore - base; diff < -1e-12 || diff > 1e-12 {
		t.Errorf("FinalScore = %v, want BaseScore %v (no discount)", *got.FinalScore, base)
	}
}

// --- Calculate: repetition combination ---

func TestCalculateRepetitionMaxDiscountStaysBounded(t *testing.T) {
	one := 1.0
	f := Factors{Fit: &one, Freshness: &one, DiscoveryBonus: &one, Diversity: &one, PlaylistFit: &one, RepetitionPenalty: &one}
	w := DefaultWeights()
	w.RepetitionWeight = 1.0

	got, err := Calculate("cand-1", f, w)
	if err != nil {
		t.Fatalf("Calculate() err = %v, want nil", err)
	}
	if got.FinalScore == nil || *got.FinalScore != 0.0 {
		t.Errorf("FinalScore = %v, want 0.0", got.FinalScore)
	}
}

func TestCalculateRepetitionMidRangeIsExact(t *testing.T) {
	half := 0.5
	one := 1.0
	f := Factors{Fit: &one, Freshness: &one, DiscoveryBonus: &one, Diversity: &one, PlaylistFit: &one, RepetitionPenalty: &half}
	w := DefaultWeights()
	w.RepetitionWeight = 0.5

	got, err := Calculate("cand-1", f, w)
	if err != nil {
		t.Fatalf("Calculate() err = %v, want nil", err)
	}
	want := 1.0 * (1 - 0.5*0.5) // BaseScore=1.0, discount 0.25 -> 0.75
	if got.FinalScore == nil {
		t.Fatalf("FinalScore = nil, want %v", want)
	}
	if diff := *got.FinalScore - want; diff < -1e-12 || diff > 1e-12 {
		t.Errorf("FinalScore = %v, want %v", *got.FinalScore, want)
	}
}

// --- Calculate: error propagation ---

func TestCalculateInvalidWeightsReturnsZeroValue(t *testing.T) {
	w := DefaultWeights()
	w.Fit = 2.0 // out of range
	got, err := Calculate("cand-1", fullFactors(), w)
	if !errors.Is(err, ErrWeightOutOfRange) {
		t.Fatalf("err = %v, want ErrWeightOutOfRange", err)
	}
	if got != (CandidateScore{}) {
		t.Errorf("CandidateScore = %+v, want zero value", got)
	}
}

func TestCalculateInvalidFactorsReturnsZeroValue(t *testing.T) {
	bad := 1.5
	f := Factors{Fit: &bad}
	got, err := Calculate("cand-1", f, DefaultWeights())
	if !errors.Is(err, ErrFactorOutOfRange) {
		t.Fatalf("err = %v, want ErrFactorOutOfRange", err)
	}
	if got != (CandidateScore{}) {
		t.Errorf("CandidateScore = %+v, want zero value", got)
	}
}

// --- Explainability ---

func TestCandidateScoreExposesEveryFactor(t *testing.T) {
	w := DefaultWeights()
	f := fullFactors()
	got, err := Calculate("cand-42", f, w)
	if err != nil {
		t.Fatalf("Calculate() err = %v, want nil", err)
	}

	if got.CandidateID != "cand-42" {
		t.Errorf("CandidateID = %v, want cand-42", got.CandidateID)
	}
	if got.ModelVersion != ModelVersion {
		t.Errorf("ModelVersion = %v, want %v", got.ModelVersion, ModelVersion)
	}
	if got.Weights != w {
		t.Errorf("Weights = %+v, want %+v", got.Weights, w)
	}
	if *got.Factors.Fit != *f.Fit || *got.Factors.Freshness != *f.Freshness ||
		*got.Factors.DiscoveryBonus != *f.DiscoveryBonus || *got.Factors.Diversity != *f.Diversity ||
		*got.Factors.PlaylistFit != *f.PlaylistFit || *got.Factors.RepetitionPenalty != *f.RepetitionPenalty {
		t.Errorf("Factors = %+v, want values matching input %+v", got.Factors, f)
	}
	if got.FinalScore == nil {
		t.Error("FinalScore = nil, want a computed value")
	}
}

// --- Determinism ---

func TestCalculateIsDeterministic(t *testing.T) {
	w := DefaultWeights()
	f := fullFactors()

	a, err := Calculate("cand-1", f, w)
	if err != nil {
		t.Fatalf("Calculate() err = %v, want nil", err)
	}
	b, err := Calculate("cand-1", f, w)
	if err != nil {
		t.Fatalf("Calculate() err = %v, want nil", err)
	}

	if *a.FinalScore != *b.FinalScore {
		t.Errorf("FinalScore differs across identical calls: %v vs %v", *a.FinalScore, *b.FinalScore)
	}
	if a.AvailableWeight != b.AvailableWeight {
		t.Errorf("AvailableWeight differs across identical calls: %v vs %v", a.AvailableWeight, b.AvailableWeight)
	}
}
