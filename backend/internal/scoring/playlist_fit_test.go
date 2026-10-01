package scoring

import (
	"testing"

	"github.com/vmmatos/sound-continuum-project/internal/musicaldna"
)

func pfProfile(mood, energy, texture, cultural string) musicaldna.Profile {
	return musicaldna.Profile{
		Mood:              fitStrPtr(mood),
		Energy:            fitStrPtr(energy),
		Texture:           fitStrPtr(texture),
		CulturalInfluence: fitStrPtr(cultural),
	}
}

func pfCtx(previous musicaldna.Profile) *CurrentEditionContext {
	return &CurrentEditionContext{Tracks: []EditionTrack{divTrack(nil, nil, previous)}}
}

// --- 1. Strong transition ---

func TestCalculatePlaylistFitStrongTransitionScoresHigh(t *testing.T) {
	profile := pfProfile("euphoric", "high", "electronic", "west african")
	got := CalculatePlaylistFit(profile, pfCtx(profile))
	if got.Value == nil {
		t.Fatal("Value = nil, want a computed value")
	}
	if *got.Value != 1.0 {
		t.Errorf("Value = %v, want 1.0 (every dimension matches exactly)", *got.Value)
	}
}

// --- 2. Weak transition ---

func TestCalculatePlaylistFitWeakTransitionScoresLow(t *testing.T) {
	previous := pfProfile("melancholic", "very low", "sparse", "west african")
	candidate := pfProfile("euphoric", "very high", "dense", "scandinavian")
	got := CalculatePlaylistFit(candidate, pfCtx(previous))
	if got.Value == nil {
		t.Fatal("Value = nil, want a computed value")
	}
	// Mood/Texture/CulturalInfluence each mismatch -> 0.5; Energy is at
	// opposite ends of the ordinal scale -> 0.0. (0.5+0.5+0.5+0.0)/4 = 0.375.
	if *got.Value != 0.375 {
		t.Errorf("Value = %v, want 0.375", *got.Value)
	}
	strong := CalculatePlaylistFit(previous, pfCtx(previous))
	if *got.Value >= *strong.Value {
		t.Errorf("weak transition Value = %v, want lower than a strong transition's %v", *got.Value, *strong.Value)
	}
}

// --- 3. Intentional contrast ---

func TestCalculatePlaylistFitIntentionalContrastScoresStrongly(t *testing.T) {
	previous := pfProfile("melancholic", "medium", "organic", "west african")
	// Mood deliberately contrasts (a meaningful shift in direction); every
	// other dimension still aligns with the previous track.
	candidate := pfProfile("euphoric", "medium", "organic", "west african")
	got := CalculatePlaylistFit(candidate, pfCtx(previous))
	if got.Value == nil {
		t.Fatal("Value = nil, want a computed value")
	}
	// (0.5 Mood + 1.0 Energy + 1.0 Texture + 1.0 CulturalInfluence) / 4 = 0.875.
	if *got.Value != 0.875 {
		t.Errorf("Value = %v, want 0.875", *got.Value)
	}
	if *got.Value < 0.8 {
		t.Errorf("Value = %v, want a strong score despite the intentional Mood contrast", *got.Value)
	}
}

// --- 4. No previous track ---

func TestCalculatePlaylistFitNilContextIsNil(t *testing.T) {
	got := CalculatePlaylistFit(pfProfile("euphoric", "high", "electronic", "west african"), nil)
	if got.Value != nil {
		t.Errorf("Value = %v, want nil", *got.Value)
	}
	if got.ContextProvided {
		t.Error("ContextProvided = true, want false for a nil context")
	}
	if got.PreviousTrackIndex != nil {
		t.Errorf("PreviousTrackIndex = %v, want nil", *got.PreviousTrackIndex)
	}
}

// --- 22. Empty edition weight renormalization (a non-nil context with zero
// tracks also yields no transition anchor, distinct from a nil context) ---

func TestCalculatePlaylistFitEmptyEditionIsNil(t *testing.T) {
	got := CalculatePlaylistFit(pfProfile("euphoric", "high", "electronic", "west african"), &CurrentEditionContext{})
	if got.Value != nil {
		t.Errorf("Value = %v, want nil", *got.Value)
	}
	if !got.ContextProvided {
		t.Error("ContextProvided = false, want true (a context was supplied, just with no tracks yet)")
	}
	if got.PreviousTrackIndex != nil {
		t.Errorf("PreviousTrackIndex = %v, want nil", *got.PreviousTrackIndex)
	}
}

// --- 5. Missing candidate profile ---

