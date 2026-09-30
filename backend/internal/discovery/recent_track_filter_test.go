package discovery

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
	"github.com/vmmatos/sound-continuum-project/internal/spotify"
)

// fixedNow is the deterministic "current time" every FilterRecentTracks
// test uses via Service.now — Card #37 requires boundary behavior tested
// without depending on the real system clock.
var fixedNow = time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

const testPlaylistID = "official-playlist-id"

func newTestSvcRecentTrackFilter(f *fakeCatalogue, lookbackDays int) *Service {
	return &Service{spotify: f, recentTrackLookbackDays: lookbackDays, now: func() time.Time { return fixedNow }}
}

func newRecentTestCandidate(t *testing.T, id string, typ candidate.Type, cat candidate.Category) candidate.CandidateTrack {
	t.Helper()
	c, err := candidate.NewCandidateTrack(candidate.NewCandidateTrackParams{
		ID:             candidate.ID(id),
		SpotifyTrackID: id,
		Source:         candidate.SourceSpotify,
		Category:       cat,
		Type:           typ,
		TrackTitle:     "Song " + id,
		TrackArtist:    "Artist " + id,
	})
	if err != nil {
		t.Fatalf("newRecentTestCandidate(%q): %v", id, err)
	}
	return c
}

func trackItem(trackID, addedAt string) spotify.PlaylistItem {
	return spotify.PlaylistItem{AddedAt: addedAt, ItemType: "track", Track: &spotify.Track{ID: trackID}}
}

func rfc3339(t time.Time) string { return t.Format(time.RFC3339) }

func TestFilterRecentTracksEmptyPlaylist(t *testing.T) {
	f := &fakeCatalogue{officialPlaylist: &spotify.OfficialPlaylist{SpotifyPlaylistID: testPlaylistID}}
	svc := newTestSvcRecentTrackFilter(f, 28)
	c := newRecentTestCandidate(t, "track-1", candidate.TypeClassic, candidate.CategoryPast)

	result, err := svc.FilterRecentTracks(context.Background(), []candidate.CandidateTrack{c})
	if err != nil {
		t.Fatalf("FilterRecentTracks: %v", err)
	}
	if result.EligibleCount != 1 || result.RecentlyUsedCount != 0 {
		t.Errorf("result = %+v, want all candidates eligible against an empty playlist", result)
	}
	if len(result.EligibleCandidates) != 1 || result.EligibleCandidates[0].ID != c.ID {
		t.Errorf("EligibleCandidates = %+v", result.EligibleCandidates)
	}
}

func TestFilterRecentTracksRecentlyUsedExcluded(t *testing.T) {
	f := &fakeCatalogue{
		officialPlaylist: &spotify.OfficialPlaylist{SpotifyPlaylistID: testPlaylistID},
		playlistItems: map[string][]spotify.PlaylistItem{
			testPlaylistID: {trackItem("track-1", rfc3339(fixedNow.AddDate(0, 0, -5)))},
		},
	}
	svc := newTestSvcRecentTrackFilter(f, 28)
	c := newRecentTestCandidate(t, "track-1", candidate.TypeClassic, candidate.CategoryPast)

	result, err := svc.FilterRecentTracks(context.Background(), []candidate.CandidateTrack{c})
	if err != nil {
		t.Fatalf("FilterRecentTracks: %v", err)
	}
	if result.EligibleCount != 0 || result.RecentlyUsedCount != 1 {
		t.Fatalf("result = %+v, want the candidate recently used", result)
	}
	ru := result.RecentlyUsedCandidates[0]
	if ru.Candidate.ID != c.ID {
		t.Errorf("RecentlyUsedCandidates[0].Candidate = %+v, want %+v", ru.Candidate, c)
	}
	if ru.Reason != ReasonRecentlyUsed {
		t.Errorf("Reason = %q, want %q", ru.Reason, ReasonRecentlyUsed)
	}
	if want := fixedNow.AddDate(0, 0, -5); !ru.LastUsedAt.Equal(want) {
		t.Errorf("LastUsedAt = %v, want %v", ru.LastUsedAt, want)
	}
	if ru.Candidate.Status != candidate.StatusDiscovered {
		t.Errorf("Status = %q, want discovered — filtering is not an editorial rejection", ru.Candidate.Status)
	}
}

