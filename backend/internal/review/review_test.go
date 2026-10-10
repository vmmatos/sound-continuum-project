package review

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
	"github.com/vmmatos/sound-continuum-project/internal/discovery"
	"github.com/vmmatos/sound-continuum-project/internal/scoring"
	"github.com/vmmatos/sound-continuum-project/internal/spotify"
)

var fixedNow = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)

var errUnexpected = errors.New("unexpected failure")

func daysBeforeNow(days int) time.Time {
	return fixedNow.AddDate(0, 0, -days)
}

// fakeDiscovery implements candidatePoolSource entirely in memory, the same
// seam-testing approach discovery's own fakeCatalogue already establishes
// (see decisions.md, Card #33).
type fakeDiscovery struct {
	pool discovery.CandidatePool

	filterResult discovery.RecentTrackFilterResult
	filterErr    error

	enrichResult discovery.EnrichmentResult
	enrichErr    error

	trackHistory    map[string]time.Time
	trackHistoryErr error

	artistHistory    map[string]time.Time
	artistHistoryErr error

	trackHistoryCalls  int
	artistHistoryCalls int
}

func (f *fakeDiscovery) DiscoverPool(ctx context.Context) discovery.CandidatePool {
	return f.pool
}

func (f *fakeDiscovery) FilterRecentTracks(ctx context.Context, candidates []candidate.CandidateTrack) (discovery.RecentTrackFilterResult, error) {
	return f.filterResult, f.filterErr
}

func (f *fakeDiscovery) EnrichCandidateMetadata(ctx context.Context, candidates []candidate.CandidateTrack) (discovery.EnrichmentResult, error) {
	return f.enrichResult, f.enrichErr
}

func (f *fakeDiscovery) PlaylistTrackHistory(ctx context.Context) (map[string]time.Time, error) {
	f.trackHistoryCalls++
	return f.trackHistory, f.trackHistoryErr
}

func (f *fakeDiscovery) PlaylistArtistHistory(ctx context.Context) (map[string]time.Time, error) {
	f.artistHistoryCalls++
	return f.artistHistory, f.artistHistoryErr
}

// fakeSelection implements selectionLookup entirely in memory, the same
// seam-testing approach fakeDiscovery already establishes.
type fakeSelection struct {
	selected    map[string]struct{}
	underReview map[string]struct{}
	rejected    map[string]struct{}
	err         error
}

func (f *fakeSelection) AllSelected(ctx context.Context) (map[string]struct{}, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.selected == nil {
		return map[string]struct{}{}, nil
	}
	return f.selected, nil
}

func (f *fakeSelection) AllUnderReview(ctx context.Context) (map[string]struct{}, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.underReview == nil {
		return map[string]struct{}{}, nil
	}
	return f.underReview, nil
}

func (f *fakeSelection) AllRejected(ctx context.Context) (map[string]struct{}, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.rejected == nil {
		return map[string]struct{}{}, nil
	}
	return f.rejected, nil
}

func newTestService(f *fakeDiscovery) *Service {
	return newTestServiceWithSelection(f, &fakeSelection{})
}

func newTestServiceWithSelection(f *fakeDiscovery, sel *fakeSelection) *Service {
	return &Service{discovery: f, selection: sel, now: func() time.Time { return fixedNow }}
}

func testCandidate(t *testing.T, id, spotifyTrackID string, artistIDs ...string) candidate.CandidateTrack {
	t.Helper()
	c, err := candidate.NewCandidateTrack(candidate.NewCandidateTrackParams{
		ID:             candidate.ID(id),
		SpotifyTrackID: spotifyTrackID,
		Source:         candidate.SourceSpotify,
		Category:       candidate.CategoryEmerging,
		Type:           candidate.TypeDiscovery,
		TrackTitle:     "Track " + id,
		TrackArtist:    "Artist " + id,
	})
	if err != nil {
		t.Fatalf("NewCandidateTrack(%s): %v", id, err)
	}
	if len(artistIDs) > 0 {
		artists := make([]candidate.CandidateArtist, 0, len(artistIDs))
		for _, aid := range artistIDs {
			artists = append(artists, candidate.CandidateArtist{SpotifyArtistID: aid, Name: aid})
		}
		c.Metadata = &candidate.CandidateMetadata{
			Title:   c.TrackTitle,
			Artists: artists,
			Album: candidate.CandidateAlbum{
				SpotifyAlbumID: "album-" + id,
				Name:           "Album " + id,
				Artwork:        []candidate.CandidateImage{{URL: "https://i.scdn.co/" + id, Width: 300, Height: 300}},
			},
		}
	}
	return c
}

