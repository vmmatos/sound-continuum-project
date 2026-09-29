package discovery

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sort"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
)

// candidateTypePriority ranks which workflow's classification wins when the
// same Spotify track is discovered by more than one workflow (a Spotify
// track ID is the only identity CandidatePool dedups on — see
// docs/memory/decisions.md for the full rule). Classic and Current resolve
// a track directly from its own reference artist's catalogue; Emerging
// surfaces it indirectly through a one-hop Last.fm similarity match onto a
// different seed artist, and DiscoverCurrent's own recent-catalogue scan
// already covers the same surface Emerging also walks. A track confirmed by
// Classic or Current is kept over the same track's Emerging classification;
// between Classic and Current, Classic (the narrower historical catalogue)
// wins. Priority is keyed by Type, not by which Result happened to be
// merged first, so it is stable no matter what order the three Discover*
// calls run in.
var candidateTypePriority = map[candidate.Type]int{
	candidate.TypeClassic:   0,
	candidate.TypeCurrent:   1,
	candidate.TypeDiscovery: 2,
}

// WorkflowError records that one of the three discovery workflows aborted
// entirely (a Spotify connection failure or, for Emerging, a Last.fm
// configuration failure) rather than merely losing an individual
// artist/album/track — those partial failures stay on the workflow's own
// Result.Failures instead. A workflow that aborts contributes no candidates
// to the pool, but the other two still run and their candidates are kept.
type WorkflowError struct {
	Workflow string // "classic", "current", or "emerging"
	Err      string
}

// CandidatePool is the merged, deduplicated output of one discovery run
// across Classic, Current, and Emerging discovery — Sound Continuum's raw
// editorial material, not yet reviewed or selected. It is a discovery
// result, not an analytics or ranking model: no scores, no ordering by
// popularity, no editorial decisions.
type CandidatePool struct {
	// Candidates is deduplicated by Spotify track ID and ordered
	// deterministically (Type priority, then TrackArtist, TrackTitle,
	// SpotifyTrackID) — never by popularity or any recommendation signal.
	Candidates []candidate.CandidateTrack

	TotalCandidates    int
	ClassicCandidates  int
	CurrentCandidates  int
	EmergingCandidates int
	DuplicatesRemoved  int

	// ClassicResult/CurrentResult/EmergingResult are each workflow's raw,
	// pre-dedup Result, so a discovery run stays fully inspectable (per-
	// workflow counts, unresolved artists, per-item failures, Emerging
	// provenance). Zero-valued when that workflow aborted — see
	// WorkflowErrors.
	ClassicResult  Result
	CurrentResult  Result
	EmergingResult Result

	WorkflowErrors []WorkflowError

	// RecentTrackFilter splits Candidates into editorially eligible and
	// recently-used (Card #37) — see FilterRecentTracks. Populated by
	// PoolHandler; DiscoverPool itself does not run the filter, since a
	// filter failure must surface as an HTTP error rather than silently
	// becoming part of this always-succeeds struct.
	RecentTrackFilter RecentTrackFilterResult

	// MetadataEnrichment reports EnrichCandidateMetadata's run over
	// RecentTrackFilter.EligibleCandidates (Card #38) — recently-used
	// candidates are never enriched. Populated by PoolHandler for the same
	// reason as RecentTrackFilter above.
	MetadataEnrichment EnrichmentResult
}

