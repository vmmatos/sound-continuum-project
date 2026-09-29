package discovery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
	"github.com/vmmatos/sound-continuum-project/internal/lastfm"
	"github.com/vmmatos/sound-continuum-project/internal/spotify"
)

func newTestSvcPool(f *fakeCatalogue, lf *fakeSimilarArtistFinder, cc Config, cu CurrentConfig, em EmergingConfig) *Service {
	return &Service{spotify: f, lastfm: lf, classicCfg: cc, currentCfg: cu, emergingCfg: em, now: time.Now}
}

// fullPoolFixture wires a fakeCatalogue/fakeSimilarArtistFinder so all three
// workflows resolve exactly one artist and produce exactly one candidate
// each: Classic via PastReferenceArtists[0] ("David Bowie"), Current via
// PresentReferenceArtists[0] ("Fred again."), Emerging via
// EmergingReferenceArtists[0] ("The Twins") similar-artist hop to a
// non-canonical "New Artist X". Every other reference artist/seed is left
// unresolved/empty, which is valid discovery behavior, not a test bug.
func fullPoolFixture() (*fakeCatalogue, *fakeSimilarArtistFinder) {
	f := &fakeCatalogue{
		// An empty official playlist (no playlistItems entry for this ID)
		// so PoolHandler's FilterRecentTracks step succeeds trivially and
		// every discovered candidate stays eligible.
		officialPlaylist: &spotify.OfficialPlaylist{SpotifyPlaylistID: "official-playlist-id"},
		artists: map[string][]spotify.Artist{
			"David Bowie":  {{ID: "bowie-id", Name: "David Bowie"}},
			"Fred again.":  {{ID: "fred-id", Name: "Fred again."}},
			"New Artist X": {{ID: "newx-id", Name: "New Artist X"}},
		},
		albums: map[string][]spotify.Album{
			"bowie-id": {{ID: "al-classic-1", Name: "Album C"}},
			"fred-id": {{ID: "al-current-1", Name: "Album Cur",
				ReleaseDate: daysAgo(10), ReleaseDatePrecision: "day"}},
			"newx-id": {{ID: "al-emerging-1", Name: "Album Em",
				ReleaseDate: daysAgo(5), ReleaseDatePrecision: "day"}},
		},
		tracks: map[string][]spotify.Track{
			"al-classic-1":  {{ID: "track-classic-1", Name: "Song C"}},
			"al-current-1":  {{ID: "track-current-1", Name: "Song Cur"}},
			"al-emerging-1": {{ID: "track-emerging-1", Name: "Song E"}},
		},
	}
	lf := &fakeSimilarArtistFinder{
		similar: map[string][]lastfm.SimilarArtist{
			"The Twins": {{Name: "New Artist X", Match: 0.9}},
		},
	}
	return f, lf
}

func fullPoolSvc() *Service {
	f, lf := fullPoolFixture()
	return newTestSvcPool(f, lf, testConfig(), testCurrentConfig(), testEmergingConfig())
}

func TestDiscoverPoolIncludesClassicCurrentEmerging(t *testing.T) {
	pool := fullPoolSvc().DiscoverPool(context.Background())

	byID := make(map[string]candidate.CandidateTrack, len(pool.Candidates))
	for _, c := range pool.Candidates {
		byID[c.SpotifyTrackID] = c
	}

	classic, ok := byID["track-classic-1"]
	if !ok {
		t.Fatalf("expected classic candidate in pool, got %+v", pool.Candidates)
	}
	if classic.Type != candidate.TypeClassic || classic.Category != candidate.CategoryPast {
		t.Errorf("classic candidate has wrong classification: %+v", classic)
	}

	current, ok := byID["track-current-1"]
	if !ok {
		t.Fatalf("expected current candidate in pool, got %+v", pool.Candidates)
	}
	if current.Type != candidate.TypeCurrent || current.Category != candidate.CategoryPresent {
		t.Errorf("current candidate has wrong classification: %+v", current)
	}

	emerging, ok := byID["track-emerging-1"]
	if !ok {
		t.Fatalf("expected emerging candidate in pool, got %+v", pool.Candidates)
	}
	if emerging.Type != candidate.TypeDiscovery || emerging.Category != candidate.CategoryEmerging {
		t.Errorf("emerging candidate has wrong classification: %+v", emerging)
	}

	for _, c := range pool.Candidates {
		if c.Status != candidate.StatusDiscovered {
			t.Errorf("candidate %s has status %q, want discovered", c.SpotifyTrackID, c.Status)
		}
		if c.Source != candidate.SourceSpotify {
			t.Errorf("candidate %s has source %q, want Spotify", c.SpotifyTrackID, c.Source)
		}
	}
}

