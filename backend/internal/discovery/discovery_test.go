package discovery

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
	"github.com/vmmatos/sound-continuum-project/internal/spotify"
)

// fakeCatalogue is an in-memory stand-in for spotify.Service, implementing
// exactly the spotifyCatalogue seam DiscoverClassic depends on. No HTTP
// server or database is needed — spotify.Service has no exported hook to
// redirect its internal client to a test server, so this package tests
// against the seam it depends on instead.
type fakeCatalogue struct {
	// artists maps a search query to the artists Search should return.
	artists map[string][]spotify.Artist
	// albums maps an artist ID to that artist's albums, in page order.
	albums map[string][]spotify.Album
	// tracks maps an album ID to that album's tracks, in page order.
	tracks map[string][]spotify.Track

	searchErr map[string]error // query -> error, checked before artists lookup
	albumsErr map[string]error // artistID -> error
	tracksErr map[string]error // albumID -> error

	searchCalls int
}

func (f *fakeCatalogue) Search(ctx context.Context, query, types string, limit, offset int) (spotify.SearchResult, error) {
	f.searchCalls++
	if err, ok := f.searchErr[query]; ok {
		return spotify.SearchResult{}, err
	}
	items := f.artists[query]
	return spotify.SearchResult{Artists: &spotify.Paging[spotify.Artist]{Items: items, Total: len(items)}}, nil
}

func (f *fakeCatalogue) ArtistAlbums(ctx context.Context, artistID string, limit, offset int) (spotify.Paging[spotify.Album], error) {
	if err, ok := f.albumsErr[artistID]; ok {
		return spotify.Paging[spotify.Album]{}, err
	}
	return page(f.albums[artistID], limit, offset), nil
}

func (f *fakeCatalogue) AlbumTracks(ctx context.Context, albumID string, limit, offset int) (spotify.Paging[spotify.Track], error) {
	if err, ok := f.tracksErr[albumID]; ok {
		return spotify.Paging[spotify.Track]{}, err
	}
	return page(f.tracks[albumID], limit, offset), nil
}

