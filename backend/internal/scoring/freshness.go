package scoring

import (
	"math"
	"time"
)

// FreshnessConfig holds Freshness's one editorial knob.
type FreshnessConfig struct {
	// HalfLifeDays is how many days it takes a used track's Freshness to
	// recover halfway back toward (but never fully to) 1.0. Must be > 0.
	HalfLifeDays float64
}

// DefaultFreshnessHalfLifeDays is Card #42's chosen half-life: 60 days.
// The 28-day Recent Track Filter (Card #37) is a hard eligibility cutoff,
// not a freshness saturation point — a track used 35 days ago and one
// used 180 days ago are both eligible but must not score the same, so
// Freshness needs a longer horizon than the filter window. At 60 days:
// t=28d -> ~0.28, t=35d -> ~0.33, t=60d -> 0.5, t=180d -> ~0.875,
// t=730d -> ~0.9998 — a legible gradient with no discontinuity at the
// 28-day boundary, and old usage reads as "high but not maximal," distinct
// from a genuinely never-used candidate's exact 1.0.
const DefaultFreshnessHalfLifeDays = 60

// DefaultFreshnessConfig returns Sound Continuum's initial Freshness
// configuration.
func DefaultFreshnessConfig() FreshnessConfig {
	return FreshnessConfig{HalfLifeDays: DefaultFreshnessHalfLifeDays}
}

// Validate checks that HalfLifeDays is a positive, non-NaN number.
func (c FreshnessConfig) Validate() error {
	if math.IsNaN(c.HalfLifeDays) || c.HalfLifeDays <= 0 {
		return ErrFreshnessHalfLifeOutOfRange
	}
	return nil
}

// FreshnessResult is the outcome of CalculateFreshness: a normalized
// [0,1] Value plus the underlying LastUsedAt/TimeSinceLastUse it was
// derived from, so a low or maximal Freshness is always explainable
// rather than an opaque number. Unlike FitResult.Value, Value here is
// never nil — Freshness is always computable, whether or not the
// candidate has ever appeared in the official playlist.
type FreshnessResult struct {
	Value            float64
	LastUsedAt       *time.Time
	TimeSinceLastUse *time.Duration
}

// CalculateFreshness computes the Freshness factor: how long it has been
// since this candidate's Spotify track ID last appeared in the official
// Sound Continuum playlist. lastUsedAt is nil for a candidate that has
// never appeared, which yields the maximum Freshness (1.0) regardless of
// how old the track itself is — Freshness measures playlist-history
// recency, never release date, popularity, CandidateType, Category, or
// discovery provenance, none of which this function even accepts as
// input.
//
// For a used candidate, Freshness follows a half-life recovery curve:
// Value = 1 - 0.5^(t / config.HalfLifeDays), where t is the number of
// days between lastUsedAt and now. This is continuous and monotonically
// increasing with t, asymptotically approaching but never reaching 1.0 —
// deliberately distinct from the exact 1.0 a never-used candidate
// receives — and has no discontinuity at the 28-day Recent Track Filter
// boundary (Card #37): Freshness is a soft gradient beyond that hard
// eligibility cutoff, not a re-implementation of it.
//
// now is a required parameter, never read from the system clock inside
// this function, so results are deterministic for a given input. If now
// is before lastUsedAt (clock skew), t is clamped to 0 rather than
// producing a negative duration.
//
// config is validated first; an invalid config returns a zero-value
// FreshnessResult and the matching error, with no partial computation.
func CalculateFreshness(lastUsedAt *time.Time, now time.Time, config FreshnessConfig) (FreshnessResult, error) {
	if err := config.Validate(); err != nil {
		return FreshnessResult{}, err
	}

	if lastUsedAt == nil {
		return FreshnessResult{Value: 1.0}, nil
	}

	since := now.Sub(*lastUsedAt)
	if since < 0 {
		since = 0
	}

	days := since.Hours() / 24
	value := 1 - math.Pow(0.5, days/config.HalfLifeDays)

	return FreshnessResult{
		Value:            value,
		LastUsedAt:       lastUsedAt,
		TimeSinceLastUse: &since,
	}, nil
}

// FreshnessLastUsedAt looks up spotifyTrackID's most recent appearance in
// history — a map of Spotify track ID to its latest added_at in the
// official playlist, the exact shape discovery.Service.PlaylistTrackHistory
// (Card #37's playlist-history retrieval, reused rather than duplicated
// here) returns. It takes a plain map rather than a discovery-package type
// so this package never needs to import discovery, keeping
// CalculateFreshness's I/O-free purity intact. Returns nil (never used) for
// an empty spotifyTrackID or one absent from history.
func FreshnessLastUsedAt(history map[string]time.Time, spotifyTrackID string) *time.Time {
	if spotifyTrackID == "" {
		return nil
	}
	addedAt, ok := history[spotifyTrackID]
	if !ok {
		return nil
	}
	return &addedAt
}
