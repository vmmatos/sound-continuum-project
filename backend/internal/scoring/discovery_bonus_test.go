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

func TestCalculateDiscoveryBonusCategoryBoundary(t *testing.T) {
	cases := []struct {
		category     candidate.Category
		value        *float64
		wantEligible bool
	}{
		{candidate.CategoryPast, float64Ptr(1.0), false},
		{candidate.CategoryPresent, float64Ptr(1.0), false},
		{candidate.CategoryPast, nil, false},
		{candidate.CategoryEmerging, float64Ptr(0.5), true},
		{candidate.Category(""), float64Ptr(0.5), false},
	}
	for _, c := range cases {
		got, err := CalculateDiscoveryBonus(c.category, c.value)
		if err != nil {
			t.Fatalf("CalculateDiscoveryBonus(%q) err = %v, want nil", c.category, err)
		}
		if got.Eligible != c.wantEligible {
			t.Errorf("Category=%q: Eligible = %v, want %v", c.category, got.Eligible, c.wantEligible)
		}
		if !c.wantEligible && got.Value != nil {
			t.Errorf("Category=%q: Value = %v, want nil (not applicable, never 0.0)", c.category, *got.Value)
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
	for _, v := range []float64{0.25, 0.75} {
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

// Independence from CandidateType, discovery provenance, Last.fm match,
// release date, Fit, Freshness, Diversity, PlaylistFit, RepetitionPenalty,
// and Spotify popularity/followers holds by construction:
// CalculateDiscoveryBonus's signature accepts only a candidate.Category and
// an editorial value, and none of those fields exist as parameters (or, for
// popularity/followers, anywhere in this codebase's Spotify model — see
// spotify/types.go) — so no test can exercise a dependency the function has
// no way to receive. TestCalculateDiscoveryBonusDeterministic above already
// covers "same inputs, same output."
