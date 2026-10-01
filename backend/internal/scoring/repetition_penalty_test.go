package scoring

import (
	"errors"
	"math"
	"testing"
	"time"
)

var repetitionNow = time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

func repetitionDaysBefore(now time.Time, days float64) *time.Time {
	t := now.Add(-time.Duration(days * float64(24*time.Hour)))
	return &t
}

// --- RepetitionPenaltyConfig ---

func TestDefaultRepetitionPenaltyConfigValid(t *testing.T) {
	if err := DefaultRepetitionPenaltyConfig().Validate(); err != nil {
		t.Fatalf("DefaultRepetitionPenaltyConfig().Validate() = %v, want nil", err)
	}
}

func TestRepetitionPenaltyConfigHorizonOutOfRange(t *testing.T) {
	for _, v := range []float64{0, -1, -90} {
		c := RepetitionPenaltyConfig{HorizonDays: v}
		if err := c.Validate(); !errors.Is(err, ErrRepetitionHorizonOutOfRange) {
			t.Errorf("HorizonDays=%v: err = %v, want ErrRepetitionHorizonOutOfRange", v, err)
		}
	}
}

func TestRepetitionPenaltyConfigHorizonNaNRejected(t *testing.T) {
	c := RepetitionPenaltyConfig{HorizonDays: math.NaN()}
	if err := c.Validate(); !errors.Is(err, ErrRepetitionHorizonOutOfRange) {
		t.Errorf("err = %v, want ErrRepetitionHorizonOutOfRange", err)
	}
}

func TestCalculateRepetitionPenaltyInvalidConfigReturnsError(t *testing.T) {
	_, err := CalculateRepetitionPenalty(nil, nil, repetitionNow, RepetitionPenaltyConfig{HorizonDays: 0})
	if !errors.Is(err, ErrRepetitionHorizonOutOfRange) {
		t.Fatalf("err = %v, want ErrRepetitionHorizonOutOfRange", err)
	}
}

// --- 1. Never-used track and artist ---

func TestCalculateRepetitionPenaltyNeverUsedIsZero(t *testing.T) {
	got, err := CalculateRepetitionPenalty(nil, nil, repetitionNow, DefaultRepetitionPenaltyConfig())
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got.Value != 0 {
		t.Errorf("Value = %v, want 0", got.Value)
	}
	if got.Track.Used || got.Artist.Used {
		t.Errorf("Track.Used = %v, Artist.Used = %v, want both false", got.Track.Used, got.Artist.Used)
	}
}

// --- 2. Recently-used track ---

func TestCalculateRepetitionPenaltyRecentTrackIsPositive(t *testing.T) {
	trackUsed := repetitionDaysBefore(repetitionNow, 5)
	got, err := CalculateRepetitionPenalty(trackUsed, nil, repetitionNow, DefaultRepetitionPenaltyConfig())
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got.Track.Value <= 0 {
		t.Errorf("Track.Value = %v, want > 0", got.Track.Value)
	}
	if got.Value <= 0 {
		t.Errorf("Value = %v, want > 0", got.Value)
	}
}

// --- 3. Older track reuse (outside the 28-day hard filter, inside the 90-day horizon) ---

func TestCalculateRepetitionPenaltyOlderTrackWeakerThanRecent(t *testing.T) {
	recent := repetitionDaysBefore(repetitionNow, 5)
	older := repetitionDaysBefore(repetitionNow, 45) // outside Card #37's 28-day filter

	recentResult, err := CalculateRepetitionPenalty(recent, nil, repetitionNow, DefaultRepetitionPenaltyConfig())
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	olderResult, err := CalculateRepetitionPenalty(older, nil, repetitionNow, DefaultRepetitionPenaltyConfig())
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}

	if olderResult.Track.Value <= 0 {
		t.Errorf("older Track.Value = %v, want > 0 (still inside the 90-day horizon)", olderResult.Track.Value)
	}
	if !(olderResult.Track.Value < recentResult.Track.Value) {
		t.Errorf("older Track.Value = %v, recent Track.Value = %v, want older < recent", olderResult.Track.Value, recentResult.Track.Value)
	}
}

// --- 4. Outside the repetition horizon ---

func TestCalculateRepetitionPenaltyOutsideHorizonIsZero(t *testing.T) {
	longAgo := repetitionDaysBefore(repetitionNow, 120) // beyond the default 90-day horizon
	got, err := CalculateRepetitionPenalty(longAgo, nil, repetitionNow, DefaultRepetitionPenaltyConfig())
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got.Track.Value != 0 {
		t.Errorf("Track.Value = %v, want 0 beyond the horizon", got.Track.Value)
	}
	if !got.Track.Used {
		t.Error("Track.Used = false, want true (it did appear, just outside the soft horizon)")
	}
}

// --- 5. New track by recently-used artist ---

func TestCalculateRepetitionPenaltyNewTrackRecentArtist(t *testing.T) {
	artistUsed := repetitionDaysBefore(repetitionNow, 10)
	got, err := CalculateRepetitionPenalty(nil, artistUsed, repetitionNow, DefaultRepetitionPenaltyConfig())
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got.Track.Value != 0 {
		t.Errorf("Track.Value = %v, want 0 (track never used)", got.Track.Value)
	}
	if got.Artist.Value <= 0 {
		t.Errorf("Artist.Value = %v, want > 0", got.Artist.Value)
	}
	if got.Value <= 0 {
		t.Errorf("Value = %v, want > 0", got.Value)
	}
}

// --- 6. New track, never-used artist ---

