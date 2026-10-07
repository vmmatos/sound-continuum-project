// Package review composes the existing discovery and scoring packages into
// the real Candidate Review pipeline: Candidate Pool -> Recent Track Filter
// -> Enrichment -> Freshness + Repetition Penalty scoring -> FinalScore ->
// deterministic ranking -> explanation -> Candidate Review. It introduces no
// new discovery or scoring algorithm — every factor value, the final score,
// the ranking, and the explanation text all come directly from the existing
// discovery/scoring implementations.
//
// Only Freshness and RepetitionPenalty are calculated here, since those are
// the only two scoring factors with genuine production inputs today (the
// official Spotify playlist's track/artist history). Fit, DiscoveryBonus,
// Diversity, and PlaylistFit are left nil — their inputs (musicaldna.Profile
// tags, CurrentEditionContext, editorial discovery values) don't exist in
// any production workflow yet, and this package never fabricates them.
package review

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
	"github.com/vmmatos/sound-continuum-project/internal/discovery"
	"github.com/vmmatos/sound-continuum-project/internal/scoring"
	"github.com/vmmatos/sound-continuum-project/internal/selection"
	"github.com/vmmatos/sound-continuum-project/internal/spotify"
)

// candidatePoolSource is the slice of *discovery.Service this package
// depends on, extracted into an unexported interface for the same reason
// discovery.spotifyCatalogue exists (see docs/memory/decisions.md, Card
// #33): discovery.Service's fields are unexported and its only constructor
// takes a concrete *spotify.Service, so this package's own tests need
// something fakeable. Production callers (NewService) are unaffected — they
// still pass a concrete *discovery.Service.
type candidatePoolSource interface {
	DiscoverPool(ctx context.Context) discovery.CandidatePool
	FilterRecentTracks(ctx context.Context, candidates []candidate.CandidateTrack) (discovery.RecentTrackFilterResult, error)
	EnrichCandidateMetadata(ctx context.Context, candidates []candidate.CandidateTrack) (discovery.EnrichmentResult, error)
	PlaylistTrackHistory(ctx context.Context) (map[string]time.Time, error)
	PlaylistArtistHistory(ctx context.Context) (map[string]time.Time, error)
}

// selectionLookup is the slice of *selection.Store this package depends on,
// extracted into an unexported interface for the same reason
// candidatePoolSource exists (see docs/memory/decisions.md, Card #33):
// production callers (NewService) still pass a concrete *selection.Store,
// this package's own tests get something fakeable.
type selectionLookup interface {
	AllSelected(ctx context.Context) (map[string]struct{}, error)
	AllUnderReview(ctx context.Context) (map[string]struct{}, error)
}

// Service orchestrates discovery + scoring into a ReviewPool. It holds no
// scoring state of its own — scoring.Calculate/Rank/GenerateExplanation are
// pure functions, called directly.
type Service struct {
	discovery candidatePoolSource
	selection selectionLookup

	// now is a clock seam, overridden directly by same-package tests for
	// deterministic Freshness/RepetitionPenalty boundary testing — the same
	// one-off pattern discovery.Service.now already establishes (Card #37)
	// rather than a repo-wide clock abstraction. Defaults to time.Now.
	now func() time.Time
}

// NewService wires a review Service to an existing discovery.Service and
// selection.Store — no second discovery pipeline, and no new candidate
// pool persistence (the pool itself is still rebuilt per request; only
// Keep's selection state, Card #56, is persisted).
func NewService(discoverySvc *discovery.Service, selectionStore *selection.Store) *Service {
	return &Service{discovery: discoverySvc, selection: selectionStore, now: time.Now}
}

// ReviewEntry is one candidate's place in the Candidate Review response:
// its rank and full score, the explanation generated from that score, and
// a potential musical bridge for the same candidate pair. Bridge/BridgeTrack
// are always nil for this card — Card #48's DetectPotentialBridge needs a
// candidate pair, which no caller of ReviewPool constructs yet.
type ReviewEntry struct {
	Ranked      scoring.RankedCandidate
	Explanation scoring.CandidateExplanation
	Bridge      *scoring.BridgeResult
	BridgeTrack *string
}

// ReviewPool is the real Candidate Review response: every eligible
// candidate, scored on the factors with genuine production inputs today,
// ranked deterministically.
type ReviewPool struct {
	Entries []ReviewEntry

	// WorkflowErrors and Failures are discovery.CandidatePool's own
	// WorkflowErrors and each workflow Result's Failures (ClassicResult/
	// CurrentResult/EmergingResult), carried through unchanged — reused
	// directly, no new failure type (Failures is the three workflows'
	// slices merged into one). They let a caller tell apart an empty
	// Entries because Discovery ran clean and nothing was eligible (both
	// empty) from an empty (or partial) Entries because Discovery hit
	// failures along the way (e.g. a Spotify 429). They are purely
	// informational: valid candidates in Entries still rank/display
	// normally regardless of unrelated failures elsewhere in the pool.
	WorkflowErrors []discovery.WorkflowError
	Failures       []discovery.Failure
}

