package scoring

import (
	"testing"

	"github.com/vmmatos/sound-continuum-project/internal/musicaldna"
)

func bridgeStrPtr(s string) *string {
	return &s
}

func bridgeFloatPtr(f float64) *float64 {
	return &f
}

// musicaldnaProfile builds a musicaldna.Profile from plain strings for test
// readability; an empty string means "dimension not supplied" (nil), never
// a literal empty-string value.
func musicaldnaProfile(mood, energy, texture, cultural string) musicaldna.Profile {
	p := musicaldna.Profile{}
	if mood != "" {
		p.Mood = bridgeStrPtr(mood)
	}
	if energy != "" {
		p.Energy = bridgeStrPtr(energy)
	}
	if texture != "" {
		p.Texture = bridgeStrPtr(texture)
	}
	if cultural != "" {
		p.CulturalInfluence = bridgeStrPtr(cultural)
	}
	return p
}

// --- 1. Strong relationship ---

func TestDetectPotentialBridgeStrongRelationshipIsTrue(t *testing.T) {
	a := BridgeTrack{
		Sound: musicaldnaProfile("melancholic", "medium", "acoustic", "latin"),
	}
	b := BridgeTrack{
		Sound: musicaldnaProfile("melancholic", "medium", "acoustic", "latin"),
	}

	result := DetectPotentialBridge(a, b, nil)

	if !result.PotentialBridge {
		t.Fatalf("expected PotentialBridge true for several matching dimensions, got EvidenceCount=%d", result.EvidenceCount)
	}
	if result.EvidenceCount < 2 {
		t.Fatalf("expected evidence count >= 2, got %d", result.EvidenceCount)
	}
}

// --- 2. Partial relationship ---

func TestDetectPotentialBridgeSingleRelationshipIsFalse(t *testing.T) {
	a := BridgeTrack{Sound: musicaldnaProfile("melancholic", "", "", "")}
	b := BridgeTrack{Sound: musicaldnaProfile("melancholic", "", "", "")}

	result := DetectPotentialBridge(a, b, nil)

	if result.PotentialBridge {
		t.Fatalf("expected PotentialBridge false with only one meaningful relationship, got EvidenceCount=%d", result.EvidenceCount)
	}
	if result.EvidenceCount != 1 {
		t.Fatalf("expected EvidenceCount 1, got %d", result.EvidenceCount)
	}
}

func TestDetectPotentialBridgeTwoRelationshipsIsTrue(t *testing.T) {
	a := BridgeTrack{Sound: musicaldnaProfile("melancholic", "", "", "")}
	b := BridgeTrack{Sound: musicaldnaProfile("melancholic", "", "", "")}

	result := DetectPotentialBridge(a, b, bridgeFloatPtr(0.8))

	if !result.PotentialBridge {
		t.Fatalf("expected PotentialBridge true with two meaningful relationships, got EvidenceCount=%d", result.EvidenceCount)
	}
	if result.EvidenceCount != 2 {
		t.Fatalf("expected EvidenceCount 2, got %d", result.EvidenceCount)
	}
}

// --- 3. Energy progression: gradual vs. abrupt ---

func TestDetectPotentialBridgeEnergyProgression(t *testing.T) {
	tests := []struct {
		name         string
		from, to     string
		wantEvidence bool
		wantRel      BridgeRelationship
	}{
		{"same level", "medium", "medium", true, BridgeRelationshipMatching},
		{"adjacent: low to medium", "low", "medium", true, BridgeRelationshipProgression},
		{"adjacent: medium to high", "medium", "high", true, BridgeRelationshipProgression},
		{"two apart: low to high", "low", "high", false, BridgeRelationshipContrasting},
		{"abrupt: low to very high", "low", "very high", false, BridgeRelationshipContrasting},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := BridgeTrack{Sound: musicaldnaProfile("", tt.from, "", "")}
			b := BridgeTrack{Sound: musicaldnaProfile("", tt.to, "", "")}

			result := DetectPotentialBridge(a, b, nil)
			energy := findBridgeDimension(t, result, BridgeDimensionEnergy)

			if energy.Evidence != tt.wantEvidence {
				t.Errorf("%s -> %s: evidence = %v, want %v (score=%v)", tt.from, tt.to, energy.Evidence, tt.wantEvidence, *energy.Score)
			}
			if energy.Relationship != tt.wantRel {
				t.Errorf("%s -> %s: relationship = %v, want %v", tt.from, tt.to, energy.Relationship, tt.wantRel)
			}
		})
	}
}