func TestDiscoverPoolMergesAndCounts(t *testing.T) {
	pool := fullPoolSvc().DiscoverPool(context.Background())

	if pool.TotalCandidates != 3 {
		t.Errorf("TotalCandidates = %d, want 3", pool.TotalCandidates)
	}
	if pool.ClassicCandidates != 1 {
		t.Errorf("ClassicCandidates = %d, want 1", pool.ClassicCandidates)
	}
	if pool.CurrentCandidates != 1 {
		t.Errorf("CurrentCandidates = %d, want 1", pool.CurrentCandidates)
	}
	if pool.EmergingCandidates != 1 {
		t.Errorf("EmergingCandidates = %d, want 1", pool.EmergingCandidates)
	}
	if pool.DuplicatesRemoved != 0 {
		t.Errorf("DuplicatesRemoved = %d, want 0", pool.DuplicatesRemoved)
	}
}

func TestDiscoverPoolDeduplicatesBySpotifyTrackID(t *testing.T) {
	f := &fakeCatalogue{
		artists: map[string][]spotify.Artist{
			"David Bowie": {{ID: "bowie-id", Name: "David Bowie"}},
			"Fred again.": {{ID: "fred-id", Name: "Fred again."}},
		},
		albums: map[string][]spotify.Album{
			"bowie-id": {{ID: "al-classic-1", Name: "Album C"}},
			"fred-id": {{ID: "al-current-1", Name: "Album Cur",
				ReleaseDate: daysAgo(10), ReleaseDatePrecision: "day"}},
		},
		tracks: map[string][]spotify.Track{
			// Same Spotify track ID discovered by both Classic and Current.
			"al-classic-1": {{ID: "track-shared-1", Name: "Shared Song"}},
			"al-current-1": {{ID: "track-shared-1", Name: "Shared Song"}},
		},
	}
	lf := &fakeSimilarArtistFinder{}
	svc := newTestSvcPool(f, lf, testConfig(), testCurrentConfig(), testEmergingConfig())

	pool := svc.DiscoverPool(context.Background())

	if pool.TotalCandidates != 1 {
		t.Fatalf("TotalCandidates = %d, want 1: %+v", pool.TotalCandidates, pool.Candidates)
	}
	if pool.DuplicatesRemoved != 1 {
		t.Errorf("DuplicatesRemoved = %d, want 1", pool.DuplicatesRemoved)
	}
	if pool.Candidates[0].Type != candidate.TypeClassic {
		t.Errorf("surviving candidate Type = %q, want Classic (Classic beats Current on priority)", pool.Candidates[0].Type)
	}
	// Deduplication removes the duplicate candidate, not its provenance:
	// the surviving candidate must carry both workflows' discovery paths.
	prov := pool.Candidates[0].Provenance
	if len(prov) != 2 {
		t.Fatalf("expected merged provenance from both workflows, got %+v", prov)
	}
	methods := map[candidate.DiscoveryMethod]bool{}
	for _, p := range prov {
		methods[p.Method] = true
	}
	if !methods[candidate.DiscoveryMethodClassicReferenceArtist] || !methods[candidate.DiscoveryMethodCurrentReferenceArtist] {
		t.Errorf("expected both classic_reference_artist and current_reference_artist provenance, got %+v", prov)
	}
}

// TestDiscoverPoolEmergingCandidateHasDistinctSourceAndProvenanceProvider
// confirms Card #39's core distinction survives the Pool: an Emerging
// candidate's Source (the track's provider) stays Spotify even though its
// discovery provenance's Provider is Last.fm (the discovery signal).
func TestDiscoverPoolEmergingCandidateHasDistinctSourceAndProvenanceProvider(t *testing.T) {
	pool := fullPoolSvc().DiscoverPool(context.Background())

	var emerging *candidate.CandidateTrack
	for i := range pool.Candidates {
		if pool.Candidates[i].SpotifyTrackID == "track-emerging-1" {
			emerging = &pool.Candidates[i]
		}
	}
	if emerging == nil {
		t.Fatalf("expected emerging candidate in pool, got %+v", pool.Candidates)
	}
	if emerging.Source != candidate.SourceSpotify {
		t.Errorf("Source = %q, want Spotify", emerging.Source)
	}
	if len(emerging.Provenance) != 1 || emerging.Provenance[0].Provider != candidate.ProvenanceProviderLastFM {
		t.Errorf("expected provenance Provider=Last.fm alongside Source=Spotify, got %+v", emerging.Provenance)
	}
}