// DiscoverPool runs Classic, Current, and Emerging discovery, then merges
// and deduplicates their candidates into one CandidatePool. It does not
// duplicate any of the three workflows' own logic — it orchestrates the
// existing DiscoverClassic/DiscoverCurrent/DiscoverEmerging methods.
//
// All three workflows always run, independent of one another. A workflow
// that aborts (Spotify connection failure, or for Emerging a Last.fm
// configuration failure) contributes no candidates but is recorded on
// WorkflowErrors — it never silently becomes an empty result, and it never
// discards the other two workflows' successful candidates. DiscoverPool
// itself never returns an error.
func (s *Service) DiscoverPool(ctx context.Context) CandidatePool {
	var pool CandidatePool

	if r, err := s.DiscoverClassic(ctx); err != nil {
		pool.WorkflowErrors = append(pool.WorkflowErrors, WorkflowError{Workflow: "classic", Err: err.Error()})
	} else {
		pool.ClassicResult = r
	}

	if r, err := s.DiscoverCurrent(ctx); err != nil {
		pool.WorkflowErrors = append(pool.WorkflowErrors, WorkflowError{Workflow: "current", Err: err.Error()})
	} else {
		pool.CurrentResult = r
	}

	if r, err := s.DiscoverEmerging(ctx); err != nil {
		pool.WorkflowErrors = append(pool.WorkflowErrors, WorkflowError{Workflow: "emerging", Err: err.Error()})
	} else {
		pool.EmergingResult = r
	}

	// Merge the three embedded Results' candidates, deduplicating by
	// Spotify track ID via candidateTypePriority. Each workflow already
	// deduplicates within its own run; this only handles the same track
	// appearing across more than one workflow.
	var all []candidate.CandidateTrack
	all = append(all, pool.ClassicResult.Candidates...)
	all = append(all, pool.CurrentResult.Candidates...)
	all = append(all, pool.EmergingResult.Candidates...)

	bySpotifyID := make(map[string]candidate.CandidateTrack, len(all))
	duplicatesRemoved := 0
	for _, c := range all {
		existing, ok := bySpotifyID[c.SpotifyTrackID]
		if !ok {
			bySpotifyID[c.SpotifyTrackID] = c
			continue
		}
		duplicatesRemoved++
		// Deduplication removes the duplicate candidate, not its
		// provenance: the surviving classification is still decided by
		// candidateTypePriority, but both candidates' discovery paths are
		// kept.
		survivor := existing
		if candidateTypePriority[c.Type] < candidateTypePriority[existing.Type] {
			survivor = c
		}
		survivor.Provenance = candidate.MergeProvenance(existing.Provenance, c.Provenance)
		bySpotifyID[c.SpotifyTrackID] = survivor
	}

	merged := make([]candidate.CandidateTrack, 0, len(bySpotifyID))
	for _, c := range bySpotifyID {
		merged = append(merged, c)
	}
	sort.Slice(merged, func(i, j int) bool {
		a, b := merged[i], merged[j]
		if pa, pb := candidateTypePriority[a.Type], candidateTypePriority[b.Type]; pa != pb {
			return pa < pb
		}
		if a.TrackArtist != b.TrackArtist {
			return a.TrackArtist < b.TrackArtist
		}
		if a.TrackTitle != b.TrackTitle {
			return a.TrackTitle < b.TrackTitle
		}
		return a.SpotifyTrackID < b.SpotifyTrackID
	})

	pool.Candidates = merged
	pool.DuplicatesRemoved = duplicatesRemoved
	pool.TotalCandidates = len(merged)
	for _, c := range merged {
		switch c.Type {
		case candidate.TypeClassic:
			pool.ClassicCandidates++
		case candidate.TypeCurrent:
			pool.CurrentCandidates++
		case candidate.TypeDiscovery:
			pool.EmergingCandidates++
		}
	}

	return pool
}

// PoolHandler exposes POST /api/candidates/pool. Runs DiscoverPool once per
// call, then FilterRecentTracks (Card #37) to split the merged candidates
// into eligible and recently-used against the official Spotify playlist.
// No persistence, no request body, no query parameters. It never modifies
// the official Spotify playlist and never selects a candidate — the pool
// is pre-editorial.
//
// A single failed discovery workflow (see WorkflowErrors) never
// invalidates the other two's candidates and still responds 200 — but a
// recent-track-filter failure (the official playlist isn't configured, or
// Spotify can't be reached) does return an HTTP error rather than a pool,
// so a temporary Spotify failure can never silently bypass the repetition
// guardrail. After filtering, PoolHandler enriches the eligible candidates
// with Spotify metadata (Card #38) — a Spotify connection failure there
// also returns an HTTP error, but an individual candidate's metadata
// lookup failure does not: that candidate stays in the response unenriched
// (Metadata nil) and recorded on MetadataEnrichment.Failures, since one
// flaky lookup among many candidates must not take down the whole pool.
func (s *Service) PoolHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pool := s.DiscoverPool(ctx)

	filtered, err := s.FilterRecentTracks(ctx, pool.Candidates)
	if err != nil {
		log.Printf("recent track filtering failed: %v", err)
		writeRecentTrackFilterError(w, err)
		return
	}
	pool.RecentTrackFilter = filtered

	enrichment, err := s.EnrichCandidateMetadata(ctx, filtered.EligibleCandidates)
	if err != nil {
		log.Printf("candidate metadata enrichment failed: %v", err)
		writeEnrichmentError(w, err)
		return
	}
	pool.RecentTrackFilter.EligibleCandidates = enrichment.EnrichedCandidates
	pool.MetadataEnrichment = enrichment

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pool)
}
