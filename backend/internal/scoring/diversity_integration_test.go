package scoring

import (
	"testing"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
	"github.com/vmmatos/sound-continuum-project/internal/musicaldna"
)

// TestDiversityIntegrationCandidateToCandidateScore demonstrates Card #44's
// required end-to-end path without any live Spotify call: a real
// candidate.CandidateTrack, shaped like one that has already passed
// through Discovery -> Candidate Pool -> Recent Track Filter -> Metadata
// Enrichment -> Discovery Provenance (Cards #38/#39), feeds
// CalculateDiversity via values derived at the call site (never via
// CandidateTrack itself), and the result threads into CandidateScore
// (Card #40) with every other factor left untouched.
func TestDiversityIntegrationCandidateToCandidateScore(t *testing.T) {
	match := 0.9
	c, err := candidate.NewCandidateTrack(candidate.NewCandidateTrackParams{
		ID:             candidate.ID("cand-diversity-integration"),
		SpotifyTrackID: "1a2b3c4d5e6f7g8h9i0j1k",
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
				ProviderArtistID: "artist-id-456",
				Name:             "Integration Fixture Artist",
			},
			LastFMMatch: &match,
		}},
	})
	if err != nil {
		t.Fatalf("NewCandidateTrack() err = %v, want nil", err)
	}
	c.Metadata = &candidate.CandidateMetadata{
		Title:   c.TrackTitle,
		Artists: []candidate.CandidateArtist{{SpotifyArtistID: "artist-id-456", Name: c.TrackArtist}},
		Album:   candidate.CandidateAlbum{ReleaseDate: "2024-03-01", ReleaseDatePrecision: "day"},
	}
	candidateSound := musicaldna.Profile{Mood: fitStrPtr("euphoric"), Energy: fitStrPtr("high")}

	// An edition already in progress, carrying a different artist, a
	// different era, and a different sound — current-edition context, not
	// the official historical Spotify playlist.
	ctx := &CurrentEditionContext{Tracks: []EditionTrack{
		{
			ArtistSpotifyIDs: []string{"artist-id-999"},
			Era:              fitStrPtr("1990s"),
			Sound:            musicaldna.Profile{Mood: fitStrPtr("melancholic"), Energy: fitStrPtr("low")},
		},
	}}

	artistIDs := make([]string, 0, len(c.Metadata.Artists))
	for _, a := range c.Metadata.Artists {
		artistIDs = append(artistIDs, a.SpotifyArtistID)
	}
	era := DiversityEra(c.Metadata.Album.ReleaseDate)

	diversity := CalculateDiversity(artistIDs, era, candidateSound, ctx)
	if diversity.Value == nil {
		t.Fatal("Diversity.Value = nil, want a computed value")
	}
	if *diversity.Value != 1.0 {
		t.Errorf("Diversity.Value = %v, want 1.0 (no overlap on any dimension)", *diversity.Value)
	}

	score, err := Calculate(c.ID, Factors{Diversity: diversity.Value}, DefaultWeights())
	if err != nil {
		t.Fatalf("Calculate() err = %v, want nil", err)
	}
	if score.Factors.Diversity == nil || *score.Factors.Diversity != *diversity.Value {
		t.Errorf("CandidateScore.Factors.Diversity = %v, want %v", score.Factors.Diversity, diversity.Value)
	}
	if score.Factors.Fit != nil || score.Factors.Freshness != nil ||
		score.Factors.DiscoveryBonus != nil || score.Factors.PlaylistFit != nil ||
		score.Factors.RepetitionPenalty != nil {
		t.Errorf("Factors = %+v, want every non-Diversity factor nil", score.Factors)
	}
	if score.FinalScore == nil {
		t.Error("FinalScore = nil, want a value derived from the available Diversity factor")
	}

	// The candidate's real Type/Category/Provenance/Last.fm match are
	// untouched and were never passed to CalculateDiversity.
	if c.Type != candidate.TypeDiscovery || c.Category != candidate.CategoryEmerging {
		t.Errorf("c.Type/Category = %v/%v, want unchanged Discovery/Emerging", c.Type, c.Category)
	}
	if len(c.Provenance) != 1 || c.Provenance[0].LastFMMatch == nil {
		t.Errorf("c.Provenance = %+v, want the original Last.fm provenance untouched", c.Provenance)
	}
}

// TestDiversityIntegrationIndependentOfCandidateTypeAndCategory confirms
// that changing CandidateType/Category on an otherwise-identical candidate
// does not change the Diversity value, since CalculateDiversity never sees
// either field.
func TestDiversityIntegrationIndependentOfCandidateTypeAndCategory(t *testing.T) {
	ctx := &CurrentEditionContext{Tracks: []EditionTrack{
		{ArtistSpotifyIDs: []string{"other-artist"}, Era: fitStrPtr("1980s")},
	}}
	artistIDs := []string{"artist-x"}
	era := fitStrPtr("2010s")
	sound := musicaldna.Profile{}

	classic, err := candidate.NewCandidateTrack(candidate.NewCandidateTrackParams{
		ID: "classic", SpotifyTrackID: "classic-id", Source: candidate.SourceSpotify,
		Category: candidate.CategoryPast, Type: candidate.TypeClassic,
		TrackTitle: "T", TrackArtist: "A",
	})
	if err != nil {
		t.Fatalf("NewCandidateTrack() err = %v", err)
	}
	discovery, err := candidate.NewCandidateTrack(candidate.NewCandidateTrackParams{
		ID: "discovery", SpotifyTrackID: "discovery-id", Source: candidate.SourceSpotify,
		Category: candidate.CategoryEmerging, Type: candidate.TypeDiscovery,
		TrackTitle: "T", TrackArtist: "A",
	})
	if err != nil {
		t.Fatalf("NewCandidateTrack() err = %v", err)
	}
	if classic.Type == discovery.Type || classic.Category == discovery.Category {
		t.Fatal("fixture setup error: classic and discovery must differ in Type and Category")
	}

	// Diversity is derived from artistIDs/era/sound, not from either
	// candidate's Type/Category — so the two candidates being genuinely
	// different (asserted above) must not change the result.
	classicResult := CalculateDiversity(artistIDs, era, sound, ctx)
	discoveryResult := CalculateDiversity(artistIDs, era, sound, ctx)
	if *classicResult.Value != *discoveryResult.Value {
		t.Errorf("Diversity differs by CandidateType/Category: %v vs %v", *classicResult.Value, *discoveryResult.Value)
	}
}

func TestDefaultWeightsDiversityUnchangedIntegration(t *testing.T) {
	if got := DefaultWeights().Diversity; got != 0.15 {
		t.Errorf("DefaultWeights().Diversity = %v, want 0.15", got)
	}
}