func TestCalculatePlaylistFitMissingCandidateProfileIsNil(t *testing.T) {
	previous := pfProfile("euphoric", "high", "electronic", "west african")
	got := CalculatePlaylistFit(musicaldna.Profile{}, pfCtx(previous))
	if got.Value != nil {
		t.Errorf("Value = %v, want nil", *got.Value)
	}
	for _, d := range got.Dimensions {
		if d.Available {
			t.Errorf("Dimension %s Available = true, want false (candidate has no profile at all)", d.Dimension)
		}
	}
}

// --- 6. Missing previous-track profile ---

func TestCalculatePlaylistFitMissingPreviousTrackProfileIsNil(t *testing.T) {
	candidate := pfProfile("euphoric", "high", "electronic", "west african")
	got := CalculatePlaylistFit(candidate, pfCtx(musicaldna.Profile{}))
	if got.Value != nil {
		t.Errorf("Value = %v, want nil", *got.Value)
	}
	for _, d := range got.Dimensions {
		if d.Available {
			t.Errorf("Dimension %s Available = true, want false (previous track has no profile at all)", d.Dimension)
		}
	}
}

// --- 7. Partial dimensions ---

func TestCalculatePlaylistFitPartialDimensionsRenormalize(t *testing.T) {
	previous := musicaldna.Profile{Mood: fitStrPtr("euphoric"), Energy: fitStrPtr("medium")}
	// Texture/CulturalInfluence unavailable on the candidate side.
	candidate := musicaldna.Profile{Mood: fitStrPtr("melancholic"), Energy: fitStrPtr("high")}
	got := CalculatePlaylistFit(candidate, pfCtx(previous))
	if got.Value == nil {
		t.Fatal("Value = nil, want a computed value from the two available dimensions")
	}
	// Mood mismatches -> 0.5; Energy medium->high is adjacent -> 0.75.
	// Renormalized over the two available dimensions: (0.5+0.75)/2 = 0.625.
	if *got.Value != 0.625 {
		t.Errorf("Value = %v, want 0.625", *got.Value)
	}
	for _, d := range got.Dimensions {
		switch d.Dimension {
		case PlaylistFitDimensionMood, PlaylistFitDimensionEnergy:
			if !d.Available {
				t.Errorf("Dimension %s Available = false, want true", d.Dimension)
			}
		case PlaylistFitDimensionTexture, PlaylistFitDimensionCulturalInfluence:
			if d.Available {
				t.Errorf("Dimension %s Available = true, want false (missing on candidate side)", d.Dimension)
			}
		}
	}
}

// --- 8. Energy progression ---

