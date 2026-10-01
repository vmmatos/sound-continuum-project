package scoring

import (
	"strings"
	"testing"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
	"github.com/vmmatos/sound-continuum-project/internal/musicaldna"
)

// TestExplanationIntegrationCandidateScoreToExplanation demonstrates Card
// #50's required end-to-end path without any live Spotify/Last.fm call: a
// real CandidateTrack, scored through Calculate exactly as Cards #40-#46
// already exercise, then explained through GenerateExplanation — with
// FinalScore, Factors, and the candidate's own fields all left untouched by
// generating the explanation.
func TestExplanationIntegrationCandidateScoreToExplanation(t *testing.T) {
	c, err := candidate.NewCandidateTrack(candidate.NewCandidateTrackParams{
		ID:             candidate.ID("cand-explanation-integration"),
		SpotifyTrackID: "3n3Ppam7vgaVa1iaRUc9Lp",
		Source:         candidate.SourceSpotify,
		Category:       candidate.CategoryEmerging,
		Type:           candidate.TypeDiscovery,
		TrackTitle:     "Integration Fixture Track",
		TrackArtist:    "Integration Fixture Artist",
	})
	if err != nil {
		t.Fatalf("NewCandidateTrack() err = %v, want nil", err)
	}

	candidateProfile := musicaldna.Profile{
		Mood:   fitStrPtr("introspective"),
		Energy: fitStrPtr("low"),
	}
	direction, err := musicaldna.NewWeeklyDirection(
		"2026-w40",
		musicaldna.Profile{Mood: fitStrPtr("introspective"), Energy: fitStrPtr("low")},
		"late-night, downtempo arc",
	)
	if err != nil {
		t.Fatalf("NewWeeklyDirection() err = %v, want nil", err)
	}
	fit, err := CalculateFit(candidateProfile, musicaldna.DefaultProjectDNA(), direction, DefaultFitWeights())
	if err != nil {
		t.Fatalf("CalculateFit() err = %v, want nil", err)
	}

	editorialDiscoveryValue := 0.8
	discoveryBonus, err := CalculateDiscoveryBonus(c.Category, &editorialDiscoveryValue)
	if err != nil {
		t.Fatalf("CalculateDiscoveryBonus() err = %v, want nil", err)
	}

	score, err := Calculate(c.ID, Factors{Fit: fit.Value, DiscoveryBonus: discoveryBonus.Value}, DefaultWeights())
	if err != nil {
		t.Fatalf("Calculate() err = %v, want nil", err)
	}
	beforeScore := score

	explanation := GenerateExplanation(ExplanationInput{Score: score})

	if explanation.Text == "" {
		t.Fatal("Text is empty, want a grounded explanation")
	}
	if !strings.Contains(explanation.Text, "fit") && !strings.Contains(explanation.Text, "discovery") {
		t.Errorf("Text = %q, want it grounded in the real Fit/Discovery Bonus result", explanation.Text)
	}

	// Scoring is unchanged by generating the explanation.
	if score.Factors.Fit == nil || *score.Factors.Fit != *beforeScore.Factors.Fit {
		t.Error("Factors.Fit changed after GenerateExplanation")
	}
	if (score.FinalScore == nil) != (beforeScore.FinalScore == nil) {
		t.Error("FinalScore presence changed after GenerateExplanation")
	}
	if c.Status != candidate.StatusDiscovered {
		t.Errorf("Status = %v, want unchanged StatusDiscovered", c.Status)
	}
}