func fakeWithEligible(candidates []candidate.CandidateTrack) *fakeDiscovery {
	return &fakeDiscovery{
		filterResult: discovery.RecentTrackFilterResult{EligibleCandidates: candidates},
		enrichResult: discovery.EnrichmentResult{EnrichedCandidates: candidates},
		trackHistory: map[string]time.Time{},
		artistHistory: map[string]time.Time{},
	}
}

// 1. Candidate with no previous playlist history.
func TestReviewPoolNoHistoryCandidate(t *testing.T) {
	c := testCandidate(t, "c1", "track-1", "artist-1")
	f := fakeWithEligible([]candidate.CandidateTrack{c})
	svc := newTestService(f)

	pool, err := svc.ReviewPool(context.Background())
	if err != nil {
		t.Fatalf("ReviewPool: %v", err)
	}
	if len(pool.Entries) != 1 {
		t.Fatalf("len(Entries) = %d, want 1", len(pool.Entries))
	}

	factors := pool.Entries[0].Ranked.Score.Factors
	if factors.Freshness == nil || *factors.Freshness != 1.0 {
		t.Errorf("Freshness = %v, want 1.0", factors.Freshness)
	}
	if factors.RepetitionPenalty == nil || *factors.RepetitionPenalty != 0 {
		t.Errorf("RepetitionPenalty = %v, want 0", factors.RepetitionPenalty)
	}
	for name, f := range map[string]*float64{
		"Fit": factors.Fit, "DiscoveryBonus": factors.DiscoveryBonus,
		"Diversity": factors.Diversity, "PlaylistFit": factors.PlaylistFit,
	} {
		if f != nil {
			t.Errorf("%s = %v, want nil", name, *f)
		}
	}
	if pool.Entries[0].Ranked.Score.FinalScore == nil {
		t.Error("FinalScore is nil, want a computed value (Freshness contributes weight)")
	}
}

// 2. Recently played track: Freshness/RepetitionPenalty reflect the exact
// values the existing scoring functions produce for the same inputs.
func TestReviewPoolRecentlyPlayedTrack(t *testing.T) {
	c := testCandidate(t, "c1", "track-1", "artist-1")
	lastUsed := daysBeforeNow(10)
	f := fakeWithEligible([]candidate.CandidateTrack{c})
	f.trackHistory = map[string]time.Time{"track-1": lastUsed}
	svc := newTestService(f)

	pool, err := svc.ReviewPool(context.Background())
	if err != nil {
		t.Fatalf("ReviewPool: %v", err)
	}

	wantFreshness, err := scoring.CalculateFreshness(&lastUsed, fixedNow, scoring.DefaultFreshnessConfig())
	if err != nil {
		t.Fatalf("CalculateFreshness: %v", err)
	}
	wantRepetition, err := scoring.CalculateRepetitionPenalty(&lastUsed, nil, fixedNow, scoring.DefaultRepetitionPenaltyConfig())
	if err != nil {
		t.Fatalf("CalculateRepetitionPenalty: %v", err)
	}

	factors := pool.Entries[0].Ranked.Score.Factors
	if factors.Freshness == nil || *factors.Freshness != wantFreshness.Value {
		t.Errorf("Freshness = %v, want %v", factors.Freshness, wantFreshness.Value)
	}
	if factors.RepetitionPenalty == nil || *factors.RepetitionPenalty != wantRepetition.Value {
		t.Errorf("RepetitionPenalty = %v, want %v", factors.RepetitionPenalty, wantRepetition.Value)
	}
	if *factors.Freshness == 1.0 {
		t.Error("Freshness == 1.0 for a used track, want < 1.0")
	}
}