// page slices items[offset:offset+limit], mimicking Spotify's own paging
// behavior (including returning fewer than limit on the last page).
func page[T any](items []T, limit, offset int) spotify.Paging[T] {
	if offset >= len(items) {
		return spotify.Paging[T]{Total: len(items), Limit: limit, Offset: offset}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return spotify.Paging[T]{Items: items[offset:end], Total: len(items), Limit: limit, Offset: offset}
}

func newTestSvc(f *fakeCatalogue, cfg Config) *Service {
	return &Service{spotify: f, classicCfg: cfg}
}

func testConfig() Config {
	return Config{MaxAlbumsPerArtist: 5, MaxTracksPerAlbum: 10, MaxTotalCandidates: 100}
}

func newTestSvcCurrent(f *fakeCatalogue, cfg CurrentConfig) *Service {
	return &Service{spotify: f, currentCfg: cfg}
}

func testCurrentConfig() CurrentConfig {
	return CurrentConfig{
		LookbackDays:              90,
		MaxAlbumsScannedPerArtist: 50,
		MaxAlbumsPerArtist:        5,
		MaxTracksPerAlbum:         10,
		MaxTotalCandidates:        100,
	}
}

// daysAgo formats a time.Time n days before now as a full "day"-precision
// Spotify release_date.
func daysAgo(n int) string {
	return time.Now().AddDate(0, 0, -n).Format("2006-01-02")
}

func TestDiscoverClassicResolvesExactNameMatch(t *testing.T) {
	f := &fakeCatalogue{
		artists: map[string][]spotify.Artist{
			"David Bowie": {{ID: "wrong", Name: "David Bowie Tribute"}, {ID: "a-bowie", Name: "David Bowie"}},
		},
		albums: map[string][]spotify.Album{"a-bowie": {{ID: "al-1", Name: "Low"}}},
		tracks: map[string][]spotify.Track{"al-1": {{ID: "t-1", Name: "Sound and Vision"}}},
	}
	svc := newTestSvc(f, testConfig())
	// DiscoverClassic always walks the full PastReferenceArtists list;
	// only "David Bowie" is seeded, so every other artist resolves as
	// unresolved (asserted separately) and contributes no candidates.
	result, err := svc.DiscoverClassic(context.Background())
	if err != nil {
		t.Fatalf("DiscoverClassic returned error: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d: %+v", len(result.Candidates), result.Candidates)
	}
	c := result.Candidates[0]
	if c.SpotifyTrackID != "t-1" || c.TrackTitle != "Sound and Vision" || c.TrackArtist != "David Bowie" {
		t.Errorf("unexpected candidate: %+v", c)
	}
}

func TestDiscoverClassicNoMatchIsUnresolvedNotFailure(t *testing.T) {
	f := &fakeCatalogue{
		artists: map[string][]spotify.Artist{
			"David Bowie": {{ID: "a-1", Name: "David Bowie Tribute Band"}},
		},
	}
	svc := newTestSvc(f, testConfig())

	result, err := svc.DiscoverClassic(context.Background())
	if err != nil {
		t.Fatalf("DiscoverClassic returned error: %v", err)
	}
	if !slices.Contains(result.UnresolvedArtists, "David Bowie") {
		t.Errorf("expected David Bowie unresolved, got %+v", result.UnresolvedArtists)
	}
	if len(result.Failures) != 0 {
		t.Errorf("a no-match should not be a Failure, got %+v", result.Failures)
	}
}

func TestDiscoverClassicUnresolvedArtistsAreSurfacedForAll(t *testing.T) {
	svc := newTestSvc(&fakeCatalogue{}, testConfig())

	result, err := svc.DiscoverClassic(context.Background())
	if err != nil {
		t.Fatalf("DiscoverClassic returned error: %v", err)
	}
	if len(result.UnresolvedArtists) != len(PastReferenceArtists) {
		t.Fatalf("expected all %d reference artists unresolved, got %d: %+v",
			len(PastReferenceArtists), len(result.UnresolvedArtists), result.UnresolvedArtists)
	}
	if len(result.Candidates) != 0 {
		t.Errorf("expected no candidates, got %+v", result.Candidates)
	}
}

func TestDiscoverClassicAlbumWalkBoundedByMaxAlbumsPerArtist(t *testing.T) {
	albums := make([]spotify.Album, 12)
	for i := range albums {
		albums[i] = spotify.Album{ID: idOf(i), Name: idOf(i)}
	}
	f := &fakeCatalogue{
		artists: map[string][]spotify.Artist{"David Bowie": {{ID: "a-bowie", Name: "David Bowie"}}},
		albums:  map[string][]spotify.Album{"a-bowie": albums},
		tracks:  map[string][]spotify.Track{},
	}
	cfg := testConfig()
	cfg.MaxAlbumsPerArtist = 3
	svc := newTestSvc(f, cfg)

	result, err := svc.DiscoverClassic(context.Background())
	if err != nil {
		t.Fatalf("DiscoverClassic returned error: %v", err)
	}
	if result.AlbumsInspected != 3 {
		t.Errorf("expected exactly 3 albums inspected (MaxAlbumsPerArtist), got %d", result.AlbumsInspected)
	}
}

func TestDiscoverClassicTrackWalkBoundedByMaxTracksPerAlbum(t *testing.T) {
	tracks := make([]spotify.Track, 25)
	for i := range tracks {
		tracks[i] = spotify.Track{ID: idOf(i), Name: idOf(i)}
	}
	f := &fakeCatalogue{
		artists: map[string][]spotify.Artist{"David Bowie": {{ID: "a-bowie", Name: "David Bowie"}}},
		albums:  map[string][]spotify.Album{"a-bowie": {{ID: "al-1", Name: "Low"}}},
		tracks:  map[string][]spotify.Track{"al-1": tracks},
	}
	cfg := testConfig()
	cfg.MaxTracksPerAlbum = 7
	svc := newTestSvc(f, cfg)

	result, err := svc.DiscoverClassic(context.Background())
	if err != nil {
		t.Fatalf("DiscoverClassic returned error: %v", err)
	}
	if result.TracksInspected != 7 {
		t.Errorf("expected exactly 7 tracks inspected (MaxTracksPerAlbum), got %d", result.TracksInspected)
	}
	if len(result.Candidates) != 7 {
		t.Errorf("expected exactly 7 candidates, got %d", len(result.Candidates))
	}
}

func TestDiscoverClassicDedupesWithinOneAlbum(t *testing.T) {
	f := &fakeCatalogue{
		artists: map[string][]spotify.Artist{"David Bowie": {{ID: "a-bowie", Name: "David Bowie"}}},
		albums:  map[string][]spotify.Album{"a-bowie": {{ID: "al-1", Name: "Reissue"}}},
		tracks: map[string][]spotify.Track{
			"al-1": {{ID: "t-1", Name: "Heroes"}, {ID: "t-1", Name: "Heroes"}},
		},
	}
	svc := newTestSvc(f, testConfig())

	result, err := svc.DiscoverClassic(context.Background())
	if err != nil {
		t.Fatalf("DiscoverClassic returned error: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("expected 1 deduplicated candidate, got %d", len(result.Candidates))
	}
	if result.DuplicatesSkipped != 1 {
		t.Errorf("expected 1 duplicate skipped, got %d", result.DuplicatesSkipped)
	}
}

func TestDiscoverClassicDedupesAcrossTwoAlbums(t *testing.T) {
	f := &fakeCatalogue{
		artists: map[string][]spotify.Artist{"David Bowie": {{ID: "a-bowie", Name: "David Bowie"}}},
		albums: map[string][]spotify.Album{
			"a-bowie": {{ID: "al-1", Name: "Original"}, {ID: "al-2", Name: "Remaster"}},
		},
		tracks: map[string][]spotify.Track{
			"al-1": {{ID: "t-1", Name: "Heroes"}},
			"al-2": {{ID: "t-1", Name: "Heroes (2017 Remaster)"}},
		},
	}
	svc := newTestSvc(f, testConfig())

	result, err := svc.DiscoverClassic(context.Background())
	if err != nil {
		t.Fatalf("DiscoverClassic returned error: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("expected 1 deduplicated candidate across albums, got %d: %+v", len(result.Candidates), result.Candidates)
	}
	if result.DuplicatesSkipped != 1 {
		t.Errorf("expected 1 duplicate skipped, got %d", result.DuplicatesSkipped)
	}
}

func TestDiscoverClassicMaxTotalCandidatesCutsOffMidRun(t *testing.T) {
	artists := map[string][]spotify.Artist{}
	albums := map[string][]spotify.Album{}
	trackMap := map[string][]spotify.Track{}
	for i, name := range PastReferenceArtists {
		aid := idOf(i)
		artists[name] = []spotify.Artist{{ID: aid, Name: name}}
		albums[aid] = []spotify.Album{{ID: aid + "-al", Name: "Album"}}
		// Track IDs are unique per artist so the global dedup map doesn't
		// collapse them — this test exercises the MaxTotalCandidates
		// cutoff, not dedup.
		tracks := make([]spotify.Track, 5)
		for j := range tracks {
			tracks[j] = spotify.Track{ID: aid + "-t" + idOf(j), Name: "Track"}
		}
		trackMap[aid+"-al"] = tracks
	}
	f := &fakeCatalogue{artists: artists, albums: albums, tracks: trackMap}
	cfg := testConfig()
	cfg.MaxTotalCandidates = 7 // less than 15 artists * 5 tracks
	svc := newTestSvc(f, cfg)

	result, err := svc.DiscoverClassic(context.Background())
	if err != nil {
		t.Fatalf("DiscoverClassic returned error: %v", err)
	}
	if len(result.Candidates) != 7 {
		t.Fatalf("expected exactly MaxTotalCandidates (7) candidates, got %d", len(result.Candidates))
	}
}

func TestDiscoverClassicAlbumFetchFailureRecordedAndContinues(t *testing.T) {
	f := &fakeCatalogue{
		artists: map[string][]spotify.Artist{
			"David Bowie": {{ID: "a-bowie", Name: "David Bowie"}},
			"Prince":      {{ID: "a-prince", Name: "Prince"}},
		},
		albumsErr: map[string]error{"a-bowie": errors.New("spotify: boom")},
		albums:    map[string][]spotify.Album{"a-prince": {{ID: "al-p", Name: "Purple Rain"}}},
		tracks:    map[string][]spotify.Track{"al-p": {{ID: "t-p", Name: "Purple Rain"}}},
	}
	svc := newTestSvc(f, testConfig())

	result, err := svc.DiscoverClassic(context.Background())
	if err != nil {
		t.Fatalf("DiscoverClassic returned error: %v", err)
	}
	found := false
	for _, fl := range result.Failures {
		if fl.Artist == "David Bowie" && fl.Stage == "albums" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an albums-stage failure for David Bowie, got %+v", result.Failures)
	}
	if len(result.Candidates) != 1 || result.Candidates[0].SpotifyTrackID != "t-p" {
		t.Errorf("expected discovery to continue to Prince, got %+v", result.Candidates)
	}
}

func TestDiscoverClassicTrackFetchFailureRecordedAndContinues(t *testing.T) {
	f := &fakeCatalogue{
		artists: map[string][]spotify.Artist{"David Bowie": {{ID: "a-bowie", Name: "David Bowie"}}},
		albums: map[string][]spotify.Album{
			"a-bowie": {{ID: "al-bad", Name: "Bad"}, {ID: "al-good", Name: "Good"}},
		},
		tracksErr: map[string]error{"al-bad": errors.New("spotify: boom")},
		tracks:    map[string][]spotify.Track{"al-good": {{ID: "t-1", Name: "Fine"}}},
	}
	svc := newTestSvc(f, testConfig())

	result, err := svc.DiscoverClassic(context.Background())
	if err != nil {
		t.Fatalf("DiscoverClassic returned error: %v", err)
	}
	found := false
	for _, fl := range result.Failures {
		if fl.Stage == "tracks" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a tracks-stage failure, got %+v", result.Failures)
	}
	if len(result.Candidates) != 1 || result.Candidates[0].SpotifyTrackID != "t-1" {
		t.Errorf("expected discovery to continue to the next album, got %+v", result.Candidates)
	}
}

func TestDiscoverClassicNotConnectedAbortsAndReturnsError(t *testing.T) {
	f := &fakeCatalogue{
		searchErr: map[string]error{"David Bowie": spotify.ErrNotConnected},
	}
	svc := newTestSvc(f, testConfig())

	_, err := svc.DiscoverClassic(context.Background())
	if !errors.Is(err, spotify.ErrNotConnected) {
		t.Fatalf("expected ErrNotConnected, got %v", err)
	}
	if f.searchCalls != 1 {
		t.Errorf("expected the run to abort after the first connection failure, got %d search calls", f.searchCalls)
	}
}

func TestDiscoverClassicInvalidGrantAbortsAndReturnsError(t *testing.T) {
	f := &fakeCatalogue{
		searchErr: map[string]error{"David Bowie": spotify.ErrInvalidGrant},
	}
	svc := newTestSvc(f, testConfig())

	_, err := svc.DiscoverClassic(context.Background())
	if !errors.Is(err, spotify.ErrInvalidGrant) {
		t.Fatalf("expected ErrInvalidGrant, got %v", err)
	}
}

func TestDiscoverClassicCandidateFieldsAreCorrect(t *testing.T) {
	f := &fakeCatalogue{
		artists: map[string][]spotify.Artist{"Kraftwerk": {{ID: "a-1", Name: "Kraftwerk"}}},
		albums:  map[string][]spotify.Album{"a-1": {{ID: "al-1", Name: "Autobahn"}}},
		tracks:  map[string][]spotify.Track{"al-1": {{ID: "t-1", Name: "Autobahn"}}},
	}
	svc := newTestSvc(f, testConfig())

	result, err := svc.DiscoverClassic(context.Background())
	if err != nil {
		t.Fatalf("DiscoverClassic returned error: %v", err)
	}
	c := result.Candidates[0]
	if c.Source != candidate.SourceSpotify {
		t.Errorf("expected Source=Spotify, got %v", c.Source)
	}
	if c.Type != candidate.TypeClassic {
		t.Errorf("expected Type=Classic, got %v", c.Type)
	}
	if c.Category != candidate.CategoryPast {
		t.Errorf("expected Category=Past, got %v", c.Category)
	}
	if c.Status != candidate.StatusDiscovered {
		t.Errorf("expected Status=discovered, got %v", c.Status)
	}
	if c.DiscoveryReason != discoveryReason {
		t.Errorf("expected DiscoveryReason %q, got %q", discoveryReason, c.DiscoveryReason)
	}
	if c.SpotifyTrackID != "t-1" || string(c.ID) != "t-1" {
		t.Errorf("expected candidate ID/SpotifyTrackID to be the Spotify track ID, got %+v", c)
	}
}

func TestDiscoverClassicNoPopularityOrAutoSelection(t *testing.T) {
	// Ensure Track never even carries a popularity-style field, so no
	// caller could accidentally rank on it. Track only exists in the
	// spotify package's own type — this is a structural assertion.
	f := &fakeCatalogue{
		artists: map[string][]spotify.Artist{"Kraftwerk": {{ID: "a-1", Name: "Kraftwerk"}}},
		albums:  map[string][]spotify.Album{"a-1": {{ID: "al-1", Name: "Autobahn"}}},
		tracks:  map[string][]spotify.Track{"al-1": {{ID: "t-1", Name: "Autobahn"}, {ID: "t-2", Name: "Kometenmelodie"}}},
	}
	svc := newTestSvc(f, testConfig())

	result, err := svc.DiscoverClassic(context.Background())
	if err != nil {
		t.Fatalf("DiscoverClassic returned error: %v", err)
	}
	for _, c := range result.Candidates {
		if c.Status != candidate.StatusDiscovered {
			t.Errorf("no candidate should be auto-selected, got Status=%v", c.Status)
		}
	}
	// Candidates preserve Spotify's own catalogue order (t-1, t-2) —
	// nothing reorders them by any score.
	if len(result.Candidates) != 2 || result.Candidates[0].SpotifyTrackID != "t-1" || result.Candidates[1].SpotifyTrackID != "t-2" {
		t.Errorf("expected candidates in catalogue order with no reordering, got %+v", result.Candidates)
	}
}

func TestClassicHandlerSuccess(t *testing.T) {
	f := &fakeCatalogue{
		artists: map[string][]spotify.Artist{"Kraftwerk": {{ID: "a-1", Name: "Kraftwerk"}}},
		albums:  map[string][]spotify.Album{"a-1": {{ID: "al-1", Name: "Autobahn"}}},
		tracks:  map[string][]spotify.Track{"al-1": {{ID: "t-1", Name: "Autobahn"}}},
	}
	svc := newTestSvc(f, testConfig())

	rec := httptest.NewRecorder()
	svc.ClassicHandler(rec, httptest.NewRequest(http.MethodPost, "/api/discovery/classic", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "t-1") {
		t.Errorf("expected the response to include the discovered track, got %s", rec.Body.String())
	}
}

func TestClassicHandlerNotConnectedMapsTo503(t *testing.T) {
	f := &fakeCatalogue{searchErr: map[string]error{"David Bowie": spotify.ErrNotConnected}}
	svc := newTestSvc(f, testConfig())

	rec := httptest.NewRecorder()
	svc.ClassicHandler(rec, httptest.NewRequest(http.MethodPost, "/api/discovery/classic", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestClassicHandlerInvalidGrantMapsTo401(t *testing.T) {
	f := &fakeCatalogue{searchErr: map[string]error{"David Bowie": spotify.ErrInvalidGrant}}
	svc := newTestSvc(f, testConfig())

	rec := httptest.NewRecorder()
	svc.ClassicHandler(rec, httptest.NewRequest(http.MethodPost, "/api/discovery/classic", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func idOf(i int) string {
	return fmt.Sprintf("id-%d", i)
}

func TestDiscoverCurrentResolvesAndBuildsCandidate(t *testing.T) {
	f := &fakeCatalogue{
		artists: map[string][]spotify.Artist{"James Blake": {{ID: "a-1", Name: "James Blake"}}},
		albums:  map[string][]spotify.Album{"a-1": {{ID: "al-1", Name: "Wind Down", ReleaseDate: daysAgo(10), ReleaseDatePrecision: "day"}}},
		tracks:  map[string][]spotify.Track{"al-1": {{ID: "t-1", Name: "Wind Down"}}},
	}
	svc := newTestSvcCurrent(f, testCurrentConfig())

	result, err := svc.DiscoverCurrent(context.Background())
	if err != nil {
		t.Fatalf("DiscoverCurrent returned error: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d: %+v", len(result.Candidates), result.Candidates)
	}
	c := result.Candidates[0]
	if c.SpotifyTrackID != "t-1" || c.TrackTitle != "Wind Down" || c.TrackArtist != "James Blake" {
		t.Errorf("unexpected candidate: %+v", c)
	}
}

func TestDiscoverCurrentCandidateFieldsAreCorrect(t *testing.T) {
	f := &fakeCatalogue{
		artists: map[string][]spotify.Artist{"Sampha": {{ID: "a-1", Name: "Sampha"}}},
		albums:  map[string][]spotify.Album{"a-1": {{ID: "al-1", Name: "Spirit 2.0", ReleaseDate: daysAgo(5), ReleaseDatePrecision: "day"}}},
		tracks:  map[string][]spotify.Track{"al-1": {{ID: "t-1", Name: "Spirit 2.0"}}},
	}
	svc := newTestSvcCurrent(f, testCurrentConfig())

	result, err := svc.DiscoverCurrent(context.Background())
	if err != nil {
		t.Fatalf("DiscoverCurrent returned error: %v", err)
	}
	c := result.Candidates[0]
	if c.Source != candidate.SourceSpotify {
		t.Errorf("expected Source=Spotify, got %v", c.Source)
	}
	if c.Type != candidate.TypeCurrent {
		t.Errorf("expected Type=Current, got %v", c.Type)
	}
	if c.Category != candidate.CategoryPresent {
		t.Errorf("expected Category=Present, got %v", c.Category)
	}
	if c.Status != candidate.StatusDiscovered {
		t.Errorf("expected Status=discovered, got %v", c.Status)
	}
	if c.DiscoveryReason != currentDiscoveryReason {
		t.Errorf("expected DiscoveryReason %q, got %q", currentDiscoveryReason, c.DiscoveryReason)
	}
	if c.SpotifyTrackID != "t-1" || string(c.ID) != "t-1" {
		t.Errorf("expected candidate ID/SpotifyTrackID to be the Spotify track ID, got %+v", c)
	}
}

func TestDiscoverCurrentExcludesReleasesOutsideWindow(t *testing.T) {
	f := &fakeCatalogue{
		artists: map[string][]spotify.Artist{"Sampha": {{ID: "a-1", Name: "Sampha"}}},
		albums: map[string][]spotify.Album{"a-1": {
			{ID: "al-recent", Name: "Recent", ReleaseDate: daysAgo(10), ReleaseDatePrecision: "day"},
			{ID: "al-old", Name: "Old", ReleaseDate: "2010-01-01", ReleaseDatePrecision: "day"},
		}},
		tracks: map[string][]spotify.Track{
			"al-recent": {{ID: "t-recent", Name: "Recent Track"}},
			"al-old":    {{ID: "t-old", Name: "Old Track"}},
		},
	}
	svc := newTestSvcCurrent(f, testCurrentConfig())

	result, err := svc.DiscoverCurrent(context.Background())
	if err != nil {
		t.Fatalf("DiscoverCurrent returned error: %v", err)
	}
	if len(result.Candidates) != 1 || result.Candidates[0].SpotifyTrackID != "t-recent" {
		t.Fatalf("expected only the recent release's track, got %+v", result.Candidates)
	}
	if result.ReleasesOutsideWindow != 1 {
		t.Errorf("expected 1 release outside window, got %d", result.ReleasesOutsideWindow)
	}
}

func TestDiscoverCurrentIncludesSingles(t *testing.T) {
	// AlbumType isn't inspected at all — ArtistAlbums already fixes
	// include_groups=album,single, so a single-shaped release needs no
	// special-casing here to be considered.
	f := &fakeCatalogue{
		artists: map[string][]spotify.Artist{"Yaeji": {{ID: "a-1", Name: "Yaeji"}}},
		albums:  map[string][]spotify.Album{"a-1": {{ID: "al-1", Name: "New Single", AlbumType: "single", ReleaseDate: daysAgo(3), ReleaseDatePrecision: "day"}}},
		tracks:  map[string][]spotify.Track{"al-1": {{ID: "t-1", Name: "New Single"}}},
	}
	svc := newTestSvcCurrent(f, testCurrentConfig())

	result, err := svc.DiscoverCurrent(context.Background())
	if err != nil {
		t.Fatalf("DiscoverCurrent returned error: %v", err)
	}
	if len(result.Candidates) != 1 || result.Candidates[0].SpotifyTrackID != "t-1" {
		t.Fatalf("expected the single's track to be discovered, got %+v", result.Candidates)
	}
}

func TestDiscoverCurrentDedupesAcrossReleases(t *testing.T) {
	f := &fakeCatalogue{
		artists: map[string][]spotify.Artist{"Arca": {{ID: "a-1", Name: "Arca"}}},
		albums: map[string][]spotify.Album{"a-1": {
			{ID: "al-1", Name: "Single Version", ReleaseDate: daysAgo(20), ReleaseDatePrecision: "day"},
			{ID: "al-2", Name: "Album Version", ReleaseDate: daysAgo(5), ReleaseDatePrecision: "day"},
		}},
		tracks: map[string][]spotify.Track{
			"al-1": {{ID: "t-1", Name: "KLK"}},
			"al-2": {{ID: "t-1", Name: "KLK (Album Version)"}},
		},
	}
	svc := newTestSvcCurrent(f, testCurrentConfig())

	result, err := svc.DiscoverCurrent(context.Background())
	if err != nil {
		t.Fatalf("DiscoverCurrent returned error: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("expected 1 deduplicated candidate, got %d: %+v", len(result.Candidates), result.Candidates)
	}
	if result.DuplicatesSkipped != 1 {
		t.Errorf("expected 1 duplicate skipped, got %d", result.DuplicatesSkipped)
	}
}

func TestDiscoverCurrentUnresolvedArtistsAreSurfaced(t *testing.T) {
	svc := newTestSvcCurrent(&fakeCatalogue{}, testCurrentConfig())

	result, err := svc.DiscoverCurrent(context.Background())
	if err != nil {
		t.Fatalf("DiscoverCurrent returned error: %v", err)
	}
	if len(result.UnresolvedArtists) != len(PresentReferenceArtists) {
		t.Fatalf("expected all %d reference artists unresolved, got %d: %+v",
			len(PresentReferenceArtists), len(result.UnresolvedArtists), result.UnresolvedArtists)
	}
	if len(result.Failures) != 0 {
		t.Errorf("a no-match should not be a Failure, got %+v", result.Failures)
	}
}

func TestDiscoverCurrentAlbumAndTrackFetchFailuresRecordedAndContinue(t *testing.T) {
	f := &fakeCatalogue{
		artists: map[string][]spotify.Artist{
			"Fred again.": {{ID: "a-fred", Name: "Fred again."}},
			"James Blake": {{ID: "a-blake", Name: "James Blake"}},
		},
		albumsErr: map[string]error{"a-fred": errors.New("spotify: boom")},
		albums:    map[string][]spotify.Album{"a-blake": {{ID: "al-1", Name: "Ok", ReleaseDate: daysAgo(5), ReleaseDatePrecision: "day"}}},
		tracksErr: map[string]error{},
		tracks:    map[string][]spotify.Track{"al-1": {{ID: "t-1", Name: "Ok"}}},
	}
	svc := newTestSvcCurrent(f, testCurrentConfig())

	result, err := svc.DiscoverCurrent(context.Background())
	if err != nil {
		t.Fatalf("DiscoverCurrent returned error: %v", err)
	}
	found := false
	for _, fl := range result.Failures {
		if fl.Artist == "Fred again." && fl.Stage == "albums" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an albums-stage failure for Fred again., got %+v", result.Failures)
	}
	if len(result.Candidates) != 1 || result.Candidates[0].SpotifyTrackID != "t-1" {
		t.Errorf("expected discovery to continue to James Blake, got %+v", result.Candidates)
	}
}

func TestDiscoverCurrentTrackFetchFailureRecordedAndContinues(t *testing.T) {
	f := &fakeCatalogue{
		artists: map[string][]spotify.Artist{"Kelela": {{ID: "a-1", Name: "Kelela"}}},
		albums: map[string][]spotify.Album{"a-1": {
			{ID: "al-bad", Name: "Bad", ReleaseDate: daysAgo(6), ReleaseDatePrecision: "day"},
			{ID: "al-good", Name: "Good", ReleaseDate: daysAgo(4), ReleaseDatePrecision: "day"},
		}},
		tracksErr: map[string]error{"al-bad": errors.New("spotify: boom")},
		tracks:    map[string][]spotify.Track{"al-good": {{ID: "t-1", Name: "Fine"}}},
	}
	svc := newTestSvcCurrent(f, testCurrentConfig())

	result, err := svc.DiscoverCurrent(context.Background())
	if err != nil {
		t.Fatalf("DiscoverCurrent returned error: %v", err)
	}
	found := false
	for _, fl := range result.Failures {
		if fl.Stage == "tracks" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a tracks-stage failure, got %+v", result.Failures)
	}
	if len(result.Candidates) != 1 || result.Candidates[0].SpotifyTrackID != "t-1" {
		t.Errorf("expected discovery to continue to the next release, got %+v", result.Candidates)
	}
}

func TestDiscoverCurrentConnectionErrorsAbortRun(t *testing.T) {
	f := &fakeCatalogue{
		searchErr: map[string]error{"Fred again.": spotify.ErrNotConnected},
	}
	svc := newTestSvcCurrent(f, testCurrentConfig())

	_, err := svc.DiscoverCurrent(context.Background())
	if !errors.Is(err, spotify.ErrNotConnected) {
		t.Fatalf("expected ErrNotConnected, got %v", err)
	}
	if f.searchCalls != 1 {
		t.Errorf("expected the run to abort after the first connection failure, got %d search calls", f.searchCalls)
	}

	f2 := &fakeCatalogue{
		searchErr: map[string]error{"Fred again.": spotify.ErrInvalidGrant},
	}
	svc2 := newTestSvcCurrent(f2, testCurrentConfig())
	_, err = svc2.DiscoverCurrent(context.Background())
	if !errors.Is(err, spotify.ErrInvalidGrant) {
		t.Fatalf("expected ErrInvalidGrant, got %v", err)
	}
}

func TestDiscoverCurrentBoundedByScanAndSelectionCaps(t *testing.T) {
	albums := make([]spotify.Album, 20)
	for i := range albums {
		// Deliberately NOT sorted newest-first in Spotify's returned
		// order, to prove DiscoverCurrent sorts explicitly rather than
		// trusting the order it receives.
		albums[i] = spotify.Album{ID: idOf(i), Name: idOf(i), ReleaseDate: daysAgo(i), ReleaseDatePrecision: "day"}
	}
	tracks := map[string][]spotify.Track{}
	for _, a := range albums {
		tracks[a.ID] = []spotify.Track{{ID: a.ID + "-t", Name: a.ID}}
	}
	f := &fakeCatalogue{
		artists: map[string][]spotify.Artist{"Peggy Gou": {{ID: "a-1", Name: "Peggy Gou"}}},
		albums:  map[string][]spotify.Album{"a-1": albums},
		tracks:  tracks,
	}
	cfg := testCurrentConfig()
	cfg.MaxAlbumsScannedPerArtist = 20
	cfg.MaxAlbumsPerArtist = 3
	svc := newTestSvcCurrent(f, cfg)

	result, err := svc.DiscoverCurrent(context.Background())
	if err != nil {
		t.Fatalf("DiscoverCurrent returned error: %v", err)
	}
	if result.AlbumsInspected != 3 {
		t.Fatalf("expected exactly 3 albums inspected (MaxAlbumsPerArtist), got %d", result.AlbumsInspected)
	}
	// The 3 kept releases must be the 3 most recent: id-0, id-1, id-2
	// (daysAgo(0) is newest).
	want := map[string]bool{"id-0-t": true, "id-1-t": true, "id-2-t": true}
	for _, c := range result.Candidates {
		if !want[c.SpotifyTrackID] {
			t.Errorf("expected only the 3 most recent releases' tracks, got %+v", c)
		}
	}
}

func TestDiscoverCurrentMaxTotalCandidatesCutsOffMidRun(t *testing.T) {
	artists := map[string][]spotify.Artist{}
	albums := map[string][]spotify.Album{}
	trackMap := map[string][]spotify.Track{}
	for i, name := range PresentReferenceArtists {
		aid := idOf(i)
		artists[name] = []spotify.Artist{{ID: aid, Name: name}}
		albums[aid] = []spotify.Album{{ID: aid + "-al", Name: "Release", ReleaseDate: daysAgo(1), ReleaseDatePrecision: "day"}}
		tracks := make([]spotify.Track, 5)
		for j := range tracks {
			tracks[j] = spotify.Track{ID: aid + "-t" + idOf(j), Name: "Track"}
		}
		trackMap[aid+"-al"] = tracks
	}
	f := &fakeCatalogue{artists: artists, albums: albums, tracks: trackMap}
	cfg := testCurrentConfig()
	cfg.MaxTotalCandidates = 7 // less than 15 artists * 5 tracks
	svc := newTestSvcCurrent(f, cfg)

	result, err := svc.DiscoverCurrent(context.Background())
	if err != nil {
		t.Fatalf("DiscoverCurrent returned error: %v", err)
	}
	if len(result.Candidates) != 7 {
		t.Fatalf("expected exactly MaxTotalCandidates (7) candidates, got %d", len(result.Candidates))
	}
}

func TestDiscoverCurrentNoPopularityReordering(t *testing.T) {
	f := &fakeCatalogue{
		artists: map[string][]spotify.Artist{"Little Simz": {{ID: "a-1", Name: "Little Simz"}}},
		albums:  map[string][]spotify.Album{"a-1": {{ID: "al-1", Name: "NO THANK YOU", ReleaseDate: daysAgo(2), ReleaseDatePrecision: "day"}}},
		tracks:  map[string][]spotify.Track{"al-1": {{ID: "t-1", Name: "Angel"}, {ID: "t-2", Name: "Mood Swings"}}},
	}
	svc := newTestSvcCurrent(f, testCurrentConfig())

	result, err := svc.DiscoverCurrent(context.Background())
	if err != nil {
		t.Fatalf("DiscoverCurrent returned error: %v", err)
	}
	for _, c := range result.Candidates {
		if c.Status != candidate.StatusDiscovered {
			t.Errorf("no candidate should be auto-selected, got Status=%v", c.Status)
		}
	}
	if len(result.Candidates) != 2 || result.Candidates[0].SpotifyTrackID != "t-1" || result.Candidates[1].SpotifyTrackID != "t-2" {
		t.Errorf("expected candidates in catalogue order with no reordering, got %+v", result.Candidates)
	}
}

func TestParseReleaseDatePrecision(t *testing.T) {
	cases := []struct {
		name      string
		raw       string
		precision string
		wantOK    bool
		want      string // RFC3339 date portion, only checked if wantOK
	}{
		{"day precision", "2024-03-15", "day", true, "2024-03-15"},
		{"month precision resolves to the 1st", "2024-03", "month", true, "2024-03-01"},
		{"year precision resolves to Jan 1st", "2024", "year", true, "2024-01-01"},
		{"unparseable date", "not-a-date", "day", false, ""},
		{"empty date", "", "day", false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseReleaseDate(tc.raw, tc.precision)
			if ok != tc.wantOK {
				t.Fatalf("parseReleaseDate(%q, %q) ok = %v, want %v", tc.raw, tc.precision, ok, tc.wantOK)
			}
			if tc.wantOK && got.Format("2006-01-02") != tc.want {
				t.Errorf("parseReleaseDate(%q, %q) = %v, want date %s", tc.raw, tc.precision, got, tc.want)
			}
		})
	}
}

func TestCurrentHandlerSuccess(t *testing.T) {
	f := &fakeCatalogue{
		artists: map[string][]spotify.Artist{"Rosalía": {{ID: "a-1", Name: "Rosalía"}}},
		albums:  map[string][]spotify.Album{"a-1": {{ID: "al-1", Name: "Motomami", ReleaseDate: daysAgo(2), ReleaseDatePrecision: "day"}}},
		tracks:  map[string][]spotify.Track{"al-1": {{ID: "t-1", Name: "Motomami"}}},
	}
	svc := newTestSvcCurrent(f, testCurrentConfig())

	rec := httptest.NewRecorder()
	svc.CurrentHandler(rec, httptest.NewRequest(http.MethodPost, "/api/discovery/current", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "t-1") {
		t.Errorf("expected the response to include the discovered track, got %s", rec.Body.String())
	}
}

func TestCurrentHandlerNotConnectedMapsTo503(t *testing.T) {
	f := &fakeCatalogue{searchErr: map[string]error{"Fred again.": spotify.ErrNotConnected}}
	svc := newTestSvcCurrent(f, testCurrentConfig())

	rec := httptest.NewRecorder()
	svc.CurrentHandler(rec, httptest.NewRequest(http.MethodPost, "/api/discovery/current", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestCurrentHandlerInvalidGrantMapsTo401(t *testing.T) {
	f := &fakeCatalogue{searchErr: map[string]error{"Fred again.": spotify.ErrInvalidGrant}}
	svc := newTestSvcCurrent(f, testCurrentConfig())

	rec := httptest.NewRecorder()
	svc.CurrentHandler(rec, httptest.NewRequest(http.MethodPost, "/api/discovery/current", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
