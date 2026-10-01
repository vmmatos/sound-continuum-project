package scoring

import (
	"math"
	"reflect"
	"testing"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
)

func testCandidate(t *testing.T, id string) candidate.CandidateTrack {
	t.Helper()
	c, err := candidate.NewCandidateTrack(candidate.NewCandidateTrackParams{
		ID:             candidate.ID(id),
		SpotifyTrackID: "spotify-" + id,
		Source:         candidate.SourceSpotify,
		Category:       candidate.CategoryEmerging,
		Type:           candidate.TypeDiscovery,
		TrackTitle:     "Track " + id,
		TrackArtist:    "Artist " + id,
		Provenance: []candidate.DiscoveryProvenance{
			{Method: candidate.DiscoveryMethodLastFMSimilarArtist, Provider: candidate.ProvenanceProviderLastFM},
		},
	})
	if err != nil {
		t.Fatalf("testCandidate(%q): %v", id, err)
	}
	c.Metadata = &candidate.CandidateMetadata{Title: "Track " + id}
	return c
}

func scoreWithFinal(t *testing.T, id string, final *float64) CandidateScore {
	t.Helper()
	return CandidateScore{
		CandidateID:  candidate.ID(id),
		ModelVersion: ModelVersion,
		Weights:      DefaultWeights(),
		FinalScore:   final,
	}
}

func TestRankBasicOrdering(t *testing.T) {
	entries := []CandidateScoreEntry{
		{Candidate: testCandidate(t, "a"), Score: scoreWithFinal(t, "a", float64Ptr(0.50))},
		{Candidate: testCandidate(t, "b"), Score: scoreWithFinal(t, "b", float64Ptr(0.90))},
		{Candidate: testCandidate(t, "c"), Score: scoreWithFinal(t, "c", float64Ptr(0.70))},
	}

	ranked := Rank(entries)

	wantOrder := []string{"b", "c", "a"}
	for i, id := range wantOrder {
		if string(ranked[i].Candidate.ID) != id {
			t.Errorf("position %d: candidate ID = %q, want %q", i, ranked[i].Candidate.ID, id)
		}
		if ranked[i].Rank != i+1 {
			t.Errorf("position %d: Rank = %d, want %d", i, ranked[i].Rank, i+1)
		}
	}
}

func TestRankWeightIntegration(t *testing.T) {
	weights := DefaultWeights()
	factors := Factors{
		Fit:            float64Ptr(0.8),
		Freshness:      float64Ptr(0.5),
		DiscoveryBonus: float64Ptr(0.9),
		Diversity:      float64Ptr(0.6),
		PlaylistFit:    float64Ptr(0.7),
	}
	score, err := Calculate(candidate.ID("a"), factors, weights)
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}

	ranked := Rank([]CandidateScoreEntry{{Candidate: testCandidate(t, "a"), Score: score}})

	if ranked[0].Score.Weights != weights {
		t.Errorf("Rank altered Weights: got %+v, want %+v", ranked[0].Score.Weights, weights)
	}
	if ranked[0].Score.FinalScore == nil || *ranked[0].Score.FinalScore != *score.FinalScore {
		t.Errorf("Rank altered FinalScore: got %v, want %v", ranked[0].Score.FinalScore, score.FinalScore)
	}
}

func TestRankMissingPositiveFactorsRenormalized(t *testing.T) {
	weights := DefaultWeights()

	// Fit and Freshness available, everything else nil — must renormalize
	// over Fit+Freshness only, not treat the missing ones as 0.
	partial := Factors{Fit: float64Ptr(0.8), Freshness: float64Ptr(0.4)}
	partialScore, err := Calculate(candidate.ID("partial"), partial, weights)
	if err != nil {
		t.Fatalf("Calculate(partial): %v", err)
	}
	wantPartial := (0.8*weights.Fit + 0.4*weights.Freshness) / (weights.Fit + weights.Freshness)
	if partialScore.FinalScore == nil || !floatsClose(*partialScore.FinalScore, wantPartial) {
		t.Fatalf("partial FinalScore = %v, want %v", partialScore.FinalScore, wantPartial)
	}

	// Explicit zero must survive Calculate and Rank unchanged, never
	// read as "missing".
	zero := Factors{Fit: float64Ptr(0.0), Freshness: float64Ptr(0.4)}
	zeroScore, err := Calculate(candidate.ID("zero"), zero, weights)
	if err != nil {
		t.Fatalf("Calculate(zero): %v", err)
	}
	if zeroScore.Factors.Fit == nil || *zeroScore.Factors.Fit != 0.0 {
		t.Fatalf("explicit zero Fit collapsed: got %v", zeroScore.Factors.Fit)
	}

	ranked := Rank([]CandidateScoreEntry{
		{Candidate: testCandidate(t, "partial"), Score: partialScore},
		{Candidate: testCandidate(t, "zero"), Score: zeroScore},
	})
	if string(ranked[0].Candidate.ID) != "partial" {
		t.Errorf("ranked[0] = %q, want %q (higher renormalized FinalScore)", ranked[0].Candidate.ID, "partial")
	}
}