func TestFilterRecentTracksOlderTrackStaysEligible(t *testing.T) {
	f := &fakeCatalogue{
		officialPlaylist: &spotify.OfficialPlaylist{SpotifyPlaylistID: testPlaylistID},
		playlistItems: map[string][]spotify.PlaylistItem{
			testPlaylistID: {trackItem("track-1", rfc3339(fixedNow.AddDate(0, 0, -29)))},
		},
	}
	svc := newTestSvcRecentTrackFilter(f, 28)
	c := newRecentTestCandidate(t, "track-1", candidate.TypeClassic, candidate.CategoryPast)

	result, err := svc.FilterRecentTracks(context.Background(), []candidate.CandidateTrack{c})
	if err != nil {
		t.Fatalf("FilterRecentTracks: %v", err)
	}
	if result.EligibleCount != 1 || result.RecentlyUsedCount != 0 {
		t.Errorf("result = %+v, want the candidate eligible (29 days is outside a 28-day lookback)", result)
	}
}

func TestFilterRecentTracksExactBoundaryIsRecentlyUsed(t *testing.T) {
	f := &fakeCatalogue{
		officialPlaylist: &spotify.OfficialPlaylist{SpotifyPlaylistID: testPlaylistID},
		playlistItems: map[string][]spotify.PlaylistItem{
			// Exactly 28 days before fixedNow — the documented >= boundary.
			testPlaylistID: {trackItem("track-1", rfc3339(fixedNow.AddDate(0, 0, -28)))},
		},
	}
	svc := newTestSvcRecentTrackFilter(f, 28)
	c := newRecentTestCandidate(t, "track-1", candidate.TypeClassic, candidate.CategoryPast)

	result, err := svc.FilterRecentTracks(context.Background(), []candidate.CandidateTrack{c})
	if err != nil {
		t.Fatalf("FilterRecentTracks: %v", err)
	}
	if result.RecentlyUsedCount != 1 || result.EligibleCount != 0 {
		t.Errorf("result = %+v, want the candidate recently used at the exact boundary (>= rule)", result)
	}
}

func TestFilterRecentTracksDuplicatePlaylistEntriesKeepLatest(t *testing.T) {
	f := &fakeCatalogue{
		officialPlaylist: &spotify.OfficialPlaylist{SpotifyPlaylistID: testPlaylistID},
		playlistItems: map[string][]spotify.PlaylistItem{
			testPlaylistID: {
				trackItem("track-1", rfc3339(fixedNow.AddDate(0, 0, -40))), // older occurrence
				trackItem("track-1", rfc3339(fixedNow.AddDate(0, 0, -2))),  // newer occurrence
			},
		},
	}
	svc := newTestSvcRecentTrackFilter(f, 28)
	c := newRecentTestCandidate(t, "track-1", candidate.TypeClassic, candidate.CategoryPast)

	result, err := svc.FilterRecentTracks(context.Background(), []candidate.CandidateTrack{c})
	if err != nil {
		t.Fatalf("FilterRecentTracks: %v", err)
	}
	if result.RecentlyUsedCount != 1 {
		t.Fatalf("result = %+v, want the candidate recently used via the newer occurrence", result)
	}
	if want := fixedNow.AddDate(0, 0, -2); !result.RecentlyUsedCandidates[0].LastUsedAt.Equal(want) {
		t.Errorf("LastUsedAt = %v, want %v (the newer occurrence)", result.RecentlyUsedCandidates[0].LastUsedAt, want)
	}
	if result.PlaylistTracksInspected != 1 {
		t.Errorf("PlaylistTracksInspected = %d, want 1 (one distinct track, not two occurrences)",
			result.PlaylistTracksInspected)
	}
}

func TestFilterRecentTracksPagination(t *testing.T) {
	// playlistItemsPageSize is 50 — build 60 items so a full walk needs a
	// second page, with the recently-used track on that second page.
	items := make([]spotify.PlaylistItem, 0, 60)
	for i := 0; i < 59; i++ {
		items = append(items, trackItem(fmt.Sprintf("filler-%d", i), rfc3339(fixedNow.AddDate(0, 0, -100))))
	}
	items = append(items, trackItem("track-page-2", rfc3339(fixedNow.AddDate(0, 0, -1))))

	f := &fakeCatalogue{
		officialPlaylist: &spotify.OfficialPlaylist{SpotifyPlaylistID: testPlaylistID},
		playlistItems:    map[string][]spotify.PlaylistItem{testPlaylistID: items},
	}
	svc := newTestSvcRecentTrackFilter(f, 28)
	c := newRecentTestCandidate(t, "track-page-2", candidate.TypeClassic, candidate.CategoryPast)

	result, err := svc.FilterRecentTracks(context.Background(), []candidate.CandidateTrack{c})
	if err != nil {
		t.Fatalf("FilterRecentTracks: %v", err)
	}
	if result.RecentlyUsedCount != 1 {
		t.Fatalf("result = %+v, want the second-page track found as recently used", result)
	}
	if result.PlaylistTracksInspected != 60 {
		t.Errorf("PlaylistTracksInspected = %d, want 60 (all pages walked)", result.PlaylistTracksInspected)
	}
}

