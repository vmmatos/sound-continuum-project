package scoring

import (
	"errors"
	"math"
	"testing"
	"time"
)

var freshnessNow = time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

func daysBefore(now time.Time, days float64) *time.Time {
	t := now.Add(-time.Duration(days * float64(24*time.Hour)))
	return &t
}

// --- FreshnessConfig ---

func TestDefaultFreshnessConfigValid(t *testing.T) {
	if err := DefaultFreshnessConfig().Validate(); err != nil {
		t.Fatalf("DefaultFreshnessConfig().Validate() = %v, want nil", err)
	}
}

func TestFreshnessConfigHalfLifeOutOfRange(t *testing.T) {
	for _, v := range []float64{0, -1, -60} {
		c := FreshnessConfig{HalfLifeDays: v}
		if err := c.Validate(); !errors.Is(err, ErrFreshnessHalfLifeOutOfRange) {
			t.Errorf("HalfLifeDays=%v: err = %v, want ErrFreshnessHalfLifeOutOfRange", v, err)
		}
	}
}

func TestFreshnessConfigHalfLifeNaNRejected(t *testing.T) {
	c := FreshnessConfig{HalfLifeDays: math.NaN()}
	if err := c.Validate(); !errors.Is(err, ErrFreshnessHalfLifeOutOfRange) {
		t.Errorf("err = %v, want ErrFreshnessHalfLifeOutOfRange", err)
	}
}

func TestCalculateFreshnessInvalidConfigReturnsError(t *testing.T) {
	_, err := CalculateFreshness(nil, freshnessNow, FreshnessConfig{HalfLifeDays: 0})
	if !errors.Is(err, ErrFreshnessHalfLifeOutOfRange) {
		t.Fatalf("err = %v, want ErrFreshnessHalfLifeOutOfRange", err)
	}
}

// --- Never used ---

func TestCalculateFreshnessNeverUsedReturnsOne(t *testing.T) {
	got, err := CalculateFreshness(nil, freshnessNow, DefaultFreshnessConfig())
	if err != nil {
		t.Fatalf("CalculateFreshness() err = %v, want nil", err)
	}
	if got.Value != 1.0 {
		t.Errorf("Value = %v, want 1.0", got.Value)
	}
	if got.LastUsedAt != nil {
		t.Errorf("LastUsedAt = %v, want nil", got.LastUsedAt)
	}
}

// --- Used candidates: gradient ---

func TestCalculateFreshnessVeryRecentIsLow(t *testing.T) {
	lastUsedAt := daysBefore(freshnessNow, 1)
	got, err := CalculateFreshness(lastUsedAt, freshnessNow, DefaultFreshnessConfig())
	if err != nil {
		t.Fatalf("CalculateFreshness() err = %v, want nil", err)
	}
	if got.Value >= 0.1 {
		t.Errorf("Value = %v, want a low value close to 0 for a 1-day-old use", got.Value)
	}
}

func TestCalculateFreshnessJustOutsideLookbackWindowBelowMax(t *testing.T) {
	// 29 days: just past #37's 28-day hard-exclusion boundary, so the
	// candidate is eligible for scoring, but Freshness must still be well
	// below the maximum.
	lastUsedAt := daysBefore(freshnessNow, 29)
	got, err := CalculateFreshness(lastUsedAt, freshnessNow, DefaultFreshnessConfig())
	if err != nil {
		t.Fatalf("CalculateFreshness() err = %v, want nil", err)
	}
	if got.Value >= 0.9 {
		t.Errorf("Value = %v, want well below max for a 29-day-old use", got.Value)
	}
}

func TestCalculateFreshnessLongerGapHigherThanRecent(t *testing.T) {
	shortGap, err := CalculateFreshness(daysBefore(freshnessNow, 35), freshnessNow, DefaultFreshnessConfig())
	if err != nil {
		t.Fatalf("CalculateFreshness() err = %v, want nil", err)
	}
	longGap, err := CalculateFreshness(daysBefore(freshnessNow, 180), freshnessNow, DefaultFreshnessConfig())
	if err != nil {
		t.Fatalf("CalculateFreshness() err = %v, want nil", err)
	}
	if !(longGap.Value > shortGap.Value) {
		t.Errorf("35d Freshness = %v, 180d Freshness = %v, want 180d > 35d", shortGap.Value, longGap.Value)
	}
}

func TestCalculateFreshnessVeryOldIsHigh(t *testing.T) {
	got, err := CalculateFreshness(daysBefore(freshnessNow, 730), freshnessNow, DefaultFreshnessConfig())
	if err != nil {
		t.Fatalf("CalculateFreshness() err = %v, want nil", err)
	}
	if got.Value <= 0.99 {
		t.Errorf("Value = %v, want > 0.99 for a 2-year-old use", got.Value)
	}
	if got.Value >= 1.0 {
		t.Errorf("Value = %v, want < 1.0 — a used track must never reach the never-used maximum", got.Value)
	}
}

// --- Normalization / boundedness ---

func TestCalculateFreshnessNormalizationBounds(t *testing.T) {
	for _, days := range []float64{0, 1, 7, 27, 28, 29, 35, 60, 180, 365, 730, 3650} {
		got, err := CalculateFreshness(daysBefore(freshnessNow, days), freshnessNow, DefaultFreshnessConfig())
		if err != nil {
			t.Fatalf("CalculateFreshness() err = %v, want nil", err)
		}
		if got.Value < 0 || got.Value > 1 {
			t.Errorf("days=%v: Value = %v, want within [0,1]", days, got.Value)
		}
	}
}

