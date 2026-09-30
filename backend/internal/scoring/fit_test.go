package scoring

import (
	"errors"
	"math"
	"testing"

	"github.com/vmmatos/sound-continuum-project/internal/musicaldna"
)

func fitStrPtr(s string) *string { return &s }

func fullProfile() musicaldna.Profile {
	return musicaldna.Profile{
		Mood:              fitStrPtr("melancholic"),
		Energy:            fitStrPtr("low"),
		Texture:           fitStrPtr("organic"),
		CulturalInfluence: fitStrPtr("west african"),
	}
}

func mustWeeklyDirection(t *testing.T, profile musicaldna.Profile) musicaldna.WeeklyDirection {
	t.Helper()
	d, err := musicaldna.NewWeeklyDirection("2026-w40", profile, "")
	if err != nil {
		t.Fatalf("NewWeeklyDirection() err = %v, want nil", err)
	}
	return d
}

// --- FitWeights ---

func TestDefaultFitWeightsValid(t *testing.T) {
	if err := DefaultFitWeights().Validate(); err != nil {
		t.Fatalf("DefaultFitWeights().Validate() = %v, want nil", err)
	}
}

func TestFitWeightsOutOfRange(t *testing.T) {
	base := DefaultFitWeights()
	setters := map[string]func(v float64) FitWeights{
		"ProjectWeight":     func(v float64) FitWeights { w := base; w.ProjectWeight = v; return w },
		"WeeklyWeight":      func(v float64) FitWeights { w := base; w.WeeklyWeight = v; return w },
		"Mood":              func(v float64) FitWeights { w := base; w.Mood = v; return w },
		"Energy":            func(v float64) FitWeights { w := base; w.Energy = v; return w },
		"Texture":           func(v float64) FitWeights { w := base; w.Texture = v; return w },
		"CulturalInfluence": func(v float64) FitWeights { w := base; w.CulturalInfluence = v; return w },
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

func TestFitWeightsNaNRejected(t *testing.T) {
	w := DefaultFitWeights()
	w.Mood = math.NaN()
	if err := w.Validate(); !errors.Is(err, ErrWeightOutOfRange) {
		t.Errorf("NaN weight: err = %v, want ErrWeightOutOfRange", err)
	}
}

func TestFitWeightsComponentSumInvalid(t *testing.T) {
	w := DefaultFitWeights()
	w.ProjectWeight, w.WeeklyWeight = 0.1, 0.5 // sums to 0.6
	if err := w.Validate(); !errors.Is(err, ErrFitComponentWeightSumInvalid) {
		t.Errorf("err = %v, want ErrFitComponentWeightSumInvalid", err)
	}
}

func TestFitWeightsDimensionSumInvalid(t *testing.T) {
	w := DefaultFitWeights()
	w.Mood, w.Energy, w.Texture, w.CulturalInfluence = 0.1, 0.1, 0.1, 0.1 // sums to 0.4
	if err := w.Validate(); !errors.Is(err, ErrFitDimensionWeightSumInvalid) {
		t.Errorf("err = %v, want ErrFitDimensionWeightSumInvalid", err)
	}
}

// --- CalculateFit: strong/weak fit ---

func TestCalculateFitStrongFitScoresHigh(t *testing.T) {
	profile := fullProfile()
	direction := mustWeeklyDirection(t, profile)

	got, err := CalculateFit(profile, musicaldna.DefaultProjectDNA(), direction, DefaultFitWeights())
	if err != nil {
		t.Fatalf("CalculateFit() err = %v, want nil", err)
	}
	if got.Value == nil {
		t.Fatal("Value = nil, want ~1.0")
	}
	if diff := *got.Value - 1.0; diff < -1e-9 || diff > 1e-9 {
		t.Errorf("Value = %v, want 1.0", *got.Value)
	}
}

func TestCalculateFitWeakFitScoresLow(t *testing.T) {
	candidateProfile := fullProfile()
	opposite := musicaldna.Profile{
		Mood:              fitStrPtr("euphoric"),
		Energy:            fitStrPtr("high"),
		Texture:           fitStrPtr("synthetic"),
		CulturalInfluence: fitStrPtr("scandinavian"),
	}
	direction := mustWeeklyDirection(t, opposite)

	got, err := CalculateFit(candidateProfile, musicaldna.DefaultProjectDNA(), direction, DefaultFitWeights())
	if err != nil {
		t.Fatalf("CalculateFit() err = %v, want nil", err)
	}
	if got.Value == nil {
		t.Fatal("Value = nil, want 0.0")
	}
	if diff := *got.Value - 0.0; diff < -1e-9 || diff > 1e-9 {
		t.Errorf("Value = %v, want 0.0", *got.Value)
	}
}

// --- CalculateFit: missing-data behavior ---

func TestCalculateFitPartialInformationDoesNotDefaultToZero(t *testing.T) {
	// Only Mood is known on both sides, and it matches. The other three
	// dimensions are unavailable, not mismatched, so Value should reflect
	// the one available (matching) dimension, not be dragged toward 0.
	candidateProfile := musicaldna.Profile{Mood: fitStrPtr("melancholic")}
	direction := mustWeeklyDirection(t, musicaldna.Profile{Mood: fitStrPtr("Melancholic")})

	got, err := CalculateFit(candidateProfile, musicaldna.DefaultProjectDNA(), direction, DefaultFitWeights())
	if err != nil {
		t.Fatalf("CalculateFit() err = %v, want nil", err)
	}
	if got.Value == nil {
		t.Fatal("Value = nil, want 1.0")
	}
	if diff := *got.Value - 1.0; diff < -1e-9 || diff > 1e-9 {
		t.Errorf("Value = %v, want 1.0 (the one available dimension matches)", *got.Value)
	}
}

func TestCalculateFitMissingDataYieldsNilNotZero(t *testing.T) {
	// Candidate has no known dimensions at all. ProjectDNA is also empty by
	// default, so nothing is comparable anywhere.
	direction := mustWeeklyDirection(t, fullProfile())

	got, err := CalculateFit(musicaldna.Profile{}, musicaldna.DefaultProjectDNA(), direction, DefaultFitWeights())
	if err != nil {
		t.Fatalf("CalculateFit() err = %v, want nil", err)
	}
	if got.Value != nil {
		t.Errorf("Value = %v, want nil (unknown must not become poor fit)", *got.Value)
	}
}

// --- CalculateFit: normalization ---

func TestCalculateFitValueStaysNormalized(t *testing.T) {
	candidateProfile := musicaldna.Profile{
		Mood:    fitStrPtr("melancholic"),
		Energy:  fitStrPtr("high"), // mismatch
		Texture: fitStrPtr("organic"),
	}
	direction := mustWeeklyDirection(t, fullProfile())

	got, err := CalculateFit(candidateProfile, musicaldna.DefaultProjectDNA(), direction, DefaultFitWeights())
	if err != nil {
		t.Fatalf("CalculateFit() err = %v, want nil", err)
	}
	if got.Value == nil {
		t.Fatal("Value = nil, want a computed value")
	}
	if *got.Value < 0 || *got.Value > 1 {
		t.Errorf("Value = %v, want within [0,1]", *got.Value)
	}
}

// --- CalculateFit: weight validation ---

func TestCalculateFitInvalidWeightsReturnsError(t *testing.T) {
	w := DefaultFitWeights()
	w.Mood = 2.0
	_, err := CalculateFit(fullProfile(), musicaldna.DefaultProjectDNA(), mustWeeklyDirection(t, fullProfile()), w)
	if !errors.Is(err, ErrWeightOutOfRange) {
		t.Fatalf("err = %v, want ErrWeightOutOfRange", err)
	}
}

// --- CalculateFit: determinism ---

func TestCalculateFitIsDeterministic(t *testing.T) {
	profile := fullProfile()
	direction := mustWeeklyDirection(t, fullProfile())
	weights := DefaultFitWeights()

	a, err := CalculateFit(profile, musicaldna.DefaultProjectDNA(), direction, weights)
	if err != nil {
		t.Fatalf("CalculateFit() err = %v, want nil", err)
	}
	b, err := CalculateFit(profile, musicaldna.DefaultProjectDNA(), direction, weights)
	if err != nil {
		t.Fatalf("CalculateFit() err = %v, want nil", err)
	}
	if *a.Value != *b.Value {
		t.Errorf("Value differs across identical calls: %v vs %v", *a.Value, *b.Value)
	}
}

// --- CalculateFit: independence from other factors' inputs ---
//
// CalculateFit takes only musicaldna.Profile/ProjectDNA/WeeklyDirection —
// never a candidate.CandidateTrack — so CandidateType, Category, release
// date, and discovery provenance cannot affect Fit by construction. Each
// test below stands in for two candidates identical except for the named
// dimension: since that dimension is never an input, the same
// Profile/WeeklyDirection must still produce the same Fit.

func assertFitUnaffectedBy(t *testing.T, dimension string) {
	t.Helper()
	profile := fullProfile()
	direction := mustWeeklyDirection(t, fullProfile())

	a, err := CalculateFit(profile, musicaldna.DefaultProjectDNA(), direction, DefaultFitWeights())
	if err != nil {
		t.Fatalf("CalculateFit() err = %v, want nil", err)
	}
	b, err := CalculateFit(profile, musicaldna.DefaultProjectDNA(), direction, DefaultFitWeights())
	if err != nil {
		t.Fatalf("CalculateFit() err = %v, want nil", err)
	}
	if *a.Value != *b.Value {
		t.Errorf("Fit differs by %s alone: %v vs %v", dimension, *a.Value, *b.Value)
	}
}

func TestCalculateFitIndependentOfCandidateClassification(t *testing.T) {
	assertFitUnaffectedBy(t, "Type/Category")
}

func TestCalculateFitIndependentOfReleaseDate(t *testing.T) {
	assertFitUnaffectedBy(t, "release date")
}

func TestCalculateFitIndependentOfProvenance(t *testing.T) {
	assertFitUnaffectedBy(t, "discovery provenance")
}

// --- Explainability ---

func TestCalculateFitDimensionsCoverBothComponents(t *testing.T) {
	got, err := CalculateFit(fullProfile(), musicaldna.DefaultProjectDNA(), mustWeeklyDirection(t, fullProfile()), DefaultFitWeights())
	if err != nil {
		t.Fatalf("CalculateFit() err = %v, want nil", err)
	}
	if len(got.Dimensions) != 8 {
		t.Fatalf("len(Dimensions) = %d, want 8 (4 dimensions x 2 components)", len(got.Dimensions))
	}

	var projectUnavailable, weeklyMatched int
	for _, d := range got.Dimensions {
		switch d.Component {
		case FitComponentProjectDNA:
			if !d.Available {
				projectUnavailable++
			}
		case FitComponentWeeklyDirection:
			if d.Available && d.Match {
				weeklyMatched++
			}
		default:
			t.Errorf("unexpected Component %v", d.Component)
		}
	}
	if projectUnavailable != 4 {
		t.Errorf("projectUnavailable = %d, want 4 (DefaultProjectDNA has no dimensions set)", projectUnavailable)
	}
	if weeklyMatched != 4 {
		t.Errorf("weeklyMatched = %d, want 4 (identical profiles)", weeklyMatched)
	}
}

// --- Score integration ---

func TestCalculateFitPopulatesCandidateScoreFitFactor(t *testing.T) {
	profile := fullProfile()
	direction := mustWeeklyDirection(t, fullProfile())

	fit, err := CalculateFit(profile, musicaldna.DefaultProjectDNA(), direction, DefaultFitWeights())
	if err != nil {
		t.Fatalf("CalculateFit() err = %v, want nil", err)
	}

	score, err := Calculate("cand-1", Factors{Fit: fit.Value}, DefaultWeights())
	if err != nil {
		t.Fatalf("Calculate() err = %v, want nil", err)
	}
	if score.Factors.Fit == nil || *score.Factors.Fit != *fit.Value {
		t.Errorf("Factors.Fit = %v, want %v", score.Factors.Fit, fit.Value)
	}
	for name, v := range map[string]*float64{
		"Freshness":         score.Factors.Freshness,
		"DiscoveryBonus":    score.Factors.DiscoveryBonus,
		"Diversity":         score.Factors.Diversity,
		"PlaylistFit":       score.Factors.PlaylistFit,
		"RepetitionPenalty": score.Factors.RepetitionPenalty,
	} {
		if v != nil {
			t.Errorf("Factors.%s = %v, want nil (untouched by Fit)", name, *v)
		}
	}
}
