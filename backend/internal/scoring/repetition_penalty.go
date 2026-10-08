package scoring

import (
	"math"
	"time"
)

// RepetitionPenaltyConfig holds Repetition Penalty's one editorial knob.
type RepetitionPenaltyConfig struct {
	// HorizonDays is how many days of playlist history still carry a
	// repetition penalty. Must be > 0.
	HorizonDays float64
}

// DefaultRepetitionHorizonDays is Card #45's chosen soft repetition
// horizon: 90 days. This is deliberately not the 28-day Recent Track
// Filter window (Card #37, a hard eligibility cutoff, not a penalty
// horizon) and not Freshness's 60-day half-life (Card #42, an asymptotic
// curve that never reaches zero) — Repetition Penalty needs a horizon
// where the penalty reaches exactly zero, so it is its own simple,
// explainable constant: long enough to give repetition memory real reach
// beyond the hard filter, short enough to avoid penalizing reuse
// indefinitely.
const DefaultRepetitionHorizonDays = 90

// DefaultRepetitionPenaltyConfig returns Sound Continuum's initial
// Repetition Penalty configuration.
func DefaultRepetitionPenaltyConfig() RepetitionPenaltyConfig {
	return RepetitionPenaltyConfig{HorizonDays: DefaultRepetitionHorizonDays}
}

// Validate checks that HorizonDays is a positive, non-NaN number.
func (c RepetitionPenaltyConfig) Validate() error {
	if math.IsNaN(c.HorizonDays) || c.HorizonDays <= 0 {
		return ErrRepetitionHorizonOutOfRange
	}
	return nil
}

// RepetitionComponentResult is one dimension's (track or artist)
// contribution to the Repetition Penalty, explainable independently of
// the other: whether it has ever appeared before, when, and the severity
// that produced.
type RepetitionComponentResult struct {
	Used             bool
	LastUsedAt       *time.Time
	TimeSinceLastUse *time.Duration
	Value            float64
}

// RepetitionPenaltyResult is the outcome of CalculateRepetitionPenalty.
// Value is never nil — Repetition Penalty is always computable, whether
// or not the candidate's track or artist has ever appeared before.
type RepetitionPenaltyResult struct {
	Value       float64
	Track       RepetitionComponentResult
	Artist      RepetitionComponentResult
	HorizonDays float64
}

// CalculateRepetitionPenalty computes the Repetition Penalty factor: does
// recent reuse of this exact track, or of its artist, warrant reducing
// this candidate's score? It is a soft scoring signal for candidates that
// already passed the Card #37 hard 28-day Recent Track Filter — never a
// second eligibility gate, never a blacklist. A track or artist can
// recur; this only measures how recently.
//
// trackLastUsedAt/artistLastUsedAt are nil when the track/artist has
// never appeared in the official playlist, which yields a severity of 0
// for that dimension. Otherwise each dimension decays linearly from 1.0
// at zero days since last use (the strongest case — consecutive reuse) to
// exactly 0 at config.HorizonDays and beyond:
//
//	severity(t) = 1 - t/HorizonDays   for 0 <= t < HorizonDays
//	severity(t) = 0                   for t >= HorizonDays
//
// The final Value is max(Track.Value, Artist.Value), not a sum — a track
// repetition event is already an artist repetition event, so summing
// would double-count the same historical fact. This keeps Value bounded
// in [0,1] by construction, with no clamping.
//
// now is a required parameter, never read from the system clock inside
// this function, so results are deterministic for a given input. If now
// is before a lastUsedAt (clock skew), t is clamped to 0 for that
// dimension, mirroring CalculateFreshness.
//
// config is validated first; an invalid config returns a zero-value
// RepetitionPenaltyResult and the matching error, with no partial
// computation.
func CalculateRepetitionPenalty(trackLastUsedAt, artistLastUsedAt *time.Time, now time.Time, config RepetitionPenaltyConfig) (RepetitionPenaltyResult, error) {
	if err := config.Validate(); err != nil {
		return RepetitionPenaltyResult{}, err
	}

	track := repetitionSeverity(trackLastUsedAt, now, config.HorizonDays)
	artist := repetitionSeverity(artistLastUsedAt, now, config.HorizonDays)

	value := max(track.Value, artist.Value)

	return RepetitionPenaltyResult{
		Value:       value,
		Track:       track,
		Artist:      artist,
		HorizonDays: config.HorizonDays,
	}, nil
}

func repetitionSeverity(lastUsedAt *time.Time, now time.Time, horizonDays float64) RepetitionComponentResult {
	if lastUsedAt == nil {
		return RepetitionComponentResult{}
	}

	since := now.Sub(*lastUsedAt)
	if since < 0 {
		since = 0
	}

	days := since.Hours() / 24
	var value float64
	if days < horizonDays {
		value = 1 - days/horizonDays
	}

	return RepetitionComponentResult{
		Used:             true,
		LastUsedAt:       lastUsedAt,
		TimeSinceLastUse: &since,
		Value:            value,
	}
}

// RepetitionArtistLastUsedAt looks up the most recent appearance, across
// any of spotifyArtistIDs, in artistHistory — a map of Spotify artist ID
// to its latest added_at in the official playlist (the shape
// discovery.Service.PlaylistArtistHistory returns). A candidate can carry
// more than one artist, so this returns the single most recent occurrence
// across all of them, or nil if none appear in artistHistory. Mirrors
// FreshnessLastUsedAt's role for the track dimension, taking a plain map
// so this package never needs to import discovery.
func RepetitionArtistLastUsedAt(artistHistory map[string]time.Time, spotifyArtistIDs []string) *time.Time {
	var mostRecent *time.Time
	for _, id := range spotifyArtistIDs {
		if id == "" {
			continue
		}
		addedAt, ok := artistHistory[id]
		if !ok {
			continue
		}
		if mostRecent == nil || addedAt.After(*mostRecent) {
			t := addedAt
			mostRecent = &t
		}
	}
	return mostRecent
}