func TestRankRepetitionPenaltyReducesFinalScore(t *testing.T) {
	weights := DefaultWeights()
	factors := Factors{Fit: float64Ptr(0.8)}

	noPenalty, err := Calculate(candidate.ID("a"), factors, weights)
	if err != nil {
		t.Fatalf("Calculate(noPenalty): %v", err)
	}
	factors.RepetitionPenalty = float64Ptr(1.0)
	withPenalty, err := Calculate(candidate.ID("a"), factors, weights)
	if err != nil {
		t.Fatalf("Calculate(withPenalty): %v", err)
	}

	wantWithPenalty := *noPenalty.FinalScore * (1 - 1.0*weights.RepetitionWeight)
	if !floatsClose(*withPenalty.FinalScore, wantWithPenalty) {
		t.Fatalf("withPenalty FinalScore = %v, want %v", *withPenalty.FinalScore, wantWithPenalty)
	}

	ranked := Rank([]CandidateScoreEntry{
		{Candidate: testCandidate(t, "no-penalty"), Score: noPenalty},
		{Candidate: testCandidate(t, "with-penalty"), Score: withPenalty},
	})
	if string(ranked[0].Candidate.ID) != "no-penalty" {
		t.Errorf("ranked[0] = %q, want %q (repetition penalty should rank lower)", ranked[0].Candidate.ID, "no-penalty")
	}
}

func TestRankTieBreakDeterministic(t *testing.T) {
	entries := []CandidateScoreEntry{
		{Candidate: testCandidate(t, "zz"), Score: scoreWithFinal(t, "zz", float64Ptr(0.5))},
		{Candidate: testCandidate(t, "aa"), Score: scoreWithFinal(t, "aa", float64Ptr(0.5))},
		{Candidate: testCandidate(t, "mm"), Score: scoreWithFinal(t, "mm", float64Ptr(0.5))},
	}

	var prev []string
	for i := 0; i < 5; i++ {
		ranked := Rank(entries)
		var order []string
		for _, r := range ranked {
			order = append(order, string(r.Candidate.ID))
		}
		if prev != nil && !reflect.DeepEqual(prev, order) {
			t.Fatalf("run %d: order changed: got %v, want %v", i, order, prev)
		}
		prev = order
	}

	want := []string{"aa", "mm", "zz"}
	if !reflect.DeepEqual(prev, want) {
		t.Errorf("tie-break order = %v, want %v (Candidate.ID ascending)", prev, want)
	}
}

func TestRankPreservesCandidateMetadata(t *testing.T) {
	c := testCandidate(t, "a")
	ranked := Rank([]CandidateScoreEntry{{Candidate: c, Score: scoreWithFinal(t, "a", float64Ptr(0.5))}})

	if !reflect.DeepEqual(ranked[0].Candidate, c) {
		t.Errorf("Candidate was altered by Rank:\ngot  %+v\nwant %+v", ranked[0].Candidate, c)
	}
}

func TestRankEmptyPool(t *testing.T) {
	ranked := Rank(nil)
	if len(ranked) != 0 {
		t.Errorf("Rank(nil): len = %d, want 0", len(ranked))
	}
	ranked = Rank([]CandidateScoreEntry{})
	if len(ranked) != 0 {
		t.Errorf("Rank([]CandidateScoreEntry{}): len = %d, want 0", len(ranked))
	}
}

func TestRankSingleCandidate(t *testing.T) {
	ranked := Rank([]CandidateScoreEntry{
		{Candidate: testCandidate(t, "only"), Score: scoreWithFinal(t, "only", float64Ptr(0.3))},
	})
	if len(ranked) != 1 {
		t.Fatalf("len(ranked) = %d, want 1", len(ranked))
	}
	if ranked[0].Rank != 1 {
		t.Errorf("Rank = %d, want 1", ranked[0].Rank)
	}
}

// TestRankIndependence confirms, by construction, that Rank's signature
// carries no popularity/release-date/genre/Last.fm-similarity input: two
// entries differing only in FinalScore must order purely by FinalScore,
// regardless of any other field on Candidate/Score.
func TestRankIndependence(t *testing.T) {
	low := testCandidate(t, "low")
	high := testCandidate(t, "high")
	// Give the lower-scored candidate "richer" metadata/provenance than
	// the higher-scored one — none of it should affect ordering.
	low.Provenance = append(low.Provenance, candidate.DiscoveryProvenance{
		Method:      candidate.DiscoveryMethodLastFMSimilarArtist,
		LastFMMatch: float64Ptr(1.0),
	})

	ranked := Rank([]CandidateScoreEntry{
		{Candidate: low, Score: scoreWithFinal(t, "low", float64Ptr(0.2))},
		{Candidate: high, Score: scoreWithFinal(t, "high", float64Ptr(0.95))},
	})

	if string(ranked[0].Candidate.ID) != "high" {
		t.Errorf("ranked[0] = %q, want %q — ordering must track only FinalScore", ranked[0].Candidate.ID, "high")
	}
}

func TestRankNilFinalScoreSortsLast(t *testing.T) {
	entries := []CandidateScoreEntry{
		{Candidate: testCandidate(t, "unscored-b"), Score: scoreWithFinal(t, "unscored-b", nil)},
		{Candidate: testCandidate(t, "scored"), Score: scoreWithFinal(t, "scored", float64Ptr(0.1))},
		{Candidate: testCandidate(t, "unscored-a"), Score: scoreWithFinal(t, "unscored-a", nil)},
	}

	ranked := Rank(entries)

	want := []string{"scored", "unscored-a", "unscored-b"}
	var got []string
	for _, r := range ranked {
		got = append(got, string(r.Candidate.ID))
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

func floatsClose(a, b float64) bool {
	const eps = 1e-9
	return math.Abs(a-b) < eps
}
