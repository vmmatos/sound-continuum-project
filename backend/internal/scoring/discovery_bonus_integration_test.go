package scoring

import (
	"testing"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
)

// TestDiscoveryBonusIntegrationCandidatePoolToCandidateScore demonstrates
// Card #43's required end-to-end path without any live Spotify/Last.fm
// call: an Emerging candidate shaped like one that has already passed
// through Discovery -> Candidate Pool -> Recent Track Filter -> Metadata
// Enrichment -> Discovery Provenance (Cards #38/#39), a human-supplied
// editorial discovery value, and Discovery Bonus calculated and exposed
// through CandidateScore (Card #40) with every other factor left untouched.
// The candidate's real discovery provenance stays available on
// c.Provenance for editorial explanation throughout — it is never passed
// into CalculateDiscoveryBonus, which only ever sees Category and the
// explicit editorial value.
func TestDiscoveryBonusIntegrationCandidatePoolToCandidateScore(t *testing.T) {
	match := 0.95
	c, err := candidate.NewCandidateTrack(candidate.NewCandidateTrackParams{
		ID:             candidate.ID("cand-discovery-bonus-integration"),
		SpotifyTrackID: "3n3Ppam7vgaVa1iaRUc9Lp",
		Source:         candidate.SourceSpotify,
		Category:       candidate.CategoryEmerging,
		Type:           candidate.TypeDiscovery,
		TrackTitle:     "Integration Fixture Track",
		TrackArtist:    "Integration Fixture Artist",
		Provenance: []candidate.DiscoveryProvenance{{
			Method:   candidate.DiscoveryMethodLastFMSimilarArtist,
			Provider: candidate.ProvenanceProviderLastFM,
			Seed:     &candidate.SeedArtist{Name: "Reference Artist"},
			DiscoveredArtist: &candidate.SeedArtist{
				Provider:         candidate.ProvenanceProviderSpotify,
				ProviderArtistID: "artist-id-123",
				Name:             "Integration Fixture Artist",
			},
			LastFMMatch: &match,
		}},
	})
	if err != nil {
		t.Fatalf("NewCandidateTrack() err = %v, want nil", err)
	}

	// A human editor has explicitly assessed this candidate's discovery
	// value — not derived from c.Provenance's LastFMMatch above, which
	// CalculateDiscoveryBonus never even receives.
	editorialValue := 0.8

	bonus, err := CalculateDiscoveryBonus(c.Category, &editorialValue)
	if err != nil {
		t.Fatalf("CalculateDiscoveryBonus() err = %v, want nil", err)
	}
	if bonus.Value == nil || *bonus.Value != editorialValue {
		t.Fatalf("DiscoveryBonusResult.Value = %v, want %v", bonus.Value, editorialValue)
	}

	score, err := Calculate(c.ID, Factors{DiscoveryBonus: bonus.Value}, DefaultWeights())
	if err != nil {
		t.Fatalf("Calculate() err = %v, want nil", err)
	}
	if score.Factors.DiscoveryBonus == nil || *score.Factors.DiscoveryBonus != editorialValue {
		t.Errorf("CandidateScore.Factors.DiscoveryBonus = %v, want %v", score.Factors.DiscoveryBonus, editorialValue)
	}
	if score.Factors.Fit != nil || score.Factors.Freshness != nil ||
		score.Factors.Diversity != nil || score.Factors.PlaylistFit != nil ||
		score.Factors.RepetitionPenalty != nil {
		t.Errorf("Factors = %+v, want every non-DiscoveryBonus factor nil", score.Factors)
	}
	if score.FinalScore == nil {
		t.Error("FinalScore = nil, want a value derived from the available Discovery Bonus factor")
	}

	// The candidate's real provenance is still there for a human/UI to
	// display alongside the bonus, even though it never fed the formula.
	if len(c.Provenance) != 1 || c.Provenance[0].LastFMMatch == nil {
		t.Errorf("c.Provenance = %+v, want the original Last.fm provenance untouched", c.Provenance)
	}
}

func TestDiscoveryBonusIntegrationNonEmergingCandidateStaysNilThroughScore(t *testing.T) {
	c, err := candidate.NewCandidateTrack(candidate.NewCandidateTrackParams{
		ID:             candidate.ID("cand-discovery-bonus-not-emerging"),
		SpotifyTrackID: "0VjIjW4GlUZAMYd2vXMi3b",
		Source:         candidate.SourceSpotify,
		Category:       candidate.CategoryPresent,
		Type:           candidate.TypeCurrent,
		TrackTitle:     "Non-Emerging Fixture Track",
		TrackArtist:    "Non-Emerging Fixture Artist",
	})
	if err != nil {
		t.Fatalf("NewCandidateTrack() err = %v, want nil", err)
	}

	// Even if an editorial value were mistakenly supplied, ineligibility
	// wins: the candidate is not the target of this factor.
	mistakenValue := 0.9
	bonus, err := CalculateDiscoveryBonus(c.Category, &mistakenValue)
	if err != nil {
		t.Fatalf("CalculateDiscoveryBonus() err = %v, want nil", err)
	}
	if bonus.Value != nil {
		t.Fatalf("DiscoveryBonusResult.Value = %v, want nil (not applicable for Category=%v)", *bonus.Value, c.Category)
	}

	score, err := Calculate(c.ID, Factors{DiscoveryBonus: bonus.Value}, DefaultWeights())
	if err != nil {
		t.Fatalf("Calculate() err = %v, want nil", err)
	}
	if score.Factors.DiscoveryBonus != nil {
		t.Errorf("Factors.DiscoveryBonus = %v, want nil", *score.Factors.DiscoveryBonus)
	}
}

func TestDefaultWeightsDiscoveryBonusUnchanged(t *testing.T) {
	if got := DefaultWeights().DiscoveryBonus; got != 0.15 {
		t.Errorf("DefaultWeights().DiscoveryBonus = %v, want 0.15 (Card #43 must not change the existing weight)", got)
	}
}

// TestDiscoveryBonusIntegrationMissingOtherFactorsRenormalizes confirms
// Discovery Bonus participates correctly in Calculate's existing
// missing-factor renormalization (Card #40), with zero changes to score.go.
func TestDiscoveryBonusIntegrationMissingOtherFactorsRenormalizes(t *testing.T) {
	weights := DefaultWeights()
	fitValue := 0.4
	bonusValue := 0.8

	score, err := Calculate("cand-renorm", Factors{Fit: &fitValue, DiscoveryBonus: &bonusValue}, weights)
	if err != nil {
		t.Fatalf("Calculate() err = %v, want nil", err)
	}

	wantAvailableWeight := weights.Fit + weights.DiscoveryBonus
	if diff := score.AvailableWeight - wantAvailableWeight; diff < -1e-9 || diff > 1e-9 {
		t.Errorf("AvailableWeight = %v, want %v", score.AvailableWeight, wantAvailableWeight)
	}

	wantFinal := (fitValue*weights.Fit + bonusValue*weights.DiscoveryBonus) / wantAvailableWeight
	if score.FinalScore == nil {
		t.Fatal("FinalScore = nil, want a computed value")
	}
	if diff := *score.FinalScore - wantFinal; diff < -1e-9 || diff > 1e-9 {
		t.Errorf("FinalScore = %v, want %v", *score.FinalScore, wantFinal)
	}
}
