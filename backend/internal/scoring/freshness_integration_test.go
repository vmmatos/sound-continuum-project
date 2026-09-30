package scoring

import (
	"testing"
	"time"
)

// TestFreshnessIntegrationHistoryToCandidateScore demonstrates Card #42's
// required end-to-end path without any live Spotify call: a playlist-
// history map shaped exactly like discovery.Service.PlaylistTrackHistory's
// return value (Card #37, reused rather than duplicated), a per-candidate
// lookup, Freshness calculation, and the result populating CandidateScore
// with every other factor left untouched.
func TestFreshnessIntegrationHistoryToCandidateScore(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

	// 1. Playlist history as discovery.Service.PlaylistTrackHistory would
	// return it: Spotify track ID -> most recent added_at.
	history := map[string]time.Time{
		"3n3Ppam7vgaVa1iaRUc9Lp": now.Add(-45 * 24 * time.Hour),
	}

	// 2. Look up this candidate's last appearance, then calculate Freshness.
	lastUsedAt := FreshnessLastUsedAt(history, "3n3Ppam7vgaVa1iaRUc9Lp")
	if lastUsedAt == nil {
		t.Fatal("FreshnessLastUsedAt() = nil, want a match")
	}

	fresh, err := CalculateFreshness(lastUsedAt, now, DefaultFreshnessConfig())
	if err != nil {
		t.Fatalf("CalculateFreshness() err = %v, want nil", err)
	}
	if fresh.Value <= 0 || fresh.Value >= 1 {
		t.Fatalf("Freshness.Value = %v, want a partial value for a 45-day-old use", fresh.Value)
	}

	// 3. Freshness populates CandidateScore.Factors.Freshness; other
	// factors remain nil.
	score, err := Calculate("cand-freshness-integration", Factors{Freshness: &fresh.Value}, DefaultWeights())
	if err != nil {
		t.Fatalf("Calculate() err = %v, want nil", err)
	}
	if score.Factors.Freshness == nil || *score.Factors.Freshness != fresh.Value {
		t.Errorf("CandidateScore.Factors.Freshness = %v, want %v", score.Factors.Freshness, fresh.Value)
	}
	if score.Factors.Fit != nil || score.Factors.DiscoveryBonus != nil ||
		score.Factors.Diversity != nil || score.Factors.PlaylistFit != nil ||
		score.Factors.RepetitionPenalty != nil {
		t.Errorf("Factors = %+v, want every non-Freshness factor nil", score.Factors)
	}
	if score.FinalScore == nil {
		t.Error("FinalScore = nil, want a value derived from the available Freshness factor")
	}
}

// TestFreshnessIntegrationEmptyPlaylistAllOne mirrors Card #42's explicit
// requirement: a successfully retrieved but genuinely empty official
// playlist means every candidate is effectively never used, so Freshness
// must be 1.0 for all of them — distinct from an unavailable playlist,
// which must surface as an error instead (see discovery package tests for
// that path, since retrieval itself lives there).
func TestFreshnessIntegrationEmptyPlaylistAllOne(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	history := map[string]time.Time{}

	for _, trackID := range []string{"track-a", "track-b", ""} {
		lastUsedAt := FreshnessLastUsedAt(history, trackID)
		fresh, err := CalculateFreshness(lastUsedAt, now, DefaultFreshnessConfig())
		if err != nil {
			t.Fatalf("CalculateFreshness() err = %v, want nil", err)
		}
		if fresh.Value != 1.0 {
			t.Errorf("trackID=%q: Value = %v, want 1.0 for an empty playlist history", trackID, fresh.Value)
		}
	}
}

// TestDefaultWeightsFreshnessUnchanged guards Card #40's 0.10 Freshness
// weight against an accidental change while wiring the calculator in.
func TestDefaultWeightsFreshnessUnchanged(t *testing.T) {
	if got := DefaultWeights().Freshness; got != 0.10 {
		t.Errorf("DefaultWeights().Freshness = %v, want 0.10", got)
	}
}