// 3. Previously-used artist, different track: RepetitionPenalty uses the
// existing artist-history behavior (track severity 0, artist severity > 0).
func TestReviewPoolArtistHistoryDifferentTrack(t *testing.T) {
	c := testCandidate(t, "c1", "track-1", "artist-1")
	artistLastUsed := daysBeforeNow(5)
	f := fakeWithEligible([]candidate.CandidateTrack{c})
	f.artistHistory = map[string]time.Time{"artist-1": artistLastUsed}
	svc := newTestService(f)

	pool, err := svc.ReviewPool(context.Background())
	if err != nil {
		t.Fatalf("ReviewPool: %v", err)
	}

	wantRepetition, err := scoring.CalculateRepetitionPenalty(nil, &artistLastUsed, fixedNow, scoring.DefaultRepetitionPenaltyConfig())
	if err != nil {
		t.Fatalf("CalculateRepetitionPenalty: %v", err)
	}

	factors := pool.Entries[0].Ranked.Score.Factors
	if factors.RepetitionPenalty == nil || *factors.RepetitionPenalty != wantRepetition.Value {
		t.Errorf("RepetitionPenalty = %v, want %v", factors.RepetitionPenalty, wantRepetition.Value)
	}
	if wantRepetition.Value == 0 {
		t.Fatal("test fixture bug: expected artist repetition to be > 0")
	}
	if factors.Freshness == nil || *factors.Freshness != 1.0 {
		t.Errorf("Freshness = %v, want 1.0 (track itself never used)", factors.Freshness)
	}
}

// 4. Empty eligible pool: valid empty response, no error, and no playlist
// history is fetched at all (never needed, never per-candidate).
func TestReviewPoolEmptyEligiblePool(t *testing.T) {
	f := &fakeDiscovery{
		filterResult:     discovery.RecentTrackFilterResult{EligibleCandidates: nil},
		enrichResult:     discovery.EnrichmentResult{EnrichedCandidates: nil},
		trackHistoryErr:  errors.New("PlaylistTrackHistory must not be called for an empty pool"),
		artistHistoryErr: errors.New("PlaylistArtistHistory must not be called for an empty pool"),
	}
	svc := newTestService(f)

	pool, err := svc.ReviewPool(context.Background())
	if err != nil {
		t.Fatalf("ReviewPool: %v", err)
	}
	if len(pool.Entries) != 0 {
		t.Errorf("len(Entries) = %d, want 0", len(pool.Entries))
	}
	if f.trackHistoryCalls != 0 || f.artistHistoryCalls != 0 {
		t.Errorf("playlist history fetched for an empty pool: track=%d artist=%d calls", f.trackHistoryCalls, f.artistHistoryCalls)
	}
}

// 5. Deterministic ranking: same inputs produce same ordering across calls.
func TestReviewPoolDeterministicRanking(t *testing.T) {
	// Two never-used candidates tie on FinalScore; Rank's tie-breaker
	// (Candidate.ID ascending) must decide the order, stably, every call.
	candidates := []candidate.CandidateTrack{
		testCandidate(t, "c2", "track-2", "artist-2"),
		testCandidate(t, "c1", "track-1", "artist-1"),
	}

	var lastOrder []candidate.ID
	for i := 0; i < 5; i++ {
		f := fakeWithEligible(candidates)
		svc := newTestService(f)
		pool, err := svc.ReviewPool(context.Background())
		if err != nil {
			t.Fatalf("ReviewPool: %v", err)
		}
		order := make([]candidate.ID, len(pool.Entries))
		for j, e := range pool.Entries {
			order[j] = e.Ranked.Candidate.ID
			if e.Ranked.Rank != j+1 {
				t.Errorf("run %d: entry %d Rank = %d, want %d", i, j, e.Ranked.Rank, j+1)
			}
		}
		if lastOrder != nil {
			if len(order) != len(lastOrder) || order[0] != lastOrder[0] || order[1] != lastOrder[1] {
				t.Fatalf("run %d: order = %v, want %v (non-deterministic)", i, order, lastOrder)
			}
		}
		lastOrder = order
	}
	if lastOrder[0] != candidate.ID("c1") {
		t.Errorf("order[0] = %q, want %q (ID-ascending tie-break)", lastOrder[0], "c1")
	}
}

// 6. Partial scoring: AvailableWeight is exactly Freshness's weight (the
// only positive factor this card populates), and no unavailable factor
// silently becomes zero.
func TestReviewPoolAvailableWeightIsFreshnessWeightOnly(t *testing.T) {
	c := testCandidate(t, "c1", "track-1", "artist-1")
	f := fakeWithEligible([]candidate.CandidateTrack{c})
	svc := newTestService(f)

	pool, err := svc.ReviewPool(context.Background())
	if err != nil {
		t.Fatalf("ReviewPool: %v", err)
	}

	want := scoring.DefaultWeights().Freshness
	got := pool.Entries[0].Ranked.Score.AvailableWeight
	if got != want {
		t.Errorf("AvailableWeight = %v, want %v (Freshness weight only)", got, want)
	}

	factors := pool.Entries[0].Ranked.Score.Factors
	for name, fv := range map[string]*float64{
		"Fit": factors.Fit, "DiscoveryBonus": factors.DiscoveryBonus,
		"Diversity": factors.Diversity, "PlaylistFit": factors.PlaylistFit,
	} {
		if fv != nil {
			t.Errorf("%s = %v, want nil (never a fabricated zero)", name, *fv)
		}
	}
}

