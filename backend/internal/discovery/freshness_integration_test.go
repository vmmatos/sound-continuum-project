package discovery

import (
	"context"
	"testing"

	"github.com/vmmatos/sound-continuum-project/internal/scoring"
	"github.com/vmmatos/sound-continuum-project/internal/spotify"
)

// TestFreshnessIntegrationDiscoveryHistoryFeedsScoring proves Card #42's
// central reuse requirement end-to-end: discovery.Service.PlaylistTrackHistory
// (Card #37's playlist-history retrieval) feeds directly into
// scoring.CalculateFreshness (Card #42) with no second retrieval
// mechanism in between.
func TestFreshnessIntegrationDiscoveryHistoryFeedsScoring(t *testing.T) {
	f := &fakeCatalogue{
		officialPlaylist: &spotify.OfficialPlaylist{SpotifyPlaylistID: testPlaylistID},
		playlistItems: map[string][]spotify.PlaylistItem{
			testPlaylistID: {trackItem("track-used", rfc3339(fixedNow.AddDate(0, 0, -35)))},
		},
	}
	svc := newTestSvcRecentTrackFilter(f, 28)

	history, err := svc.PlaylistTrackHistory(context.Background())
	if err != nil {
		t.Fatalf("PlaylistTrackHistory: %v", err)
	}

	usedFresh, err := scoring.CalculateFreshness(
		scoring.FreshnessLastUsedAt(history, "track-used"), fixedNow, scoring.DefaultFreshnessConfig())
	if err != nil {
		t.Fatalf("CalculateFreshness(used): %v", err)
	}
	neverUsedFresh, err := scoring.CalculateFreshness(
		scoring.FreshnessLastUsedAt(history, "track-never-used"), fixedNow, scoring.DefaultFreshnessConfig())
	if err != nil {
		t.Fatalf("CalculateFreshness(never used): %v", err)
	}

	if neverUsedFresh.Value != 1.0 {
		t.Errorf("never-used Freshness = %v, want 1.0", neverUsedFresh.Value)
	}
	if !(usedFresh.Value < neverUsedFresh.Value) {
		t.Errorf("used Freshness = %v, never-used Freshness = %v, want used < never-used", usedFresh.Value, neverUsedFresh.Value)
	}
}
