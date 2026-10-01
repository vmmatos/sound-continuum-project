package scoring

import (
	"sort"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
)

// CandidateScoreEntry pairs a candidate with its already-computed score —
// Rank's input. Building Score is the caller's responsibility, via
// Calculate (reusing Card #40's existing weighting/renormalization logic
// unchanged — Rank never recomputes it).
type CandidateScoreEntry struct {
	Candidate candidate.CandidateTrack
	Score     CandidateScore
}

// RankedCandidate is one entry in a ranked view: the original candidate —
// full metadata/provenance/status, untouched — its full score (factors,
// weights, FinalScore, for explainability), and its 1-based Rank.
type RankedCandidate struct {
	Rank      int
	Candidate candidate.CandidateTrack
	Score     CandidateScore
}

// Rank sorts entries by Score.FinalScore descending into a deterministic
// ranked view for curator review. It is read-only curation assistance:
// it does not call Calculate, does not mutate Candidate.Status or any
// other field, does not filter or drop any entry, and does not select
// anything — Rank 1 is not "selected."
//
// An entry whose FinalScore is nil (no positive factor was available to
// score it yet — see Calculate) sorts after every entry with a non-nil
// FinalScore: "nothing to report yet" is the weakest state, not a
// fabricated zero.
//
// Ties — including among nil-FinalScore entries — break on
// Candidate.ID ascending. ID is used rather than SpotifyTrackID because
// every valid CandidateTrack has a non-empty ID regardless of Source
// (see CandidateTrack.Validate), while SpotifyTrackID is only guaranteed
// non-empty when Source == SourceSpotify.
func Rank(entries []CandidateScoreEntry) []RankedCandidate {
	sorted := make([]CandidateScoreEntry, len(entries))
	copy(sorted, entries)

	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i].Score.FinalScore, sorted[j].Score.FinalScore
		switch {
		case a != nil && b != nil && *a != *b:
			return *a > *b
		case a != nil && b == nil:
			return true
		case a == nil && b != nil:
			return false
		default:
			return sorted[i].Candidate.ID < sorted[j].Candidate.ID
		}
	})

	ranked := make([]RankedCandidate, len(sorted))
	for i, e := range sorted {
		ranked[i] = RankedCandidate{
			Rank:      i + 1,
			Candidate: e.Candidate,
			Score:     e.Score,
		}
	}
	return ranked
}