// 7. Explanation is generated exclusively by scoring.GenerateExplanation.
func TestReviewPoolExplanationComesOnlyFromGenerateExplanation(t *testing.T) {
	c := testCandidate(t, "c1", "track-1", "artist-1")
	f := fakeWithEligible([]candidate.CandidateTrack{c})
	svc := newTestService(f)

	pool, err := svc.ReviewPool(context.Background())
	if err != nil {
		t.Fatalf("ReviewPool: %v", err)
	}

	entry := pool.Entries[0]
	want := scoring.GenerateExplanation(scoring.ExplanationInput{Score: entry.Ranked.Score})
	if entry.Explanation.Text != want.Text {
		t.Errorf("Explanation.Text = %q, want %q", entry.Explanation.Text, want.Text)
	}
	if len(entry.Explanation.Reasons) != len(want.Reasons) {
		t.Errorf("Explanation.Reasons = %v, want %v", entry.Explanation.Reasons, want.Reasons)
	}
}

// 8. Bridge remains nil.
func TestReviewPoolBridgeAlwaysNil(t *testing.T) {
	c := testCandidate(t, "c1", "track-1", "artist-1")
	f := fakeWithEligible([]candidate.CandidateTrack{c})
	svc := newTestService(f)

	pool, err := svc.ReviewPool(context.Background())
	if err != nil {
		t.Fatalf("ReviewPool: %v", err)
	}
	entry := pool.Entries[0]
	if entry.Bridge != nil {
		t.Errorf("Bridge = %v, want nil", entry.Bridge)
	}
	if entry.BridgeTrack != nil {
		t.Errorf("BridgeTrack = %v, want nil", entry.BridgeTrack)
	}
}

// 9. Clean empty pool: no WorkflowErrors, no Failures, from either
// Discovery or the per-workflow Results.
func TestReviewPoolCleanEmptyPoolHasNoFailures(t *testing.T) {
	f := &fakeDiscovery{}
	svc := newTestService(f)

	pool, err := svc.ReviewPool(context.Background())
	if err != nil {
		t.Fatalf("ReviewPool: %v", err)
	}
	if len(pool.Entries) != 0 {
		t.Errorf("len(Entries) = %d, want 0", len(pool.Entries))
	}
	if len(pool.WorkflowErrors) != 0 {
		t.Errorf("WorkflowErrors = %v, want empty", pool.WorkflowErrors)
	}
	if len(pool.Failures) != 0 {
		t.Errorf("Failures = %v, want empty", pool.Failures)
	}
}

// 10. Empty pool with a WorkflowError only (e.g. a Spotify connection
// failure aborted one workflow) — surfaced on ReviewPool.WorkflowErrors.
func TestReviewPoolEmptyPoolWithWorkflowErrorOnly(t *testing.T) {
	f := &fakeDiscovery{
		pool: discovery.CandidatePool{
			WorkflowErrors: []discovery.WorkflowError{{Workflow: "classic", Err: "spotify: not connected"}},
		},
	}
	svc := newTestService(f)

	pool, err := svc.ReviewPool(context.Background())
	if err != nil {
		t.Fatalf("ReviewPool: %v", err)
	}
	if len(pool.Entries) != 0 {
		t.Errorf("len(Entries) = %d, want 0", len(pool.Entries))
	}
	if len(pool.WorkflowErrors) != 1 || pool.WorkflowErrors[0].Workflow != "classic" {
		t.Errorf("WorkflowErrors = %v, want one classic entry", pool.WorkflowErrors)
	}
	if len(pool.Failures) != 0 {
		t.Errorf("Failures = %v, want empty", pool.Failures)
	}
}