func TestDiscoverPoolClassicPartialFailureKeepsCurrentAndEmerging(t *testing.T) {
	f, lf := fullPoolFixture()
	f.searchErr = map[string]error{"David Bowie": spotify.ErrNotConnected}
	svc := newTestSvcPool(f, lf, testConfig(), testCurrentConfig(), testEmergingConfig())

	pool := svc.DiscoverPool(context.Background())

	if len(pool.WorkflowErrors) != 1 || pool.WorkflowErrors[0].Workflow != "classic" {
		t.Fatalf("WorkflowErrors = %+v, want exactly one classic entry", pool.WorkflowErrors)
	}
	if pool.WorkflowErrors[0].Err != spotify.ErrNotConnected.Error() {
		t.Errorf("WorkflowErrors[0].Err = %q, want the real underlying error message %q",
			pool.WorkflowErrors[0].Err, spotify.ErrNotConnected.Error())
	}
	if pool.ClassicResult.ArtistsInspected != 0 || len(pool.ClassicResult.Candidates) != 0 {
		t.Errorf("ClassicResult = %+v, want zero value on abort", pool.ClassicResult)
	}
	if len(pool.CurrentResult.Candidates) != 1 || len(pool.EmergingResult.Candidates) != 1 {
		t.Errorf("expected Current and Emerging to still succeed, got Current=%+v Emerging=%+v",
			pool.CurrentResult, pool.EmergingResult)
	}
	if pool.TotalCandidates != 2 {
		t.Errorf("TotalCandidates = %d, want 2 (current + emerging only)", pool.TotalCandidates)
	}
}

func TestDiscoverPoolCurrentPartialFailureKeepsClassicAndEmerging(t *testing.T) {
	f, lf := fullPoolFixture()
	f.searchErr = map[string]error{"Fred again.": spotify.ErrNotConnected}
	svc := newTestSvcPool(f, lf, testConfig(), testCurrentConfig(), testEmergingConfig())

	pool := svc.DiscoverPool(context.Background())

	if len(pool.WorkflowErrors) != 1 || pool.WorkflowErrors[0].Workflow != "current" {
		t.Fatalf("WorkflowErrors = %+v, want exactly one current entry", pool.WorkflowErrors)
	}
	if pool.CurrentResult.ArtistsInspected != 0 || len(pool.CurrentResult.Candidates) != 0 {
		t.Errorf("CurrentResult = %+v, want zero value on abort", pool.CurrentResult)
	}
	if len(pool.ClassicResult.Candidates) != 1 || len(pool.EmergingResult.Candidates) != 1 {
		t.Errorf("expected Classic and Emerging to still succeed, got Classic=%+v Emerging=%+v",
			pool.ClassicResult, pool.EmergingResult)
	}
	if pool.TotalCandidates != 2 {
		t.Errorf("TotalCandidates = %d, want 2 (classic + emerging only)", pool.TotalCandidates)
	}
}

func TestDiscoverPoolEmergingSeedFailureContinues(t *testing.T) {
	f, lf := fullPoolFixture()
	// EmergingReferenceArtists[0] is "The Twins" (fullPoolFixture's seed);
	// fail it with a non-config error and add a second, independent seed
	// that still succeeds.
	lf.err = map[string]error{"The Twins": lastfm.ErrRateLimited}
	lf.similar["Toxe"] = []lastfm.SimilarArtist{{Name: "Another New Artist", Match: 0.5}}
	f.artists["Another New Artist"] = []spotify.Artist{{ID: "another-id", Name: "Another New Artist"}}
	f.albums["another-id"] = []spotify.Album{{ID: "al-another-1", Name: "Album Another",
		ReleaseDate: daysAgo(3), ReleaseDatePrecision: "day"}}
	f.tracks["al-another-1"] = []spotify.Track{{ID: "track-another-1", Name: "Song Another"}}

	svc := newTestSvcPool(f, lf, testConfig(), testCurrentConfig(), testEmergingConfig())
	pool := svc.DiscoverPool(context.Background())

	if len(pool.WorkflowErrors) != 0 {
		t.Fatalf("WorkflowErrors = %+v, want none (a per-seed failure is not an abort)", pool.WorkflowErrors)
	}
	if len(pool.EmergingResult.Failures) != 1 || pool.EmergingResult.Failures[0].Stage != "similar" {
		t.Errorf("EmergingResult.Failures = %+v, want one similar-stage failure", pool.EmergingResult.Failures)
	}
	if len(pool.EmergingResult.Candidates) != 1 {
		t.Errorf("EmergingResult.Candidates = %+v, want 1 (from the surviving seed)", pool.EmergingResult.Candidates)
	}
}

