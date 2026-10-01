package discovery

import (
	"context"
	"testing"

	"github.com/vmmatos/sound-continuum-project/internal/scoring"
	"github.com/vmmatos/sound-continuum-project/internal/spotify"
)

// TestRepetitionPenaltyIntegrationDiscoveryHistoryFeedsScoring proves
// Card #45's central reuse requirement end-to-end: discovery.Service's
// PlaylistTrackHistory and PlaylistArtistHistory (both built from Card
// #37's single playlist-history retrieval, extended in Card #45 — see
// playlistHistoryIndexes) feed directly into scoring.CalculateRepetitionPenalty
// with no second playlist-history mechanism in between.
func TestRepetitionPenaltyIntegrationDiscoveryHistoryFeedsScoring(t *testing.T) {
	f := &fakeCatalogue{
		officialPlaylist: &spotify.OfficialPlaylist{SpotifyPlaylistID: testPlaylistID},
		playlistItems: map[string][]spotify.PlaylistItem{
			testPlaylistID: {
				trackItemWithArtists("track-used", rfc3339(fixedNow.AddDate(0, 0, -10)), "artist-used"),
			},
		},
	}
	svc := newTestSvcRecentTrackFilter(f, 28)

	trackHistory, err := svc.PlaylistTrackHistory(context.Background())
	if err != nil {
		t.Fatalf("PlaylistTrackHistory: %v", err)
	}
	artistHistory, err := svc.PlaylistArtistHistory(context.Background())
	if err != nil {
		t.Fatalf("PlaylistArtistHistory: %v", err)
	}

	// Candidate 1: same track, same artist, used 10 days ago.
	trackLastUsedAt := scoring.FreshnessLastUsedAt(trackHistory, "track-used")
	artistLastUsedAt := scoring.RepetitionArtistLastUsedAt(artistHistory, []string{"artist-used"})
	used, err := scoring.CalculateRepetitionPenalty(trackLastUsedAt, artistLastUsedAt, fixedNow, scoring.DefaultRepetitionPenaltyConfig())
	if err != nil {
		t.Fatalf("CalculateRepetitionPenalty(used): %v", err)
	}

	// Candidate 2: a brand-new track by a brand-new artist.
	neverTrackLastUsedAt := scoring.FreshnessLastUsedAt(trackHistory, "track-never-used")
	neverArtistLastUsedAt := scoring.RepetitionArtistLastUsedAt(artistHistory, []string{"artist-never-used"})
	never, err := scoring.CalculateRepetitionPenalty(neverTrackLastUsedAt, neverArtistLastUsedAt, fixedNow, scoring.DefaultRepetitionPenaltyConfig())
	if err != nil {
		t.Fatalf("CalculateRepetitionPenalty(never used): %v", err)
	}

	if never.Value != 0 {
		t.Errorf("never-used penalty = %v, want 0", never.Value)
	}
	if used.Value <= 0 {
		t.Errorf("used penalty = %v, want > 0", used.Value)
	}

	// Candidate 3: a new track by the same recently-used artist — track
	// repetition must be 0 while artist repetition is positive.
	newTrackSameArtist, err := scoring.CalculateRepetitionPenalty(
		scoring.FreshnessLastUsedAt(trackHistory, "track-never-used"),
		scoring.RepetitionArtistLastUsedAt(artistHistory, []string{"artist-used"}),
		fixedNow, scoring.DefaultRepetitionPenaltyConfig())
	if err != nil {
		t.Fatalf("CalculateRepetitionPenalty(new track, used artist): %v", err)
	}
	if newTrackSameArtist.Track.Value != 0 {
		t.Errorf("Track.Value = %v, want 0 (track never used)", newTrackSameArtist.Track.Value)
	}
	if newTrackSameArtist.Artist.Value <= 0 {
		t.Errorf("Artist.Value = %v, want > 0 (artist used recently)", newTrackSameArtist.Artist.Value)
	}
}

// TestRepetitionPenaltyIntegrationEmptyPlaylistAllZero mirrors Card #45's
// explicit requirement: a successfully retrieved but genuinely empty
// official playlist means there is no historical repetition at all, so
// Repetition Penalty must be 0 for every candidate — distinct from an
// unavailable playlist, which must surface as an error instead.
func TestRepetitionPenaltyIntegrationEmptyPlaylistAllZero(t *testing.T) {
	f := &fakeCatalogue{officialPlaylist: &spotify.OfficialPlaylist{SpotifyPlaylistID: testPlaylistID}}
	svc := newTestSvcRecentTrackFilter(f, 28)

	trackHistory, err := svc.PlaylistTrackHistory(context.Background())
	if err != nil {
		t.Fatalf("PlaylistTrackHistory: %v", err)
	}
	artistHistory, err := svc.PlaylistArtistHistory(context.Background())
	if err != nil {
		t.Fatalf("PlaylistArtistHistory: %v", err)
	}

	got, err := scoring.CalculateRepetitionPenalty(
		scoring.FreshnessLastUsedAt(trackHistory, "track-a"),
		scoring.RepetitionArtistLastUsedAt(artistHistory, []string{"artist-a"}),
		fixedNow, scoring.DefaultRepetitionPenaltyConfig())
	if err != nil {
		t.Fatalf("CalculateRepetitionPenalty: %v", err)
	}
	if got.Value != 0 {
		t.Errorf("Value = %v, want 0 for an empty playlist history", got.Value)
	}
}

// TestRepetitionPenaltyIntegrationMissingHistoryPropagatesError confirms
// an unavailable official playlist surfaces as an error through
// PlaylistArtistHistory too, never a silently empty map that would read
// as "we verified there is no repetition" — the same error convention
// Card #37/#42 already established for PlaylistTrackHistory.
func TestRepetitionPenaltyIntegrationMissingHistoryPropagatesError(t *testing.T) {
	f := &fakeCatalogue{} // no officialPlaylist configured
	svc := newTestSvcRecentTrackFilter(f, 28)

	if _, err := svc.PlaylistArtistHistory(context.Background()); err == nil {
		t.Fatal("PlaylistArtistHistory() err = nil, want ErrOfficialPlaylistNotConfigured")
	}
}