// 11. Empty pool with per-item Failures only (e.g. a Spotify 429 recorded
// on a workflow's own Result, which never aborts the workflow) —
// surfaced on ReviewPool.Failures, distinct from a clean empty pool.
func TestReviewPoolEmptyPoolWithFailuresOnly(t *testing.T) {
	f := &fakeDiscovery{
		pool: discovery.CandidatePool{
			ClassicResult: discovery.Result{
				Failures: []discovery.Failure{{Artist: "Some Artist", Stage: "albums", Err: "429 Too Many Requests"}},
			},
		},
	}
	svc := newTestService(f)

	pool, err := svc.ReviewPool(context.Background())
	if err != nil {
		t.Fatalf("ReviewPool: %v", err)
	}
	if len(pool.Entries) != 0 {
		t.Errorf("len(Entries) = %d, want 0", len(pool.Entries))
	}
	if len(pool.WorkflowErrors) != 0 {
		t.Errorf("WorkflowErrors = %v, want empty", pool.WorkflowErrors)
	}
	if len(pool.Failures) != 1 || pool.Failures[0].Stage != "albums" {
		t.Errorf("Failures = %v, want one albums-stage entry", pool.Failures)
	}
}

// 12. Failures present alongside valid candidates: the candidates still
// rank/display normally, and the failure is still surfaced — failures are
// informational, never a gate on Entries.
func TestReviewPoolFailuresDoNotHideValidCandidates(t *testing.T) {
	c := testCandidate(t, "c1", "track-1", "artist-1")
	f := fakeWithEligible([]candidate.CandidateTrack{c})
	f.pool = discovery.CandidatePool{
		CurrentResult: discovery.Result{
			Failures: []discovery.Failure{{Artist: "Other Artist", Stage: "albums", Err: "429 Too Many Requests"}},
		},
	}
	svc := newTestService(f)

	pool, err := svc.ReviewPool(context.Background())
	if err != nil {
		t.Fatalf("ReviewPool: %v", err)
	}
	if len(pool.Entries) != 1 {
		t.Fatalf("len(Entries) = %d, want 1", len(pool.Entries))
	}
	if pool.Entries[0].Ranked.Candidate.ID != c.ID {
		t.Errorf("Entries[0].Ranked.Candidate.ID = %q, want %q", pool.Entries[0].Ranked.Candidate.ID, c.ID)
	}
	if len(pool.Failures) != 1 || pool.Failures[0].Stage != "albums" {
		t.Errorf("Failures = %v, want one albums-stage entry", pool.Failures)
	}
}

// A recent-track-filter failure propagates instead of producing an empty
// or partial review.
func TestReviewPoolPropagatesFilterError(t *testing.T) {
	f := &fakeDiscovery{filterErr: spotify.ErrOfficialPlaylistNotConfigured}
	svc := newTestService(f)

	_, err := svc.ReviewPool(context.Background())
	if !errors.Is(err, spotify.ErrOfficialPlaylistNotConfigured) {
		t.Errorf("err = %v, want ErrOfficialPlaylistNotConfigured", err)
	}
}

// 13. A selected candidate (Card #56) comes back with Status overlaid to
// StatusSelected, while an unrelated candidate stays StatusDiscovered.
func TestReviewPoolOverlaysSelectedStatus(t *testing.T) {
	c1 := testCandidate(t, "c1", "track-1", "artist-1")
	c2 := testCandidate(t, "c2", "track-2", "artist-2")
	f := fakeWithEligible([]candidate.CandidateTrack{c1, c2})
	svc := newTestServiceWithSelection(f, &fakeSelection{selected: map[string]struct{}{"c1": {}}})

	pool, err := svc.ReviewPool(context.Background())
	if err != nil {
		t.Fatalf("ReviewPool: %v", err)
	}
	if len(pool.Entries) != 2 {
		t.Fatalf("len(Entries) = %d, want 2", len(pool.Entries))
	}

	for _, e := range pool.Entries {
		switch e.Ranked.Candidate.ID {
		case "c1":
			if e.Ranked.Candidate.Status != candidate.StatusSelected {
				t.Errorf("c1 Status = %q, want %q", e.Ranked.Candidate.Status, candidate.StatusSelected)
			}
		case "c2":
			if e.Ranked.Candidate.Status != candidate.StatusDiscovered {
				t.Errorf("c2 Status = %q, want %q (unrelated candidate must not be modified)", e.Ranked.Candidate.Status, candidate.StatusDiscovered)
			}
		}
	}
}