func TestCalculateFreshnessFutureLastUsedAtClampsToZero(t *testing.T) {
	future := freshnessNow.Add(24 * time.Hour)
	got, err := CalculateFreshness(&future, freshnessNow, DefaultFreshnessConfig())
	if err != nil {
		t.Fatalf("CalculateFreshness() err = %v, want nil", err)
	}
	if got.Value != 0 {
		t.Errorf("Value = %v, want 0 (clock skew clamped to t=0)", got.Value)
	}
}

// --- Monotonicity ---

func TestCalculateFreshnessMonotonicNondecreasing(t *testing.T) {
	days := []float64{0, 5, 15, 28, 35, 60, 120, 365, 1000}
	prev := -1.0
	for _, d := range days {
		got, err := CalculateFreshness(daysBefore(freshnessNow, d), freshnessNow, DefaultFreshnessConfig())
		if err != nil {
			t.Fatalf("CalculateFreshness() err = %v, want nil", err)
		}
		if got.Value < prev {
			t.Errorf("days=%v: Value = %v, want >= previous Value %v (must not decrease as time passes)", d, got.Value, prev)
		}
		prev = got.Value
	}
}

// --- Determinism ---

func TestCalculateFreshnessDeterministic(t *testing.T) {
	lastUsedAt := daysBefore(freshnessNow, 42)
	a, err := CalculateFreshness(lastUsedAt, freshnessNow, DefaultFreshnessConfig())
	if err != nil {
		t.Fatalf("CalculateFreshness() err = %v, want nil", err)
	}
	b, err := CalculateFreshness(lastUsedAt, freshnessNow, DefaultFreshnessConfig())
	if err != nil {
		t.Fatalf("CalculateFreshness() err = %v, want nil", err)
	}
	if a.Value != b.Value {
		t.Errorf("Value differs across identical calls: %v vs %v", a.Value, b.Value)
	}
}

// --- Boundary behaviour around the 28-day Recent Track Filter window ---

func TestCalculateFreshnessBoundaryAroundLookbackWindowNoDiscontinuity(t *testing.T) {
	d27, err := CalculateFreshness(daysBefore(freshnessNow, 27), freshnessNow, DefaultFreshnessConfig())
	if err != nil {
		t.Fatalf("CalculateFreshness() err = %v, want nil", err)
	}
	d28, err := CalculateFreshness(daysBefore(freshnessNow, 28), freshnessNow, DefaultFreshnessConfig())
	if err != nil {
		t.Fatalf("CalculateFreshness() err = %v, want nil", err)
	}
	d29, err := CalculateFreshness(daysBefore(freshnessNow, 29), freshnessNow, DefaultFreshnessConfig())
	if err != nil {
		t.Fatalf("CalculateFreshness() err = %v, want nil", err)
	}
	// A day either side of the hard 28-day exclusion boundary must change
	// Freshness only slightly — not jump from near-0 to 1.0. This is what
	// keeps Freshness from re-implementing the Recent Track Filter.
	if diff := d29.Value - d27.Value; diff < 0 || diff > 0.02 {
		t.Errorf("Value(27d)=%v Value(28d)=%v Value(29d)=%v, want a small, gradual change across the boundary", d27.Value, d28.Value, d29.Value)
	}
}

// --- FreshnessLastUsedAt ---

func TestFreshnessLastUsedAtEmptyOrMissingReturnsNil(t *testing.T) {
	history := map[string]time.Time{"track-1": freshnessNow}

	if got := FreshnessLastUsedAt(history, ""); got != nil {
		t.Errorf("FreshnessLastUsedAt(_, \"\") = %v, want nil", got)
	}
	if got := FreshnessLastUsedAt(history, "track-2"); got != nil {
		t.Errorf("FreshnessLastUsedAt(_, \"track-2\") = %v, want nil (absent from history)", got)
	}
}

func TestFreshnessLastUsedAtReturnsLatestAddedAt(t *testing.T) {
	want := freshnessNow.Add(-40 * 24 * time.Hour)
	history := map[string]time.Time{"track-1": want}

	got := FreshnessLastUsedAt(history, "track-1")
	if got == nil || !got.Equal(want) {
		t.Errorf("FreshnessLastUsedAt() = %v, want %v", got, want)
	}
}

// --- Independence from other candidate concepts ---
//
// CalculateFreshness takes only lastUsedAt/now/config — never a
// candidate.CandidateTrack — so CandidateType, Category, release date, and
// discovery provenance cannot affect Freshness by construction: the
// function has no parameter through which they could. This test is a
// regression guard on that call-site discipline, mirroring fit_test.go's
// idiom for CalculateFit.

func TestCalculateFreshnessIndependentOfCandidateClassification(t *testing.T) {
	lastUsedAt := daysBefore(freshnessNow, 45)
	config := DefaultFreshnessConfig()

	// Two calls standing in for candidates that differ only in
	// Type/Category/release date/provenance — none of which
	// CalculateFreshness can see.
	a, err := CalculateFreshness(lastUsedAt, freshnessNow, config)
	if err != nil {
		t.Fatalf("CalculateFreshness() err = %v, want nil", err)
	}
	b, err := CalculateFreshness(lastUsedAt, freshnessNow, config)
	if err != nil {
		t.Fatalf("CalculateFreshness() err = %v, want nil", err)
	}
	if a.Value != b.Value {
		t.Errorf("Freshness differs for identical playlist history alone: %v vs %v", a.Value, b.Value)
	}
}