// --- 4. Contrast is not automatically rejected ---

func TestDetectPotentialBridgeContrastNotAutomaticallyRejected(t *testing.T) {
	// Mood/Texture/CulturalInfluence all contrast, but a shared artist and a
	// matching era provide two independent meaningful relationships.
	a := BridgeTrack{
		ArtistSpotifyIDs: []string{"artist-1"},
		Era:              bridgeStrPtr("1990s"),
		Sound:            musicaldnaProfile("euphoric", "", "acoustic", "latin"),
	}
	b := BridgeTrack{
		ArtistSpotifyIDs: []string{"artist-1"},
		Era:              bridgeStrPtr("1990s"),
		Sound:            musicaldnaProfile("melancholic", "", "electronic", "nordic"),
	}

	result := DetectPotentialBridge(a, b, nil)

	if !result.PotentialBridge {
		t.Fatalf("expected contrast on explicit dimensions not to block a bridge supported by other evidence, got EvidenceCount=%d", result.EvidenceCount)
	}

	for _, d := range result.Dimensions {
		if d.Dimension == BridgeDimensionMood || d.Dimension == BridgeDimensionTexture || d.Dimension == BridgeDimensionCulturalInfluence {
			if d.Relationship != BridgeRelationshipContrasting {
				t.Errorf("expected %s to be contrasting, got %s", d.Dimension, d.Relationship)
			}
			if d.Evidence {
				t.Errorf("expected contrasting %s to not count as evidence", d.Dimension)
			}
		}
	}
}

// --- 5. No meaningful evidence ---

func TestDetectPotentialBridgeNoMeaningfulEvidenceIsFalse(t *testing.T) {
	a := BridgeTrack{
		Sound: musicaldnaProfile("euphoric", "low", "acoustic", "latin"),
	}
	b := BridgeTrack{
		Sound: musicaldnaProfile("melancholic", "high", "electronic", "nordic"),
	}

	result := DetectPotentialBridge(a, b, nil)

	if result.PotentialBridge {
		t.Fatalf("expected PotentialBridge false with no shared dimensions/signals, got EvidenceCount=%d", result.EvidenceCount)
	}
	if result.EvidenceCount != 0 {
		t.Fatalf("expected EvidenceCount 0, got %d", result.EvidenceCount)
	}
}

// --- 6. Last.fm signal contributes but does not independently force a bridge ---

func TestDetectPotentialBridgeLastFMAloneDoesNotForceBridge(t *testing.T) {
	a := BridgeTrack{Sound: musicaldnaProfile("euphoric", "low", "acoustic", "latin")}
	b := BridgeTrack{Sound: musicaldnaProfile("melancholic", "high", "electronic", "nordic")}

	result := DetectPotentialBridge(a, b, bridgeFloatPtr(1.0))

	lastfm := findBridgeSignal(t, result, BridgeSignalLastFMArtistSimilarity)
	if !lastfm.Present {
		t.Fatalf("expected Last.fm similarity to register as evidence")
	}
	if result.PotentialBridge {
		t.Fatalf("expected a single Last.fm signal to not independently force a bridge, got EvidenceCount=%d", result.EvidenceCount)
	}
}

func TestDetectPotentialBridgeLastFMUnavailableWhenNil(t *testing.T) {
	a := BridgeTrack{Sound: musicaldnaProfile("euphoric", "", "", "")}
	b := BridgeTrack{Sound: musicaldnaProfile("melancholic", "", "", "")}

	result := DetectPotentialBridge(a, b, nil)

	lastfm := findBridgeSignal(t, result, BridgeSignalLastFMArtistSimilarity)
	if lastfm.Available {
		t.Fatalf("expected Last.fm signal unavailable when no match value supplied")
	}
}