// ReviewPool runs the full production pipeline: DiscoverPool ->
// FilterRecentTracks -> EnrichCandidateMetadata -> Freshness +
// RepetitionPenalty -> scoring.Calculate -> scoring.Rank ->
// scoring.GenerateExplanation. A recent-track-filter or metadata-enrichment
// or playlist-history failure propagates as an error, matching
// discovery.Service.PoolHandler's own precedent — a temporary Spotify
// failure must never silently become an empty or partial review. An empty
// eligible pool is not an error: it returns a ReviewPool with no entries,
// carrying whatever WorkflowErrors/Failures DiscoverPool already recorded.
func (s *Service) ReviewPool(ctx context.Context) (ReviewPool, error) {
	pool := s.discovery.DiscoverPool(ctx)

	var failures []discovery.Failure
	failures = append(failures, pool.ClassicResult.Failures...)
	failures = append(failures, pool.CurrentResult.Failures...)
	failures = append(failures, pool.EmergingResult.Failures...)

	filtered, err := s.discovery.FilterRecentTracks(ctx, pool.Candidates)
	if err != nil {
		return ReviewPool{}, err
	}

	enrichment, err := s.discovery.EnrichCandidateMetadata(ctx, filtered.EligibleCandidates)
	if err != nil {
		return ReviewPool{}, err
	}
	eligible := enrichment.EnrichedCandidates

	if len(eligible) == 0 {
		return ReviewPool{Entries: []ReviewEntry{}, WorkflowErrors: pool.WorkflowErrors, Failures: failures}, nil
	}

	// Playlist history is fetched once for the whole operation, never once
	// per candidate.
	trackHistory, err := s.discovery.PlaylistTrackHistory(ctx)
	if err != nil {
		return ReviewPool{}, err
	}
	artistHistory, err := s.discovery.PlaylistArtistHistory(ctx)
	if err != nil {
		return ReviewPool{}, err
	}

	now := s.now()
	entries := make([]scoring.CandidateScoreEntry, 0, len(eligible))
	for _, c := range eligible {
		trackLastUsedAt := scoring.FreshnessLastUsedAt(trackHistory, c.SpotifyTrackID)

		freshness, err := scoring.CalculateFreshness(trackLastUsedAt, now, scoring.DefaultFreshnessConfig())
		if err != nil {
			return ReviewPool{}, err
		}

		var artistIDs []string
		if c.Metadata != nil {
			for _, a := range c.Metadata.Artists {
				artistIDs = append(artistIDs, a.SpotifyArtistID)
			}
		}
		artistLastUsedAt := scoring.RepetitionArtistLastUsedAt(artistHistory, artistIDs)

		repetition, err := scoring.CalculateRepetitionPenalty(trackLastUsedAt, artistLastUsedAt, now, scoring.DefaultRepetitionPenaltyConfig())
		if err != nil {
			return ReviewPool{}, err
		}

		factors := scoring.Factors{
			Freshness:         &freshness.Value,
			RepetitionPenalty: &repetition.Value,
		}

		score, err := scoring.Calculate(c.ID, factors, scoring.DefaultWeights())
		if err != nil {
			return ReviewPool{}, err
		}

		entries = append(entries, scoring.CandidateScoreEntry{Candidate: c, Score: score})
	}

	ranked := scoring.Rank(entries)

	reviewEntries := make([]ReviewEntry, 0, len(ranked))
	for _, rc := range ranked {
		explanation := scoring.GenerateExplanation(scoring.ExplanationInput{Score: rc.Score})
		reviewEntries = append(reviewEntries, ReviewEntry{
			Ranked:      rc,
			Explanation: explanation,
		})
	}

	// Overlay persisted Keep/Maybe decisions (Cards #56/#57) onto this run's
	// freshly-discovered candidates — every candidate.NewCandidateTrack call
	// always produces StatusDiscovered, so a decided candidate only shows as
	// StatusSelected/StatusUnderReview via this lookup, re-applied on every
	// ReviewPool call (a refresh always reflects the persisted state). The
	// selection store guarantees a candidate ID can only ever be in one of
	// AllSelected/AllUnderReview's sets (one row per candidate_id, see
	// selection.Store), so a single pass can safely check both.
	selected, err := s.selection.AllSelected(ctx)
	if err != nil {
		return ReviewPool{}, err
	}
	underReview, err := s.selection.AllUnderReview(ctx)
	if err != nil {
		return ReviewPool{}, err
	}
	for i := range reviewEntries {
		id := string(reviewEntries[i].Ranked.Candidate.ID)
		if _, ok := selected[id]; ok {
			reviewEntries[i].Ranked.Candidate.Status = candidate.StatusSelected
		} else if _, ok := underReview[id]; ok {
			reviewEntries[i].Ranked.Candidate.Status = candidate.StatusUnderReview
		}
	}

	return ReviewPool{Entries: reviewEntries, WorkflowErrors: pool.WorkflowErrors, Failures: failures}, nil
}

// Handler exposes GET /api/candidates/review. No request body, no query
// parameters, no persistence.
func (s *Service) Handler(w http.ResponseWriter, r *http.Request) {
	pool, err := s.ReviewPool(r.Context())
	if err != nil {
		log.Printf("candidate review failed: %v", err)
		writeReviewError(w, err)
		return
	}
	writeJSON(w, pool)
}

// writeConnectionError and writeReviewError mirror discovery's own
// writeConnectionError/writeXError pattern (see
// discovery/discovery.go/recent_track_filter.go) — each package in this
// repo defines its own copy rather than exporting one, since this is a
// small, package-local concern.
func writeConnectionError(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, spotify.ErrNotConnected):
		http.Error(w, "Spotify is not connected", http.StatusServiceUnavailable)
	case errors.Is(err, spotify.ErrInvalidGrant):
		http.Error(w, "Spotify authorization required", http.StatusUnauthorized)
	default:
		return false
	}
	return true
}

func writeReviewError(w http.ResponseWriter, err error) {
	if writeConnectionError(w, err) {
		return
	}
	switch {
	case errors.Is(err, spotify.ErrOfficialPlaylistNotConfigured):
		http.Error(w, "Sound Continuum playlist is not initialized", http.StatusServiceUnavailable)
	default:
		http.Error(w, "candidate review failed", http.StatusBadGateway)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