func TestCalculateRepetitionPenaltyNewTrackNewArtistIsZero(t *testing.T) {
	got, err := CalculateRepetitionPenalty(nil, nil, repetitionNow, DefaultRepetitionPenaltyConfig())
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got.Track.Value != 0 || got.Artist.Value != 0 || got.Value != 0 {
		t.Errorf("got = %+v, want all zero", got)
	}
}

// --- 7. Recently-used track and artist (max combination, no double-count) ---

func TestCalculateRepetitionPenaltyCombinesByMax(t *testing.T) {
	trackUsed := repetitionDaysBefore(repetitionNow, 60)  // weaker severity
	artistUsed := repetitionDaysBefore(repetitionNow, 10) // stronger severity

	got, err := CalculateRepetitionPenalty(trackUsed, artistUsed, repetitionNow, DefaultRepetitionPenaltyConfig())
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got.Track.Value <= 0 || got.Artist.Value <= 0 {
		t.Fatalf("got = %+v, want both dimensions positive", got)
	}
	if got.Artist.Value <= got.Track.Value {
		t.Fatalf("test setup invalid: want Artist.Value > Track.Value, got %+v", got)
	}
	if got.Value != got.Artist.Value {
		t.Errorf("Value = %v, want max() = Artist.Value = %v", got.Value, got.Artist.Value)
	}
	if got.Value == got.Track.Value+got.Artist.Value {
		t.Error("Value equals Track.Value + Artist.Value — looks summed, not maxed")
	}
}

// --- 8. Consecutive track reuse ---

func TestCalculateRepetitionPenaltyConsecutiveIsStrongest(t *testing.T) {
	justUsed := repetitionDaysBefore(repetitionNow, 0)
	got, err := CalculateRepetitionPenalty(justUsed, nil, repetitionNow, DefaultRepetitionPenaltyConfig())
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got.Track.Value != 1.0 {
		t.Errorf("Track.Value = %v, want 1.0 for consecutive reuse (day 0)", got.Track.Value)
	}
	if got.Value != 1.0 {
		t.Errorf("Value = %v, want 1.0", got.Value)
	}
}

// --- Clock skew ---

func TestCalculateRepetitionPenaltyClockSkewClamped(t *testing.T) {
	future := repetitionNow.Add(24 * time.Hour)
	got, err := CalculateRepetitionPenalty(&future, nil, repetitionNow, DefaultRepetitionPenaltyConfig())
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got.Track.Value != 1.0 {
		t.Errorf("Track.Value = %v, want 1.0 (clock skew clamped to 0 days)", got.Track.Value)
	}
}

// --- Normalization bounds ---

func TestCalculateRepetitionPenaltyBounded(t *testing.T) {
	for _, days := range []float64{0, 1, 28, 45, 89.99, 90, 91, 500} {
		lastUsed := repetitionDaysBefore(repetitionNow, days)
		got, err := CalculateRepetitionPenalty(lastUsed, lastUsed, repetitionNow, DefaultRepetitionPenaltyConfig())
		if err != nil {
			t.Fatalf("days=%v: err = %v, want nil", days, err)
		}
		if got.Value < 0 || got.Value > 1 {
			t.Errorf("days=%v: Value = %v, want in [0,1]", days, got.Value)
		}
	}
}

// --- Determinism ---

func TestCalculateRepetitionPenaltyDeterministic(t *testing.T) {
	trackUsed := repetitionDaysBefore(repetitionNow, 15)
	artistUsed := repetitionDaysBefore(repetitionNow, 50)
	a, err := CalculateRepetitionPenalty(trackUsed, artistUsed, repetitionNow, DefaultRepetitionPenaltyConfig())
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	b, err := CalculateRepetitionPenalty(trackUsed, artistUsed, repetitionNow, DefaultRepetitionPenaltyConfig())
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if a.Value != b.Value {
		t.Errorf("a.Value = %v, b.Value = %v, want equal for identical inputs", a.Value, b.Value)
	}
}

// --- Independence from classification/other factors: proven by construction —
// CalculateRepetitionPenalty accepts only *time.Time/time.Time/config, never
// a candidate.CandidateTrack, CandidateType, Category, or any other factor's
// value, so none of them can influence it. This mirrors the regression-guard
// style already used for Fit/Freshness/Discovery Bonus/Diversity.

// --- RepetitionArtistLastUsedAt ---

func TestRepetitionArtistLastUsedAtEmptyIDsReturnsNil(t *testing.T) {
	history := map[string]time.Time{"artist-1": repetitionNow}
	if got := RepetitionArtistLastUsedAt(history, nil); got != nil {
		t.Errorf("got = %v, want nil", got)
	}
	if got := RepetitionArtistLastUsedAt(history, []string{""}); got != nil {
		t.Errorf("got = %v, want nil", got)
	}
}

func TestRepetitionArtistLastUsedAtNoMatchReturnsNil(t *testing.T) {
	history := map[string]time.Time{"artist-1": repetitionNow}
	if got := RepetitionArtistLastUsedAt(history, []string{"artist-2"}); got != nil {
		t.Errorf("got = %v, want nil", got)
	}
}

func TestRepetitionArtistLastUsedAtMultiArtistPicksMostRecent(t *testing.T) {
	older := repetitionNow.AddDate(0, 0, -40)
	newer := repetitionNow.AddDate(0, 0, -5)
	history := map[string]time.Time{
		"artist-older": older,
		"artist-newer": newer,
	}
	got := RepetitionArtistLastUsedAt(history, []string{"artist-older", "artist-newer", "artist-absent"})
	if got == nil || !got.Equal(newer) {
		t.Errorf("got = %v, want %v", got, newer)
	}
}