func TestFilterRecentTracksEpisodeItemsIgnored(t *testing.T) {
	f := &fakeCatalogue{
		officialPlaylist: &spotify.OfficialPlaylist{SpotifyPlaylistID: testPlaylistID},
		playlistItems: map[string][]spotify.PlaylistItem{
			testPlaylistID: {{
				AddedAt: rfc3339(fixedNow.AddDate(0, 0, -1)), ItemType: "episode",
				Episode: &spotify.Episode{ID: "ep-1"},
			}},
		},
	}
	svc := newTestSvcRecentTrackFilter(f, 28)
	c := newRecentTestCandidate(t, "track-1", candidate.TypeClassic, candidate.CategoryPast)

	result, err := svc.FilterRecentTracks(context.Background(), []candidate.CandidateTrack{c})
	if err != nil {
		t.Fatalf("FilterRecentTracks: %v", err)
	}
	if result.EligibleCount != 1 || result.PlaylistTracksInspected != 0 {
		t.Errorf("result = %+v, want the episode ignored entirely", result)
	}
}

func TestFilterRecentTracksUnavailableItemsIgnored(t *testing.T) {
	f := &fakeCatalogue{
		officialPlaylist: &spotify.OfficialPlaylist{SpotifyPlaylistID: testPlaylistID},
		playlistItems:    map[string][]spotify.PlaylistItem{testPlaylistID: {{ItemType: "unavailable"}}},
	}
	svc := newTestSvcRecentTrackFilter(f, 28)
	c := newRecentTestCandidate(t, "track-1", candidate.TypeClassic, candidate.CategoryPast)

	result, err := svc.FilterRecentTracks(context.Background(), []candidate.CandidateTrack{c})
	if err != nil {
		t.Fatalf("FilterRecentTracks: %v", err)
	}
	if result.EligibleCount != 1 || result.PlaylistTracksInspected != 0 {
		t.Errorf("result = %+v, want the unavailable item ignored entirely", result)
	}
}

func TestFilterRecentTracksCandidateWithNoSpotifyIDStaysEligible(t *testing.T) {
	f := &fakeCatalogue{
		officialPlaylist: &spotify.OfficialPlaylist{SpotifyPlaylistID: testPlaylistID},
		playlistItems: map[string][]spotify.PlaylistItem{
			testPlaylistID: {trackItem("track-1", rfc3339(fixedNow.AddDate(0, 0, -1)))},
		},
	}
	svc := newTestSvcRecentTrackFilter(f, 28)
	// A candidate with no Spotify track ID can never match a playlist
	// entry. Constructed directly (not via NewCandidateTrack, which
	// requires a non-empty SpotifyTrackID for the only Source this repo
	// supports today) to exercise FilterRecentTracks against this edge
	// input without inventing a fake identity.
	c := candidate.CandidateTrack{
		ID:          "manual-1",
		Source:      candidate.SourceSpotify,
		Category:    candidate.CategoryPast,
		Type:        candidate.TypeClassic,
		Status:      candidate.StatusDiscovered,
		TrackTitle:  "No Spotify ID Song",
		TrackArtist: "Unknown",
	}

	result, err := svc.FilterRecentTracks(context.Background(), []candidate.CandidateTrack{c})
	if err != nil {
		t.Fatalf("FilterRecentTracks: %v", err)
	}
	if result.EligibleCount != 1 || result.RecentlyUsedCount != 0 {
		t.Errorf("result = %+v, want the candidate with no Spotify ID to stay eligible", result)
	}
}

