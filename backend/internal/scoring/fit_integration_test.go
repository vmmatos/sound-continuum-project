package scoring

import (
	"testing"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
	"github.com/vmmatos/sound-continuum-project/internal/musicaldna"
)

// TestFitIntegrationCandidatePoolToCandidateScore demonstrates Card #41's
// required end-to-end path without any live Spotify call: a candidate
// shaped like one that has already passed through Discovery -> Candidate
// Pool -> Recent Track Filter -> Metadata Enrichment -> Discovery
// Provenance (Cards #38/#39), a weekly musical direction a human curator has
// set, and Musical Fit calculated and exposed through CandidateScore
// (Card #40) with every other factor left untouched.
//
// No musical claim here is a real-world assertion about the fixture
// artist/track — Mood/Energy/Texture/CulturalInfluence are illustrative
// editorial tags for this test only, exactly the kind of explicit input
// CalculateFit requires and never fabricates on its own.
func TestFitIntegrationCandidatePoolToCandidateScore(t *testing.T) {
	// 1. A candidate as it exists after the existing pipeline stages.
	c, err := candidate.NewCandidateTrack(candidate.NewCandidateTrackParams{
		ID:             candidate.ID("cand-fit-integration"),
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
		}},
	})
	if err != nil {
		t.Fatalf("NewCandidateTrack() err = %v, want nil", err)
	}
	c.Metadata = &candidate.CandidateMetadata{
		Title:   c.TrackTitle,
		Artists: []candidate.CandidateArtist{{Name: c.TrackArtist}},
		Album: candidate.CandidateAlbum{
			Name:                 "Integration Fixture Album",
			ReleaseDate:          "2025-11-01",
			ReleaseDatePrecision: "day",
		},
	}

	// 2. A curator-supplied editorial tag for the candidate's musical
	// character (see musicaldna.Profile's doc comment: never derived from
	// Metadata above — supplied separately, on purpose).
	candidateProfile := musicaldna.Profile{
		Mood:    fitStrPtr("introspective"),
		Energy:  fitStrPtr("low"),
		Texture: fitStrPtr("organic"),
	}

	// 3. The current edition's explicit weekly direction.
	direction, err := musicaldna.NewWeeklyDirection(
		"2026-w40",
		musicaldna.Profile{
			Mood:    fitStrPtr("introspective"),
			Energy:  fitStrPtr("low"),
			Texture: fitStrPtr("electronic"), // deliberate partial mismatch
		},
		"late-night, downtempo arc",
	)
	if err != nil {
		t.Fatalf("NewWeeklyDirection() err = %v, want nil", err)
	}

	// 4. Fit calculation.
	fit, err := CalculateFit(candidateProfile, musicaldna.DefaultProjectDNA(), direction, DefaultFitWeights())
	if err != nil {
		t.Fatalf("CalculateFit() err = %v, want nil", err)
	}
	if fit.Value == nil {
		t.Fatal("Fit.Value = nil, want a computed value (Mood and Energy are available and match)")
	}
	// Mood and Energy match, Texture mismatches: partial, not saturated.
	if *fit.Value <= 0 || *fit.Value >= 1 {
		t.Fatalf("Fit.Value = %v, want a partial score reflecting the Texture mismatch", *fit.Value)
	}

	// 5. Fit populates CandidateScore.Factors.Fit; other factors remain nil.
	score, err := Calculate(c.ID, Factors{Fit: fit.Value}, DefaultWeights())
	if err != nil {
		t.Fatalf("Calculate() err = %v, want nil", err)
	}
	if score.Factors.Fit == nil || *score.Factors.Fit != *fit.Value {
		t.Errorf("CandidateScore.Factors.Fit = %v, want %v", score.Factors.Fit, fit.Value)
	}
	if score.Factors.Freshness != nil || score.Factors.DiscoveryBonus != nil ||
		score.Factors.Diversity != nil || score.Factors.PlaylistFit != nil ||
		score.Factors.RepetitionPenalty != nil {
		t.Errorf("Factors = %+v, want every non-Fit factor nil", score.Factors)
	}
	if score.FinalScore == nil {
		t.Error("FinalScore = nil, want a value derived from the available Fit factor")
	}

	// 6. Nothing about the candidate's lifecycle or the playlist changed.
	if c.Status != candidate.StatusDiscovered {
		t.Errorf("Status = %v, want unchanged StatusDiscovered", c.Status)
	}
}
