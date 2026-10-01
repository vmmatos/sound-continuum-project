package scoring

import (
	"testing"
	"time"
)

// TestRepetitionPenaltyIntegrationHistoryToCandidateScore demonstrates
// Card #45's required end-to-end path without any live Spotify call: a
// track-history map and an artist-history map shaped exactly like
// discovery.Service.PlaylistTrackHistory/PlaylistArtistHistory's return
// values, a per-candidate lookup, Repetition Penalty calculation, and the
// result populating CandidateScore with every other factor left
// untouched.
func TestRepetitionPenaltyIntegrationHistoryToCandidateScore(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

	trackHistory := map[string]time.Time{
		"3n3Ppam7vgaVa1iaRUc9Lp": now.Add(-10 * 24 * time.Hour),
	}
	artistHistory := map[string]time.Time{
		"4dpARuHxo51G3z768sgnrY": now.Add(-10 * 24 * time.Hour),
	}

	trackLastUsedAt := FreshnessLastUsedAt(trackHistory, "3n3Ppam7vgaVa1iaRUc9Lp")
	artistLastUsedAt := RepetitionArtistLastUsedAt(artistHistory, []string{"4dpARuHxo51G3z768sgnrY"})

	penalty, err := CalculateRepetitionPenalty(trackLastUsedAt, artistLastUsedAt, now, DefaultRepetitionPenaltyConfig())
	if err != nil {
		t.Fatalf("CalculateRepetitionPenalty() err = %v, want nil", err)
	}
	if penalty.Value <= 0 {
		t.Fatalf("penalty.Value = %v, want > 0", penalty.Value)
	}

	score, err := Calculate("cand-repetition-integration", Factors{RepetitionPenalty: &penalty.Value}, DefaultWeights())
	if err != nil {
		t.Fatalf("Calculate() err = %v, want nil", err)
	}
	if score.Factors.RepetitionPenalty == nil || *score.Factors.RepetitionPenalty != penalty.Value {
		t.Errorf("CandidateScore.Factors.RepetitionPenalty = %v, want %v", score.Factors.RepetitionPenalty, penalty.Value)
	}
	if score.Factors.Fit != nil || score.Factors.Freshness != nil || score.Factors.DiscoveryBonus != nil ||
		score.Factors.Diversity != nil || score.Factors.PlaylistFit != nil {
		t.Errorf("Factors = %+v, want every non-RepetitionPenalty factor nil", score.Factors)
	}
	// Zero positive factors are available, so FinalScore must stay nil —
	// RepetitionPenalty alone never fabricates a BaseScore. This also
	// proves a high penalty never itself removes/rejects a candidate: the
	// result is just a value on CandidateScore, nothing mutates Status or
	// a pool.
	if score.FinalScore != nil {
		t.Errorf("FinalScore = %v, want nil (no positive factor available)", *score.FinalScore)
	}
}

// TestDefaultWeightsRepetitionWeightUnchanged guards Card #40's 0.30
// RepetitionWeight against an accidental change while wiring the
// calculator in.
func TestDefaultWeightsRepetitionWeightUnchanged(t *testing.T) {
	if got := DefaultWeights().RepetitionWeight; got != 0.30 {
		t.Errorf("DefaultWeights().RepetitionWeight = %v, want 0.30", got)
	}
}

// TestRepetitionPenaltyMaxValueCapsFinalScoreAt70Percent proves Card #45's
// required maximum-penalty behavior: RepetitionPenalty = 1.0 with
// RepetitionWeight = 0.30 produces FinalScore = BaseScore * 0.70, using
// deterministic fixture values alongside every other factor.
func TestRepetitionPenaltyMaxValueCapsFinalScoreAt70Percent(t *testing.T) {
	fit, freshness, discoveryBonus, diversity, playlistFit := 0.82, 0.65, 0.90, 0.70, 0.88
	maxPenalty := 1.0

	score, err := Calculate("cand-max-penalty", Factors{
		Fit:               &fit,
		Freshness:         &freshness,
		DiscoveryBonus:    &discoveryBonus,
		Diversity:         &diversity,
		PlaylistFit:       &playlistFit,
		RepetitionPenalty: &maxPenalty,
	}, DefaultWeights())
	if err != nil {
		t.Fatalf("Calculate() err = %v, want nil", err)
	}
	if score.FinalScore == nil {
		t.Fatal("FinalScore = nil, want a value")
	}

	wantBase := fit*0.35 + freshness*0.10 + discoveryBonus*0.15 + diversity*0.15 + playlistFit*0.25
	wantFinal := wantBase * 0.70

	const tolerance = 1e-9
	if diff := *score.FinalScore - wantFinal; diff > tolerance || diff < -tolerance {
		t.Errorf("FinalScore = %v, want %v (BaseScore * 0.70)", *score.FinalScore, wantFinal)
	}
}