func TestDiscoverPoolMissingLastfmConfigSurfaced(t *testing.T) {
	f, lf := fullPoolFixture()
	// "The Twins" is EmergingReferenceArtists[0] — a config error on the
	// very first seed aborts DiscoverEmerging before any other seed runs.
	lf.err = map[string]error{"The Twins": lastfm.ErrMissingAPIKey}
	svc := newTestSvcPool(f, lf, testConfig(), testCurrentConfig(), testEmergingConfig())

	pool := svc.DiscoverPool(context.Background())

	if len(pool.WorkflowErrors) != 1 || pool.WorkflowErrors[0].Workflow != "emerging" {
		t.Fatalf("WorkflowErrors = %+v, want exactly one emerging entry", pool.WorkflowErrors)
	}
	if pool.EmergingResult.ArtistsInspected != 0 || len(pool.EmergingResult.Candidates) != 0 {
		t.Errorf("EmergingResult = %+v, want zero value on abort", pool.EmergingResult)
	}
	if len(pool.ClassicResult.Candidates) != 1 || len(pool.CurrentResult.Candidates) != 1 {
		t.Errorf("expected Classic and Current to still succeed, got Classic=%+v Current=%+v",
			pool.ClassicResult, pool.CurrentResult)
	}
}

func TestDiscoverPoolOrderingIsDeterministic(t *testing.T) {
	run := func() []candidate.CandidateTrack {
		return fullPoolSvc().DiscoverPool(context.Background()).Candidates
	}
	first := run()
	second := run()

	if len(first) != len(second) {
		t.Fatalf("got different candidate counts across runs: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].SpotifyTrackID != second[i].SpotifyTrackID {
			t.Fatalf("ordering not stable across runs at index %d: %q vs %q",
				i, first[i].SpotifyTrackID, second[i].SpotifyTrackID)
		}
	}
	// Classic (priority 0) must sort before Current (1) before Emerging (2).
	for i := 1; i < len(first); i++ {
		if candidateTypePriority[first[i-1].Type] > candidateTypePriority[first[i].Type] {
			t.Errorf("candidates not ordered by Type priority: %+v before %+v", first[i-1], first[i])
		}
	}
}

func TestDiscoverPoolNoEditorialSelection(t *testing.T) {
	pool := fullPoolSvc().DiscoverPool(context.Background())

	rawTotal := len(pool.ClassicResult.Candidates) + len(pool.CurrentResult.Candidates) + len(pool.EmergingResult.Candidates)
	if pool.TotalCandidates != rawTotal-pool.DuplicatesRemoved {
		t.Errorf("pool dropped candidates beyond dedup: raw=%d duplicatesRemoved=%d total=%d",
			rawTotal, pool.DuplicatesRemoved, pool.TotalCandidates)
	}
	for _, c := range pool.Candidates {
		if c.Status != candidate.StatusDiscovered {
			t.Errorf("candidate %s has status %q, editorial review has not happened yet", c.SpotifyTrackID, c.Status)
		}
	}
}

func TestDiscoverPoolEmptyResultsHandled(t *testing.T) {
	f := &fakeCatalogue{}
	lf := &fakeSimilarArtistFinder{}
	svc := newTestSvcPool(f, lf, testConfig(), testCurrentConfig(), testEmergingConfig())

	pool := svc.DiscoverPool(context.Background())

	if pool.TotalCandidates != 0 || len(pool.Candidates) != 0 {
		t.Errorf("expected an empty pool, got %+v", pool)
	}
	if len(pool.WorkflowErrors) != 0 {
		t.Errorf("WorkflowErrors = %+v, want none (no artists resolving is not a configuration failure)", pool.WorkflowErrors)
	}
	if len(pool.ClassicResult.UnresolvedArtists) != len(PastReferenceArtists) {
		t.Errorf("ClassicResult.UnresolvedArtists has %d entries, want %d",
			len(pool.ClassicResult.UnresolvedArtists), len(PastReferenceArtists))
	}
}

