package scoring

import (
	"slices"
	"strings"

	"github.com/vmmatos/sound-continuum-project/internal/musicaldna"
)

// BridgeTrack is the minimal representation of one track in a candidate
// pair being evaluated for a potential musical bridge — mirrors
// EditionTrack's field shape (diversity.go) for the same reason: reuse the
// existing artist/era/sound vocabulary rather than invent a second one.
// Era is caller-derived via the existing DiversityEra.
type BridgeTrack struct {
	ArtistSpotifyIDs []string
	Era              *string
	Sound            musicaldna.Profile
}

// BridgeDimension identifies one of the four explicit musical dimensions
// compared between a track pair — the same vocabulary Fit (Card #41) and
// Playlist Fit (Card #46) already use.
type BridgeDimension string

const (
	BridgeDimensionMood              BridgeDimension = "mood"
	BridgeDimensionEnergy            BridgeDimension = "energy"
	BridgeDimensionTexture           BridgeDimension = "texture"
	BridgeDimensionCulturalInfluence BridgeDimension = "cultural_influence"
)

// BridgeSignal identifies one contextual signal considered alongside the
// four explicit dimensions. These are the only signals Card #47's
// investigation found actually available — no genre signal exists here by
// design (see DetectPotentialBridge's doc comment).
type BridgeSignal string

const (
	BridgeSignalSharedArtist           BridgeSignal = "shared_artist"
	BridgeSignalLastFMArtistSimilarity BridgeSignal = "lastfm_artist_similarity"
	BridgeSignalReleaseEra             BridgeSignal = "release_era"
)

// BridgeRelationship describes how two tracks relate on one dimension.
// "progression" only ever applies to Energy's ordinal adjacency; the three
// categorical dimensions only ever report "matching" or "contrasting".
type BridgeRelationship string

const (
	BridgeRelationshipMatching    BridgeRelationship = "matching"
	BridgeRelationshipProgression BridgeRelationship = "progression"
	BridgeRelationshipContrasting BridgeRelationship = "contrasting"
	BridgeRelationshipUnavailable BridgeRelationship = "unavailable"
)

// BridgeDimensionResult explains one dimension's contribution. Available is
// false when either side lacks this dimension — excluded entirely, never
// scored, never counted as negative evidence. Evidence is true only when
// this dimension counts toward PotentialBridge (see DetectPotentialBridge).
type BridgeDimensionResult struct {
	Dimension    BridgeDimension
	Available    bool
	Relationship BridgeRelationship
	Score        *float64
	Evidence     bool
}

// BridgeSignalResult explains one contextual signal. Available is false
// when the signal could not be assessed at all (e.g. no Last.fm match was
// supplied, or one side has no artist IDs) — this is distinct from
// Present, which is false both when unavailable and when the signal was
// assessed but found no relationship. Present counts as evidence whenever
// it is true (it is always false when Available is false).
type BridgeSignalResult struct {
	Signal    BridgeSignal
	Available bool
	Present   bool
}

// DefaultMinimumBridgeEvidence is the minimum number of independently
// corroborating signals (across the four dimensions and three contextual
// signals) required before DetectPotentialBridge reports true. The card
// explicitly forbids treating any single signal as proof of a bridge
// (e.g. Last.fm similarity alone must never force a bridge) — requiring at
// least two is the smallest rule that satisfies that constraint without
// inventing a weighted score. A single exported constant, deterministic
// and trivially changed later if the threshold proves wrong.
const DefaultMinimumBridgeEvidence = 2

// BridgeResult is the outcome of DetectPotentialBridge: PotentialBridge is
// a deterministic editorial signal, never a claim of confirmed musical
// similarity — see docs/bridge-detection.md. Dimensions and Signals make
// every contributing (and non-contributing) piece of evidence independently
// inspectable, so an editor can see exactly why a pair was or wasn't
// flagged.
type BridgeResult struct {
	PotentialBridge bool
	EvidenceCount   int
	Dimensions      []BridgeDimensionResult
	Signals         []BridgeSignalResult
}

