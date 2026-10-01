package scoring

import (
	"testing"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
	"github.com/vmmatos/sound-continuum-project/internal/musicaldna"
)

// TestPlaylistFitIntegrationCandidateToCandidateScore demonstrates Card
// #46's required end-to-end path without any live Spotify call: a real
// candidate.CandidateTrack, shaped like one that has already passed
// through Discovery -> Candidate Pool -> Recent Track Filter -> Metadata
// Enrichment -> Discovery Provenance (Cards #38/#39), feeds
// CalculatePlaylistFit via values derived at the call site (never via
// CandidateTrack itself), and the result threads into CandidateScore
// (Card #40) with every other factor left untouched.
func TestPlaylistFitIntegrationCandidateToCandidateScore(t *testing.T) {
	c, err := candidate.NewCandidateTrack(candidate.NewCandidateTrackParams{
		ID:             candidate.ID("cand-playlist-fit-integration"),
		SpotifyTrackID: "2b3c4d5e6f7g8h9i0j1k2l",
		Source:         candidate.SourceSpotify,
		Category:       candidate.CategoryPresent,
		Type:           candidate.TypeCurrent,
		TrackTitle:     "Integration Fixture Track",
		TrackArtist:    "Integration Fixture Artist",
	})
	if err != nil {
		t.Fatalf("NewCandidateTrack() err = %v, want nil", err)
	}
	c.Metadata = &candidate.CandidateMetadata{
		Title:   c.TrackTitle,
		Artists: []candidate.CandidateArtist{{SpotifyArtistID: "artist-id-789", Name: c.TrackArtist}},
	}
	candidateSound := musicaldna.Profile{Mood: fitStrPtr("calm"), Energy: fitStrPtr("low")}

	// An edition already in progress — current-edition context, not the
	// official historical Spotify playlist. The last selected track is the
	// transition anchor.
	ctx := &CurrentEditionContext{Tracks: []EditionTrack{
		divTrack([]string{"artist-id-999"}, fitStrPtr("1990s"), musicaldna.Profile{Mood: fitStrPtr("tense"), Energy: fitStrPtr("high")}),
		divTrack([]string{"artist-id-111"}, fitStrPtr("2010s"), musicaldna.Profile{Mood: fitStrPtr("calm"), Energy: fitStrPtr("low")}),
	}}

	playlistFit := CalculatePlaylistFit(candidateSound, ctx)
	if playlistFit.Value == nil {
		t.Fatal("PlaylistFit.Value = nil, want a computed value")
	}
	if *playlistFit.Value != 1.0 {
		t.Errorf("PlaylistFit.Value = %v, want 1.0 (matches the immediately preceding track exactly)", *playlistFit.Value)
	}
	if playlistFit.PreviousTrackIndex == nil || *playlistFit.PreviousTrackIndex != 1 {
		t.Fatalf("PreviousTrackIndex = %v, want 1 (the second, most recent, edition track)", playlistFit.PreviousTrackIndex)
	}

	score, err := Calculate(c.ID, Factors{PlaylistFit: playlistFit.Value}, DefaultWeights())
	if err != nil {
		t.Fatalf("Calculate() err = %v, want nil", err)
	}
	if score.Factors.PlaylistFit == nil || *score.Factors.PlaylistFit != *playlistFit.Value {
		t.Errorf("CandidateScore.Factors.PlaylistFit = %v, want %v", score.Factors.PlaylistFit, playlistFit.Value)
	}
	if score.Factors.Fit != nil || score.Factors.Freshness != nil ||
		score.Factors.DiscoveryBonus != nil || score.Factors.Diversity != nil ||
		score.Factors.RepetitionPenalty != nil {
		t.Errorf("Factors = %+v, want every non-PlaylistFit factor nil", score.Factors)
	}
	if score.FinalScore == nil {
		t.Error("FinalScore = nil, want a value derived from the available PlaylistFit factor")
	}

	// The candidate's real Type/Category/Metadata are untouched and were
	// never passed to CalculatePlaylistFit.
	if c.Type != candidate.TypeCurrent || c.Category != candidate.CategoryPresent {
		t.Errorf("c.Type/Category = %v/%v, want unchanged Current/Present", c.Type, c.Category)
	}
	if c.Metadata == nil || len(c.Metadata.Artists) != 1 {
		t.Errorf("c.Metadata = %+v, want the original metadata untouched", c.Metadata)
	}
}

// --- 24. Existing weight unchanged ---

func TestDefaultWeightsPlaylistFitUnchanged(t *testing.T) {
	if got := DefaultWeights().PlaylistFit; got != 0.25 {
		t.Errorf("DefaultWeights().PlaylistFit = %v, want 0.25 (Card #46 must not change the existing weight)", got)
	}
}