func TestFilterRecentTracksConsistentAcrossTypesAndCategories(t *testing.T) {
	f := &fakeCatalogue{
		officialPlaylist: &spotify.OfficialPlaylist{SpotifyPlaylistID: testPlaylistID},
		playlistItems: map[string][]spotify.PlaylistItem{
			testPlaylistID: {
				trackItem("track-classic", rfc3339(fixedNow.AddDate(0, 0, -1))),
				trackItem("track-current", rfc3339(fixedNow.AddDate(0, 0, -1))),
				trackItem("track-discovery", rfc3339(fixedNow.AddDate(0, 0, -1))),
			},
		},
	}
	svc := newTestSvcRecentTrackFilter(f, 28)
	candidates := []candidate.CandidateTrack{
		newRecentTestCandidate(t, "track-classic", candidate.TypeClassic, candidate.CategoryPast),
		newRecentTestCandidate(t, "track-current", candidate.TypeCurrent, candidate.CategoryPresent),
		newRecentTestCandidate(t, "track-discovery", candidate.TypeDiscovery, candidate.CategoryEmerging),
	}

	result, err := svc.FilterRecentTracks(context.Background(), candidates)
	if err != nil {
		t.Fatalf("FilterRecentTracks: %v", err)
	}
	if result.RecentlyUsedCount != 3 {
		t.Errorf("result = %+v, want all 3 candidates recently used regardless of Type/Category", result)
	}
}

func TestFilterRecentTracksPreservesProvenance(t *testing.T) {
	f := &fakeCatalogue{
		officialPlaylist: &spotify.OfficialPlaylist{SpotifyPlaylistID: testPlaylistID},
	}
	svc := newTestSvcRecentTrackFilter(f, 28)
	c := newRecentTestCandidate(t, "track-1", candidate.TypeClassic, candidate.CategoryPast)
	c.Provenance = []candidate.DiscoveryProvenance{{
		Method:   candidate.DiscoveryMethodClassicReferenceArtist,
		Provider: candidate.ProvenanceProviderSpotify,
		Seed:     &candidate.SeedArtist{Provider: candidate.ProvenanceProviderSpotify, ProviderArtistID: "a-1", Name: "Artist track-1"},
	}}

	result, err := svc.FilterRecentTracks(context.Background(), []candidate.CandidateTrack{c})
	if err != nil {
		t.Fatalf("FilterRecentTracks: %v", err)
	}
	if len(result.EligibleCandidates) != 1 || !reflect.DeepEqual(result.EligibleCandidates[0].Provenance, c.Provenance) {
		t.Errorf("FilterRecentTracks did not preserve Provenance: got %+v, want %+v", result.EligibleCandidates, c.Provenance)
	}
}

func TestFilterRecentTracksSpotifyFailureReturnsError(t *testing.T) {
	f := &fakeCatalogue{
		officialPlaylist: &spotify.OfficialPlaylist{SpotifyPlaylistID: testPlaylistID},
		playlistItemsErr: map[string]error{testPlaylistID: spotify.ErrAPIFailure},
	}
	svc := newTestSvcRecentTrackFilter(f, 28)
	c := newRecentTestCandidate(t, "track-1", candidate.TypeClassic, candidate.CategoryPast)

	result, err := svc.FilterRecentTracks(context.Background(), []candidate.CandidateTrack{c})
	if err == nil {
		t.Fatalf("FilterRecentTracks returned no error, want the Spotify failure surfaced; result = %+v", result)
	}
}

func TestFilterRecentTracksOfficialPlaylistNotConfiguredReturnsError(t *testing.T) {
	f := &fakeCatalogue{} // no official playlist persisted yet
	svc := newTestSvcRecentTrackFilter(f, 28)
	c := newRecentTestCandidate(t, "track-1", candidate.TypeClassic, candidate.CategoryPast)

	_, err := svc.FilterRecentTracks(context.Background(), []candidate.CandidateTrack{c})
	if !errors.Is(err, spotify.ErrOfficialPlaylistNotConfigured) {
		t.Fatalf("err = %v, want ErrOfficialPlaylistNotConfigured", err)
	}
}

func TestFilterRecentTracksAllRecentlyUsed(t *testing.T) {
	f := &fakeCatalogue{
		officialPlaylist: &spotify.OfficialPlaylist{SpotifyPlaylistID: testPlaylistID},
		playlistItems: map[string][]spotify.PlaylistItem{
			testPlaylistID: {
				trackItem("track-1", rfc3339(fixedNow.AddDate(0, 0, -1))),
				trackItem("track-2", rfc3339(fixedNow.AddDate(0, 0, -2))),
			},
		},
	}
	svc := newTestSvcRecentTrackFilter(f, 28)
	candidates := []candidate.CandidateTrack{
		newRecentTestCandidate(t, "track-1", candidate.TypeClassic, candidate.CategoryPast),
		newRecentTestCandidate(t, "track-2", candidate.TypeCurrent, candidate.CategoryPresent),
	}

	result, err := svc.FilterRecentTracks(context.Background(), candidates)
	if err != nil {
		t.Fatalf("FilterRecentTracks: %v", err)
	}
	if len(result.EligibleCandidates) != 0 {
		t.Errorf("EligibleCandidates = %+v, want none", result.EligibleCandidates)
	}
	if len(result.RecentlyUsedCandidates) != 2 {
		t.Errorf("RecentlyUsedCandidates = %+v, want 2", result.RecentlyUsedCandidates)
	}
}

