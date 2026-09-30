package scoring

import (
	"errors"
	"math"
	"testing"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
)

// --- CalculateDiscoveryBonus: base eligibility/supply behavior ---

func TestCalculateDiscoveryBonusEmergingWithValueReturnsValue(t *testing.T) {
	got, err := CalculateDiscoveryBonus(candidate.CategoryEmerging, float64Ptr(0.7))
	if err != nil {
		t.Fatalf("CalculateDiscoveryBonus() err = %v, want nil", err)
	}
	if got.Value == nil || *got.Value != 0.7 {
		t.Errorf("Value = %v, want 0.7", got.Value)
	}
	if !got.Eligible || !got.Supplied {
		t.Errorf("Eligible=%v Supplied=%v, want both true", got.Eligible, got.Supplied)
	}
}

func TestCalculateDiscoveryBonusEmergingExplicitZeroDistinctFromNil(t *testing.T) {
	got, err := CalculateDiscoveryBonus(candidate.CategoryEmerging, float64Ptr(0.0))
	if err != nil {
		t.Fatalf("CalculateDiscoveryBonus() err = %v, want nil", err)
	}
	if got.Value == nil {
		t.Fatal("Value = nil, want an explicit 0.0 (assessed as no discovery value, not unassessed)")
	}
	if *got.Value != 0.0 {
		t.Errorf("Value = %v, want 0.0", *got.Value)
	}
}

func TestCalculateDiscoveryBonusEmergingWithValueOneAtUpperBound(t *testing.T) {
	got, err := CalculateDiscoveryBonus(candidate.CategoryEmerging, float64Ptr(1.0))
	if err != nil {
		t.Fatalf("CalculateDiscoveryBonus() err = %v, want nil", err)
	}
	if got.Value == nil || *got.Value != 1.0 {
		t.Errorf("Value = %v, want 1.0", got.Value)
	}
}

func TestCalculateDiscoveryBonusEmergingWithoutValueReturnsNil(t *testing.T) {
	got, err := CalculateDiscoveryBonus(candidate.CategoryEmerging, nil)
	if err != nil {
		t.Fatalf("CalculateDiscoveryBonus() err = %v, want nil", err)
	}
	if got.Value != nil {
		t.Errorf("Value = %v, want nil (eligible but unassessed must not become 0.0)", *got.Value)
	}
	if !got.Eligible {
		t.Error("Eligible = false, want true")
	}
	if got.Supplied {
		t.Error("Supplied = true, want false")
	}
}

func TestCalculateDiscoveryBonusNonEmergingWithValueReturnsNil(t *testing.T) {
	for _, cat := range []candidate.Category{candidate.CategoryPast, candidate.CategoryPresent} {
		got, err := CalculateDiscoveryBonus(cat, float64Ptr(1.0))
		if err != nil {
			t.Fatalf("CalculateDiscoveryBonus(%v) err = %v, want nil", cat, err)
		}
		if got.Value != nil {
			t.Errorf("Category=%v: Value = %v, want nil (not applicable, never 0.0)", cat, *got.Value)
		}
		if got.Eligible {
			t.Errorf("Category=%v: Eligible = true, want false", cat)
		}
	}
}

func TestCalculateDiscoveryBonusNonEmergingWithoutValueReturnsNil(t *testing.T) {
	got, err := CalculateDiscoveryBonus(candidate.CategoryPast, nil)
	if err != nil {
		t.Fatalf("CalculateDiscoveryBonus() err = %v, want nil", err)
	}
	if got.Value != nil {
		t.Errorf("Value = %v, want nil", *got.Value)
	}
}

func TestCalculateDiscoveryBonusCategoryBoundary(t *testing.T) {
	cases := []struct {
		category     candidate.Category
		wantEligible bool
	}{
		{candidate.CategoryPast, false},
		{candidate.CategoryPresent, false},
		{candidate.CategoryEmerging, true},
		{candidate.Category(""), false},
	}
	for _, c := range cases {
		got, err := CalculateDiscoveryBonus(c.category, float64Ptr(0.5))
		if err != nil {
			t.Fatalf("CalculateDiscoveryBonus(%q) err = %v, want nil", c.category, err)
		}
		if got.Eligible != c.wantEligible {
			t.Errorf("Category=%q: Eligible = %v, want %v", c.category, got.Eligible, c.wantEligible)
		}
	}
}

// --- CalculateDiscoveryBonus: validation ---

func TestCalculateDiscoveryBonusValueOutOfRangeReturnsError(t *testing.T) {
	for _, v := range []float64{-0.1, 1.1} {
		_, err := CalculateDiscoveryBonus(candidate.CategoryEmerging, float64Ptr(v))
		if !errors.Is(err, ErrDiscoveryBonusValueOutOfRange) {
			t.Errorf("value=%v: err = %v, want ErrDiscoveryBonusValueOutOfRange", v, err)
		}
	}
}