// DetectPotentialBridge evaluates whether a pair of tracks deserves
// editorial consideration as a potential musical bridge. It is NOT a
// similarity score, NOT a ranking mechanism, and NOT a claim that the two
// tracks are musically similar — see docs/bridge-detection.md for the full
// model.
//
// It reuses playlist_fit.go's exact comparators (playlistFitMatchOrBaseline,
// playlistFitEnergyScore) rather than duplicating Mood/Energy/Texture/
// Cultural Influence comparison logic: a categorical exact match scores
// 1.0, a known mismatch scores a flat 0.5 baseline (contrast is allowed,
// never treated as negative), and Energy uses the same 5-level ordinal
// distance (1 - |levelDiff|/4) with the same safe fallback for unknown
// values.
//
// lastFMArtistMatch is supplied by the caller (e.g. from an existing
// lastfm.Client.SimilarArtists lookup, or
// candidate.DiscoveryProvenance.LastFMMatch) — this function makes no
// Last.fm or Spotify call itself and is purely deterministic.
//
// Genre is deliberately not a parameter: Card #47's research found Spotify
// genres unreliable (artist-level, optional, frequently null, no
// taxonomy), and the card forbids treating genre overlap as bridge
// evidence. Omitting it from the signature guarantees it can never
// contribute, by construction.
//
// Each of the four dimensions and three contextual signals independently
// reports whether it counts as a "meaningful relationship": a dimension
// counts when Available && Score > 0.5 (an exact match or an adjacent-
// energy progression; a 0.5 baseline mismatch or a distant energy gap is
// neutral, never negative); a contextual signal counts when Present (which
// is always false when the signal is unavailable). PotentialBridge is true
// when the total count of meaningful relationships is at least
// DefaultMinimumBridgeEvidence.
//
// a/b and lastFMArtistMatch are plain values — DetectPotentialBridge takes
// no candidate.CandidateTrack, CandidateScore, or any other scoring
// factor's value, so it is independent of Freshness, Diversity, Repetition
// Penalty, Playlist Fit, popularity, and release popularity by
// construction. It never selects, reorders, ranks, or persists anything.
func DetectPotentialBridge(a, b BridgeTrack, lastFMArtistMatch *float64) BridgeResult {
	dims := []struct {
		name  BridgeDimension
		x, y  *string
		score func(x, y string) float64
	}{
		{BridgeDimensionMood, a.Sound.Mood, b.Sound.Mood, playlistFitMatchOrBaseline},
		{BridgeDimensionEnergy, a.Sound.Energy, b.Sound.Energy, playlistFitEnergyScore},
		{BridgeDimensionTexture, a.Sound.Texture, b.Sound.Texture, playlistFitMatchOrBaseline},
		{BridgeDimensionCulturalInfluence, a.Sound.CulturalInfluence, b.Sound.CulturalInfluence, playlistFitMatchOrBaseline},
	}

	dimResults := make([]BridgeDimensionResult, 0, len(dims))
	evidenceCount := 0
	for _, d := range dims {
		if d.x == nil || d.y == nil {
			dimResults = append(dimResults, BridgeDimensionResult{
				Dimension:    d.name,
				Available:    false,
				Relationship: BridgeRelationshipUnavailable,
			})
			continue
		}

		score := d.score(*d.x, *d.y)
		evidence := score > 0.5

		relationship := BridgeRelationshipContrasting
		switch {
		case score == 1.0:
			relationship = BridgeRelationshipMatching
		case d.name == BridgeDimensionEnergy && evidence:
			relationship = BridgeRelationshipProgression
		}

		if evidence {
			evidenceCount++
		}
		dimResults = append(dimResults, BridgeDimensionResult{
			Dimension:    d.name,
			Available:    true,
			Relationship: relationship,
			Score:        &score,
			Evidence:     evidence,
		})
	}

	sharedArtist := bridgeSharedArtistSignal(a.ArtistSpotifyIDs, b.ArtistSpotifyIDs)
	lastFM := bridgeLastFMSignal(lastFMArtistMatch)
	era := bridgeEraSignal(a.Era, b.Era)

	signals := []BridgeSignalResult{sharedArtist, lastFM, era}
	for _, s := range signals {
		if s.Present {
			evidenceCount++
		}
	}

	return BridgeResult{
		PotentialBridge: evidenceCount >= DefaultMinimumBridgeEvidence,
		EvidenceCount:   evidenceCount,
		Dimensions:      dimResults,
		Signals:         signals,
	}
}

func bridgeSharedArtistSignal(a, b []string) BridgeSignalResult {
	if len(a) == 0 || len(b) == 0 {
		return BridgeSignalResult{Signal: BridgeSignalSharedArtist, Available: false}
	}

	present := slices.ContainsFunc(a, func(id string) bool {
		return slices.Contains(b, id)
	})
	return BridgeSignalResult{
		Signal:    BridgeSignalSharedArtist,
		Available: true,
		Present:   present,
	}
}

func bridgeLastFMSignal(match *float64) BridgeSignalResult {
	if match == nil {
		return BridgeSignalResult{Signal: BridgeSignalLastFMArtistSimilarity, Available: false}
	}

	present := *match > 0
	return BridgeSignalResult{
		Signal:    BridgeSignalLastFMArtistSimilarity,
		Available: true,
		Present:   present,
	}
}

func bridgeEraSignal(a, b *string) BridgeSignalResult {
	if a == nil || b == nil {
		return BridgeSignalResult{Signal: BridgeSignalReleaseEra, Available: false}
	}

	present := strings.EqualFold(strings.TrimSpace(*a), strings.TrimSpace(*b))
	return BridgeSignalResult{
		Signal:    BridgeSignalReleaseEra,
		Available: true,
		Present:   present,
	}
}