// --- PlaylistTrackHistory ---
//
// These test the exported method directly — FilterRecentTracks' own tests
// above already exercise it indirectly, so this focuses on the reuse
// surface a future factor (e.g. scoring.Freshness, Card #42) depends on.

func TestPlaylistTrackHistoryDuplicateTrackKeepsLatest(t *testing.T) {
	f := &fakeCatalogue{
		officialPlaylist: &spotify.OfficialPlaylist{SpotifyPlaylistID: testPlaylistID},
		playlistItems: map[string][]spotify.PlaylistItem{
			testPlaylistID: {
				trackItem("track-1", rfc3339(fixedNow.AddDate(0, 0, -180))),
				trackItem("track-1", rfc3339(fixedNow.AddDate(0, 0, -40))),
			},
		},
	}
	svc := newTestSvcRecentTrackFilter(f, 28)

	history, err := svc.PlaylistTrackHistory(context.Background())
	if err != nil {
		t.Fatalf("PlaylistTrackHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("history = %+v, want 1 distinct track", history)
	}
	if want := fixedNow.AddDate(0, 0, -40); !history["track-1"].Equal(want) {
		t.Errorf("history[track-1] = %v, want %v (the newer occurrence)", history["track-1"], want)
	}
}

func TestPlaylistTrackHistoryEpisodesAndNullItemsIgnored(t *testing.T) {
	f := &fakeCatalogue{
		officialPlaylist: &spotify.OfficialPlaylist{SpotifyPlaylistID: testPlaylistID},
		playlistItems: map[string][]spotify.PlaylistItem{
			testPlaylistID: {
				{AddedAt: rfc3339(fixedNow.AddDate(0, 0, -1)), ItemType: "episode", Episode: &spotify.Episode{ID: "ep-1"}},
				{ItemType: "unavailable"},
				trackItem("track-1", rfc3339(fixedNow.AddDate(0, 0, -1))),
			},
		},
	}
	svc := newTestSvcRecentTrackFilter(f, 28)

	history, err := svc.PlaylistTrackHistory(context.Background())
	if err != nil {
		t.Fatalf("PlaylistTrackHistory: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("history = %+v, want only the one track item, episodes/unavailable ignored", history)
	}
}

func TestPlaylistTrackHistoryRetrievalFailureReturnsError(t *testing.T) {
	t.Run("official playlist not configured", func(t *testing.T) {
		f := &fakeCatalogue{}
		svc := newTestSvcRecentTrackFilter(f, 28)

		_, err := svc.PlaylistTrackHistory(context.Background())
		if !errors.Is(err, spotify.ErrOfficialPlaylistNotConfigured) {
			t.Fatalf("err = %v, want ErrOfficialPlaylistNotConfigured", err)
		}
	})

	t.Run("playlist items retrieval fails", func(t *testing.T) {
		f := &fakeCatalogue{
			officialPlaylist: &spotify.OfficialPlaylist{SpotifyPlaylistID: testPlaylistID},
			playlistItemsErr: map[string]error{testPlaylistID: spotify.ErrAPIFailure},
		}
		svc := newTestSvcRecentTrackFilter(f, 28)

		_, err := svc.PlaylistTrackHistory(context.Background())
		if !errors.Is(err, spotify.ErrAPIFailure) {
			t.Fatalf("err = %v, want ErrAPIFailure", err)
		}
	})
}

func TestFilterRecentTracksNoMatches(t *testing.T) {
	f := &fakeCatalogue{
		officialPlaylist: &spotify.OfficialPlaylist{SpotifyPlaylistID: testPlaylistID},
		playlistItems: map[string][]spotify.PlaylistItem{
			testPlaylistID: {trackItem("unrelated-track", rfc3339(fixedNow.AddDate(0, 0, -1)))},
		},
	}
	svc := newTestSvcRecentTrackFilter(f, 28)
	c := newRecentTestCandidate(t, "track-1", candidate.TypeClassic, candidate.CategoryPast)

	result, err := svc.FilterRecentTracks(context.Background(), []candidate.CandidateTrack{c})
	if err != nil {
		t.Fatalf("FilterRecentTracks: %v", err)
	}
	if result.EligibleCount != 1 || result.RecentlyUsedCount != 0 {
		t.Errorf("result = %+v, want the candidate eligible (no matching playlist track)", result)
	}
}