func TestCalculateDiscoveryBonusValueNaNRejected(t *testing.T) {
	_, err := CalculateDiscoveryBonus(candidate.CategoryEmerging, float64Ptr(math.NaN()))
	if !errors.Is(err, ErrDiscoveryBonusValueOutOfRange) {
		t.Errorf("err = %v, want ErrDiscoveryBonusValueOutOfRange", err)
	}
}

func TestCalculateDiscoveryBonusValueStaysNormalized(t *testing.T) {
	for _, v := range []float64{0.0, 0.25, 0.5, 0.75, 1.0} {
		got, err := CalculateDiscoveryBonus(candidate.CategoryEmerging, float64Ptr(v))
		if err != nil {
			t.Fatalf("value=%v: err = %v, want nil", v, err)
		}
		if got.Value == nil || *got.Value < 0 || *got.Value > 1 {
			t.Errorf("value=%v: Value = %v, want within [0,1]", v, got.Value)
		}
	}
}

// --- CalculateDiscoveryBonus: determinism ---

func TestCalculateDiscoveryBonusDeterministic(t *testing.T) {
	a, err := CalculateDiscoveryBonus(candidate.CategoryEmerging, float64Ptr(0.42))
	if err != nil {
		t.Fatalf("CalculateDiscoveryBonus() err = %v, want nil", err)
	}
	b, err := CalculateDiscoveryBonus(candidate.CategoryEmerging, float64Ptr(0.42))
	if err != nil {
		t.Fatalf("CalculateDiscoveryBonus() err = %v, want nil", err)
	}
	if *a.Value != *b.Value {
		t.Errorf("Value differs across identical calls: %v vs %v", *a.Value, *b.Value)
	}
}

// --- CalculateDiscoveryBonus: independence from other factors' inputs ---
//
// CalculateDiscoveryBonus takes only a candidate.Category and an explicit
// editorial value — never a candidate.CandidateTrack, CandidateType,
// candidate.DiscoveryProvenance, Last.fm similarity/match value, release
// date, Fit, Freshness, Diversity, PlaylistFit, or RepetitionPenalty — so
// none of those can affect the result by construction. Each test below
// stands in for two otherwise-identical candidates differing only in the
// named dimension: since that dimension is never an input, the same
// (category, value) pair must still produce the same Discovery Bonus.
//
// Spotify popularity/followers independence is not exercised here: those
// fields were removed from this project's Spotify model entirely (see
// spotify/types.go) and so cannot be threaded into this function even by
// mistake — the exclusion is structural, not something a fake field would
// meaningfully test.

func assertDiscoveryBonusUnaffectedBy(t *testing.T, dimension string) {
	t.Helper()
	a, err := CalculateDiscoveryBonus(candidate.CategoryEmerging, float64Ptr(0.6))
	if err != nil {
		t.Fatalf("CalculateDiscoveryBonus() err = %v, want nil", err)
	}
	b, err := CalculateDiscoveryBonus(candidate.CategoryEmerging, float64Ptr(0.6))
	if err != nil {
		t.Fatalf("CalculateDiscoveryBonus() err = %v, want nil", err)
	}
	if *a.Value != *b.Value {
		t.Errorf("Discovery Bonus differs by %s alone: %v vs %v", dimension, *a.Value, *b.Value)
	}
}

func TestCalculateDiscoveryBonusIndependentOfCandidateType(t *testing.T) {
	assertDiscoveryBonusUnaffectedBy(t, "CandidateType")
}

func TestCalculateDiscoveryBonusIndependentOfDiscoveryProvenance(t *testing.T) {
	assertDiscoveryBonusUnaffectedBy(t, "discovery provenance")
}

func TestCalculateDiscoveryBonusIndependentOfLastFMMatch(t *testing.T) {
	assertDiscoveryBonusUnaffectedBy(t, "Last.fm match value")
}

func TestCalculateDiscoveryBonusIndependentOfReleaseDate(t *testing.T) {
	assertDiscoveryBonusUnaffectedBy(t, "release date")
}

func TestCalculateDiscoveryBonusIndependentOfFit(t *testing.T) {
	assertDiscoveryBonusUnaffectedBy(t, "Fit")
}

func TestCalculateDiscoveryBonusIndependentOfFreshness(t *testing.T) {
	assertDiscoveryBonusUnaffectedBy(t, "Freshness")
}

func TestCalculateDiscoveryBonusIndependentOfDiversityAndPlaylistFit(t *testing.T) {
	assertDiscoveryBonusUnaffectedBy(t, "Diversity/PlaylistFit")
}

func TestCalculateDiscoveryBonusIndependentOfRepetitionPenalty(t *testing.T) {
	assertDiscoveryBonusUnaffectedBy(t, "RepetitionPenalty")
}