// Genre cannot contribute evidence because DetectPotentialBridge's
// signature has no genre parameter at all — a compile-time guarantee, not
// something a unit test can additionally exercise.

// --- 8. Missing data is not automatically negative ---

func TestDetectPotentialBridgeMissingDimensionsAreExcludedNotNegative(t *testing.T) {
	a := BridgeTrack{Sound: musicaldnaProfile("melancholic", "", "", "")}
	b := BridgeTrack{Sound: musicaldnaProfile("melancholic", "", "", "")}

	result := DetectPotentialBridge(a, b, nil)

	for _, d := range result.Dimensions {
		if d.Dimension == BridgeDimensionMood {
			continue
		}
		if d.Available {
			t.Fatalf("expected %s unavailable in this fixture", d.Dimension)
		}
		if d.Evidence {
			t.Fatalf("missing dimension %s must never count as evidence", d.Dimension)
		}
		if d.Relationship != BridgeRelationshipUnavailable {
			t.Fatalf("expected %s relationship unavailable, got %s", d.Dimension, d.Relationship)
		}
	}
}

func TestDetectPotentialBridgeMissingArtistIDsLeavesSharedArtistUnavailable(t *testing.T) {
	a := BridgeTrack{Sound: musicaldnaProfile("melancholic", "", "", "")}
	b := BridgeTrack{Sound: musicaldnaProfile("melancholic", "", "", "")}

	result := DetectPotentialBridge(a, b, nil)

	shared := findBridgeSignal(t, result, BridgeSignalSharedArtist)
	if shared.Available {
		t.Fatalf("expected shared artist signal unavailable with no artist IDs on either side")
	}
	if shared.Present {
		t.Fatalf("unavailable shared artist signal must never count as evidence")
	}
}

// --- 9. Determinism ---

func TestDetectPotentialBridgeIsDeterministic(t *testing.T) {
	a := BridgeTrack{
		ArtistSpotifyIDs: []string{"artist-1"},
		Era:              bridgeStrPtr("1990s"),
		Sound:            musicaldnaProfile("melancholic", "medium", "acoustic", "latin"),
	}
	b := BridgeTrack{
		ArtistSpotifyIDs: []string{"artist-2"},
		Era:              bridgeStrPtr("1990s"),
		Sound:            musicaldnaProfile("melancholic", "high", "electronic", "nordic"),
	}

	first := DetectPotentialBridge(a, b, bridgeFloatPtr(0.6))
	second := DetectPotentialBridge(a, b, bridgeFloatPtr(0.6))

	if first.PotentialBridge != second.PotentialBridge || first.EvidenceCount != second.EvidenceCount {
		t.Fatalf("expected identical inputs to produce identical results, got %+v vs %+v", first, second)
	}
}

// --- 10. Independence: signature carries no scoring/candidate types ---
//
// DetectPotentialBridge's signature (BridgeTrack, BridgeTrack, *float64) has
// no candidate.CandidateTrack, no CandidateScore, no Factors, and no other
// factor's computed value — it is structurally impossible for it to depend
// on CandidateScore, Freshness, Diversity, Repetition Penalty, Playlist
// Fit, popularity, or release popularity. This is a compile-time guarantee,
// not something a unit test can additionally falsify; the determinism test
// above and the fixtures throughout this file are the behavioral evidence.

func findBridgeDimension(t *testing.T, result BridgeResult, dim BridgeDimension) BridgeDimensionResult {
	t.Helper()
	for _, d := range result.Dimensions {
		if d.Dimension == dim {
			return d
		}
	}
	t.Fatalf("dimension %s not found in result", dim)
	return BridgeDimensionResult{}
}

func findBridgeSignal(t *testing.T, result BridgeResult, sig BridgeSignal) BridgeSignalResult {
	t.Helper()
	for _, s := range result.Signals {
		if s.Signal == sig {
			return s
		}
	}
	t.Fatalf("signal %s not found in result", sig)
	return BridgeSignalResult{}
}