func TestPoolHandlerReturns200WithJSON(t *testing.T) {
	svc := fullPoolSvc()

	rec := httptest.NewRecorder()
	svc.PoolHandler(rec, httptest.NewRequest(http.MethodPost, "/api/candidates/pool", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var pool CandidatePool
	if err := json.Unmarshal(rec.Body.Bytes(), &pool); err != nil {
		t.Fatalf("response body did not decode as CandidatePool: %v", err)
	}
	if pool.TotalCandidates != 3 {
		t.Errorf("TotalCandidates = %d, want 3", pool.TotalCandidates)
	}
	if pool.RecentTrackFilter.EligibleCount != 3 || pool.RecentTrackFilter.RecentlyUsedCount != 0 {
		t.Errorf("RecentTrackFilter = %+v, want all 3 candidates eligible (empty official playlist)",
			pool.RecentTrackFilter)
	}
}

func TestPoolHandlerEnrichesEligibleNotRecentlyUsed(t *testing.T) {
	f, lf := fullPoolFixture()
	// track-classic-1 was added to the official playlist just now, so it's
	// recently used and must not be enriched. track-current-1/
	// track-emerging-1 stay eligible and get real metadata.
	f.playlistItems = map[string][]spotify.PlaylistItem{
		"official-playlist-id": {
			{ItemType: "track", AddedAt: time.Now().Format(time.RFC3339), Track: &spotify.Track{ID: "track-classic-1"}},
		},
	}
	f.trackByID = map[string]spotify.Track{
		"track-current-1":  {ID: "track-current-1", Name: "Song Cur"},
		"track-emerging-1": {ID: "track-emerging-1", Name: "Song E"},
	}
	svc := newTestSvcPool(f, lf, testConfig(), testCurrentConfig(), testEmergingConfig())
	svc.recentTrackLookbackDays = DefaultRecentTrackLookbackDays

	rec := httptest.NewRecorder()
	svc.PoolHandler(rec, httptest.NewRequest(http.MethodPost, "/api/candidates/pool", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var pool CandidatePool
	if err := json.Unmarshal(rec.Body.Bytes(), &pool); err != nil {
		t.Fatalf("response body did not decode as CandidatePool: %v", err)
	}

	if len(pool.RecentTrackFilter.RecentlyUsedCandidates) != 1 ||
		pool.RecentTrackFilter.RecentlyUsedCandidates[0].Candidate.SpotifyTrackID != "track-classic-1" {
		t.Fatalf("RecentlyUsedCandidates = %+v, want exactly track-classic-1", pool.RecentTrackFilter.RecentlyUsedCandidates)
	}

	if len(pool.RecentTrackFilter.EligibleCandidates) != 2 {
		t.Fatalf("EligibleCandidates = %+v, want 2", pool.RecentTrackFilter.EligibleCandidates)
	}
	for _, c := range pool.RecentTrackFilter.EligibleCandidates {
		if c.Metadata == nil || c.Metadata.Title == "" {
			t.Errorf("eligible candidate %s was not enriched: %+v", c.SpotifyTrackID, c)
		}
	}

	for _, id := range f.trackCalls {
		if id == "track-classic-1" {
			t.Error("recently-used candidate track-classic-1 must not be enriched (no Track call expected)")
		}
	}
	if pool.MetadataEnrichment.EnrichedCount != 2 {
		t.Errorf("MetadataEnrichment.EnrichedCount = %d, want 2", pool.MetadataEnrichment.EnrichedCount)
	}
}

func TestPoolHandlerReturnsErrorWhenOfficialPlaylistNotConfigured(t *testing.T) {
	f, lf := fullPoolFixture()
	f.officialPlaylist = nil // not initialized yet
	svc := newTestSvcPool(f, lf, testConfig(), testCurrentConfig(), testEmergingConfig())

	rec := httptest.NewRecorder()
	svc.PoolHandler(rec, httptest.NewRequest(http.MethodPost, "/api/candidates/pool", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503: %s", rec.Code, rec.Body.String())
	}
}