func TestCalculatePlaylistFitEnergyProgression(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want float64
	}{
		{"same level", "medium", "medium", 1.0},
		{"low to medium", "low", "medium", 0.75},
		{"medium to high", "medium", "high", 0.75},
		{"high to medium (symmetric)", "high", "medium", 0.75},
		{"two apart", "low", "high", 0.5},
		{"opposite ends", "very low", "very high", 0.0},
		{"unrecognized value falls back to match", "medium", "unknown descriptor", 0.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := playlistFitEnergyScore(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("playlistFitEnergyScore(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}

	// A transition toward higher energy must not score automatically
	// better than the same transition toward lower energy.
	up := playlistFitEnergyScore("low", "high")
	down := playlistFitEnergyScore("high", "low")
	if up != down {
		t.Errorf("playlistFitEnergyScore(low,high) = %v, playlistFitEnergyScore(high,low) = %v, want equal", up, down)
	}
}

// --- 9. Mood relationship without exact equality ---

func TestCalculatePlaylistFitMoodRelationshipWithoutExactEquality(t *testing.T) {
	previous := musicaldna.Profile{Mood: fitStrPtr("melancholic")}
	candidate := musicaldna.Profile{Mood: fitStrPtr("euphoric")}
	got := CalculatePlaylistFit(candidate, pfCtx(previous))
	if got.Value == nil {
		t.Fatal("Value = nil, want a computed value")
	}
	if *got.Value != 0.5 {
		t.Errorf("Value = %v, want 0.5 (a differing mood still contributes a meaningful baseline, not 0)", *got.Value)
	}
}

// --- 10. Texture relationship ---

func TestCalculatePlaylistFitTextureRelationship(t *testing.T) {
	previous := musicaldna.Profile{Texture: fitStrPtr("sparse")}
	candidate := musicaldna.Profile{Texture: fitStrPtr("dense")}
	got := CalculatePlaylistFit(candidate, pfCtx(previous))
	if got.Value == nil {
		t.Fatal("Value = nil, want a computed value")
	}
	if *got.Value != 0.5 {
		t.Errorf("Value = %v, want 0.5", *got.Value)
	}
}

// --- 11. Cultural influence relationship ---

func TestCalculatePlaylistFitCulturalInfluenceRelationship(t *testing.T) {
	previous := musicaldna.Profile{CulturalInfluence: fitStrPtr("west african")}
	candidate := musicaldna.Profile{CulturalInfluence: fitStrPtr("scandinavian")}
	got := CalculatePlaylistFit(candidate, pfCtx(previous))
	if got.Value == nil {
		t.Fatal("Value = nil, want a computed value")
	}
	if *got.Value != 0.5 {
		t.Errorf("Value = %v, want 0.5", *got.Value)
	}
}

// --- 12/13/14/15/16/17/18/19/20. Independence from other factors/fields ---
// CalculatePlaylistFit structurally takes only a musicaldna.Profile and a
// *CurrentEditionContext — no candidate.CandidateTrack, CandidateType,
// Category, Fit, Freshness, DiscoveryBonus, Diversity, RepetitionPenalty,
// popularity, or historical (non-edition) playlist usage value can be
// passed to it at all, so independence from every one of those holds by
// construction. See playlist_fit_integration_test.go for a regression test
// built on a real CandidateTrack whose unrelated fields are left
// untouched.

// --- 21. Sequence anchor ---

func TestCalculatePlaylistFitSequenceAnchorUsesImmediatePredecessor(t *testing.T) {
	trackA := musicaldna.Profile{Mood: fitStrPtr("aggressive"), Energy: fitStrPtr("very high")}
	trackB := musicaldna.Profile{Mood: fitStrPtr("tense"), Energy: fitStrPtr("high")}
	trackC := musicaldna.Profile{Mood: fitStrPtr("calm"), Energy: fitStrPtr("low")}
	candidateD := musicaldna.Profile{Mood: fitStrPtr("calm"), Energy: fitStrPtr("low")}

	ctx := &CurrentEditionContext{Tracks: []EditionTrack{
		divTrack(nil, nil, trackA),
		divTrack(nil, nil, trackB),
		divTrack(nil, nil, trackC),
	}}

	got := CalculatePlaylistFit(candidateD, ctx)
	if got.Value == nil {
		t.Fatal("Value = nil, want a computed value")
	}
	if got.PreviousTrackIndex == nil || *got.PreviousTrackIndex != 2 {
		t.Fatalf("PreviousTrackIndex = %v, want 2 (track C, the immediate predecessor)", got.PreviousTrackIndex)
	}
	// D matches C exactly -> 1.0. If A or B had been used instead, the
	// result would be lower (A/B differ from D on both dimensions).
	if *got.Value != 1.0 {
		t.Errorf("Value = %v, want 1.0 (anchored on C->D, not A->D or B->D)", *got.Value)
	}

	wrongAnchorAToD := CalculatePlaylistFit(candidateD, &CurrentEditionContext{Tracks: []EditionTrack{divTrack(nil, nil, trackA)}})
	wrongAnchorBToD := CalculatePlaylistFit(candidateD, &CurrentEditionContext{Tracks: []EditionTrack{divTrack(nil, nil, trackB)}})
	if *wrongAnchorAToD.Value == *got.Value {
		t.Error("A->D produced the same Value as C->D; the test fixture must distinguish the anchors")
	}
	if *wrongAnchorBToD.Value == *got.Value {
		t.Error("B->D produced the same Value as C->D; the test fixture must distinguish the anchors")
	}
}

// --- 25. Normalization ---

func TestCalculatePlaylistFitValueStaysNormalized(t *testing.T) {
	tests := []struct {
		name      string
		previous  musicaldna.Profile
		candidate musicaldna.Profile
	}{
		{"all match", pfProfile("a", "high", "b", "c"), pfProfile("a", "high", "b", "c")},
		{"all mismatch", pfProfile("a", "very low", "b", "c"), pfProfile("x", "very high", "y", "z")},
		{"partial", musicaldna.Profile{Mood: fitStrPtr("a")}, musicaldna.Profile{Mood: fitStrPtr("x")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculatePlaylistFit(tt.candidate, pfCtx(tt.previous))
			if got.Value == nil {
				t.Fatal("Value = nil, want a computed value")
			}
			if *got.Value < 0.0 || *got.Value > 1.0 {
				t.Errorf("Value = %v, want in [0,1]", *got.Value)
			}
		})
	}
}

// --- 26. Determinism ---

func TestCalculatePlaylistFitIsDeterministic(t *testing.T) {
	previous := pfProfile("melancholic", "medium", "organic", "west african")
	candidate := pfProfile("euphoric", "high", "electronic", "scandinavian")
	a := CalculatePlaylistFit(candidate, pfCtx(previous))
	b := CalculatePlaylistFit(candidate, pfCtx(previous))
	if *a.Value != *b.Value {
		t.Errorf("Value differs across identical calls: %v vs %v", *a.Value, *b.Value)
	}
}