// 13b. A candidate marked Maybe (Card #57) comes back with Status overlaid
// to StatusUnderReview, while an unrelated candidate stays StatusDiscovered,
// and a selected candidate's overlay is unaffected by an unrelated Maybe.
func TestReviewPoolOverlaysUnderReviewStatus(t *testing.T) {
	c1 := testCandidate(t, "c1", "track-1", "artist-1")
	c2 := testCandidate(t, "c2", "track-2", "artist-2")
	c3 := testCandidate(t, "c3", "track-3", "artist-3")
	f := fakeWithEligible([]candidate.CandidateTrack{c1, c2, c3})
	svc := newTestServiceWithSelection(f, &fakeSelection{
		selected:    map[string]struct{}{"c3": {}},
		underReview: map[string]struct{}{"c1": {}},
	})

	pool, err := svc.ReviewPool(context.Background())
	if err != nil {
		t.Fatalf("ReviewPool: %v", err)
	}
	if len(pool.Entries) != 3 {
		t.Fatalf("len(Entries) = %d, want 3", len(pool.Entries))
	}

	for _, e := range pool.Entries {
		switch e.Ranked.Candidate.ID {
		case "c1":
			if e.Ranked.Candidate.Status != candidate.StatusUnderReview {
				t.Errorf("c1 Status = %q, want %q", e.Ranked.Candidate.Status, candidate.StatusUnderReview)
			}
		case "c2":
			if e.Ranked.Candidate.Status != candidate.StatusDiscovered {
				t.Errorf("c2 Status = %q, want %q (unrelated candidate must not be modified)", e.Ranked.Candidate.Status, candidate.StatusDiscovered)
			}
		case "c3":
			if e.Ranked.Candidate.Status != candidate.StatusSelected {
				t.Errorf("c3 Status = %q, want %q (unaffected by unrelated Maybe)", e.Ranked.Candidate.Status, candidate.StatusSelected)
			}
		}
	}
}

// 13c. A candidate marked Skip (Card #58) comes back with Status overlaid
// to StatusRejected, while an unrelated candidate stays StatusDiscovered,
// and unrelated Keep/Maybe overlays are unaffected by an unrelated Skip.
func TestReviewPoolOverlaysRejectedStatus(t *testing.T) {
	c1 := testCandidate(t, "c1", "track-1", "artist-1")
	c2 := testCandidate(t, "c2", "track-2", "artist-2")
	c3 := testCandidate(t, "c3", "track-3", "artist-3")
	c4 := testCandidate(t, "c4", "track-4", "artist-4")
	f := fakeWithEligible([]candidate.CandidateTrack{c1, c2, c3, c4})
	svc := newTestServiceWithSelection(f, &fakeSelection{
		selected:    map[string]struct{}{"c3": {}},
		underReview: map[string]struct{}{"c4": {}},
		rejected:    map[string]struct{}{"c1": {}},
	})

	pool, err := svc.ReviewPool(context.Background())
	if err != nil {
		t.Fatalf("ReviewPool: %v", err)
	}
	if len(pool.Entries) != 4 {
		t.Fatalf("len(Entries) = %d, want 4", len(pool.Entries))
	}

	for _, e := range pool.Entries {
		switch e.Ranked.Candidate.ID {
		case "c1":
			if e.Ranked.Candidate.Status != candidate.StatusRejected {
				t.Errorf("c1 Status = %q, want %q", e.Ranked.Candidate.Status, candidate.StatusRejected)
			}
		case "c2":
			if e.Ranked.Candidate.Status != candidate.StatusDiscovered {
				t.Errorf("c2 Status = %q, want %q (unrelated candidate must not be modified)", e.Ranked.Candidate.Status, candidate.StatusDiscovered)
			}
		case "c3":
			if e.Ranked.Candidate.Status != candidate.StatusSelected {
				t.Errorf("c3 Status = %q, want %q (unaffected by unrelated Skip)", e.Ranked.Candidate.Status, candidate.StatusSelected)
			}
		case "c4":
			if e.Ranked.Candidate.Status != candidate.StatusUnderReview {
				t.Errorf("c4 Status = %q, want %q (unaffected by unrelated Skip)", e.Ranked.Candidate.Status, candidate.StatusUnderReview)
			}
		}
	}
}

// 14. A selection-store error propagates instead of silently returning an
// un-overlaid (or empty) review.
func TestReviewPoolPropagatesSelectionError(t *testing.T) {
	c := testCandidate(t, "c1", "track-1", "artist-1")
	f := fakeWithEligible([]candidate.CandidateTrack{c})
	svc := newTestServiceWithSelection(f, &fakeSelection{err: errUnexpected})

	_, err := svc.ReviewPool(context.Background())
	if !errors.Is(err, errUnexpected) {
		t.Errorf("err = %v, want errUnexpected", err)
	}
}
