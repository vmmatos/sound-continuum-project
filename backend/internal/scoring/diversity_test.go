package scoring

import (
	"testing"

	"github.com/vmmatos/sound-continuum-project/internal/musicaldna"
)

func divTrack(artistIDs []string, era *string, sound musicaldna.Profile) EditionTrack {
	return EditionTrack{ArtistSpotifyIDs: artistIDs, Era: era, Sound: sound}
}

// --- DiversityEra ---

func TestDiversityEra(t *testing.T) {
	tests := []struct {
		name        string
		releaseDate string
		want        *string
	}{
		{"full date", "1994-05-12", fitStrPtr("1990s")},
		{"year-month", "2003-11", fitStrPtr("2000s")},
		{"year only", "1979", fitStrPtr("1970s")},
		{"decade boundary", "1980-01-01", fitStrPtr("1980s")},
		{"empty", "", nil},
		{"unparseable", "unknown", nil},
		{"too short", "19", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DiversityEra(tt.releaseDate)
			if (got == nil) != (tt.want == nil) {
				t.Fatalf("DiversityEra(%q) = %v, want %v", tt.releaseDate, got, tt.want)
			}
			if got != nil && *got != *tt.want {
				t.Errorf("DiversityEra(%q) = %v, want %v", tt.releaseDate, *got, *tt.want)
			}
		})
	}
}

// --- 1. New artist in an edition ---

func TestDiversityArtistNewArtistStrongContribution(t *testing.T) {
	ctx := &CurrentEditionContext{Tracks: []EditionTrack{
		divTrack([]string{"artist-other"}, nil, musicaldna.Profile{}),
	}}
	result := CalculateDiversity([]string{"artist-candidate"}, nil, musicaldna.Profile{}, ctx)
	if !result.Artist.Available || result.Artist.Occurrences != 0 {
		t.Fatalf("Artist = %+v, want Available with 0 occurrences", result.Artist)
	}
	if result.Artist.Value == nil || *result.Artist.Value != 1.0 {
		t.Errorf("Artist.Value = %v, want 1.0", result.Artist.Value)
	}
}

// --- 2. Repeated artist ---

func TestDiversityArtistRepeatedArtistLowerContribution(t *testing.T) {
	ctx := &CurrentEditionContext{Tracks: []EditionTrack{
		divTrack([]string{"artist-candidate"}, nil, musicaldna.Profile{}),
	}}
	result := CalculateDiversity([]string{"artist-candidate"}, nil, musicaldna.Profile{}, ctx)
	if result.Artist.Occurrences != 1 {
		t.Fatalf("Occurrences = %v, want 1", result.Artist.Occurrences)
	}
	if result.Artist.Value == nil || *result.Artist.Value >= 1.0 {
		t.Errorf("Artist.Value = %v, want < 1.0", result.Artist.Value)
	}
}

// --- 3. Increasing artist concentration decreases Artist Diversity ---

func TestDiversityArtistIncreasingConcentrationDecreasesContribution(t *testing.T) {
	none := CalculateDiversity([]string{"a"}, nil, musicaldna.Profile{}, &CurrentEditionContext{
		Tracks: []EditionTrack{divTrack([]string{"other"}, nil, musicaldna.Profile{})},
	})
	one := CalculateDiversity([]string{"a"}, nil, musicaldna.Profile{}, &CurrentEditionContext{
		Tracks: []EditionTrack{divTrack([]string{"a"}, nil, musicaldna.Profile{})},
	})
	many := CalculateDiversity([]string{"a"}, nil, musicaldna.Profile{}, &CurrentEditionContext{
		Tracks: []EditionTrack{
			divTrack([]string{"a"}, nil, musicaldna.Profile{}),
			divTrack([]string{"a"}, nil, musicaldna.Profile{}),
			divTrack([]string{"a"}, nil, musicaldna.Profile{}),
		},
	})

	if *none.Artist.Value <= *one.Artist.Value {
		t.Errorf("none.Artist.Value = %v, want > one.Artist.Value = %v", *none.Artist.Value, *one.Artist.Value)
	}
	if *one.Artist.Value <= *many.Artist.Value {
		t.Errorf("one.Artist.Value = %v, want > many.Artist.Value = %v", *one.Artist.Value, *many.Artist.Value)
	}
}

// --- 4. Different era ---

func TestDiversityEraDifferentEraStrongerContribution(t *testing.T) {
	ctx := &CurrentEditionContext{Tracks: []EditionTrack{
		divTrack(nil, fitStrPtr("1980s"), musicaldna.Profile{}),
	}}
	result := CalculateDiversity(nil, fitStrPtr("2010s"), musicaldna.Profile{}, ctx)
	if result.Era.Occurrences != 0 {
		t.Fatalf("Occurrences = %v, want 0", result.Era.Occurrences)
	}
	if *result.Era.Value != 1.0 {
		t.Errorf("Era.Value = %v, want 1.0", *result.Era.Value)
	}
}

// --- 5. Concentrated era ---

func TestDiversityEraConcentratedEraLowerContribution(t *testing.T) {
	ctx := &CurrentEditionContext{Tracks: []EditionTrack{
		divTrack(nil, fitStrPtr("2010s"), musicaldna.Profile{}),
		divTrack(nil, fitStrPtr("2010s"), musicaldna.Profile{}),
	}}
	result := CalculateDiversity(nil, fitStrPtr("2010s"), musicaldna.Profile{}, ctx)
	if result.Era.Occurrences != 2 {
		t.Fatalf("Occurrences = %v, want 2", result.Era.Occurrences)
	}
	if *result.Era.Value >= 1.0 {
		t.Errorf("Era.Value = %v, want < 1.0", *result.Era.Value)
	}
}

// --- 6. Different sound ---

func TestDiversitySoundDifferentSoundStronger(t *testing.T) {
	ctx := &CurrentEditionContext{Tracks: []EditionTrack{
		divTrack(nil, nil, musicaldna.Profile{Mood: fitStrPtr("melancholic"), Energy: fitStrPtr("low")}),
	}}
	candidate := musicaldna.Profile{Mood: fitStrPtr("euphoric"), Energy: fitStrPtr("high")}
	result := CalculateDiversity(nil, nil, candidate, ctx)
	if !result.Sound.Available || *result.Sound.Value != 1.0 {
		t.Errorf("Sound = %+v, want Available with Value 1.0", result.Sound)
	}
}

// --- 7. Similar sound ---

func TestDiversitySoundSimilarSoundLower(t *testing.T) {
	ctx := &CurrentEditionContext{Tracks: []EditionTrack{
		divTrack(nil, nil, musicaldna.Profile{Mood: fitStrPtr("Melancholic"), Energy: fitStrPtr(" Low ")}),
	}}
	candidate := musicaldna.Profile{Mood: fitStrPtr("melancholic"), Energy: fitStrPtr("low")}
	result := CalculateDiversity(nil, nil, candidate, ctx)
	if *result.Sound.Value >= 1.0 {
		t.Errorf("Sound.Value = %v, want < 1.0", *result.Sound.Value)
	}
	for _, d := range result.Sound.Dimensions {
		if d.Dimension == DiversitySoundMood || d.Dimension == DiversitySoundEnergy {
			if d.Occurrences != 1 {
				t.Errorf("%s.Occurrences = %v, want 1 (case/whitespace-insensitive match)", d.Dimension, d.Occurrences)
			}
		}
	}
}

// --- 8. Empty edition ---

func TestDiversityEmptyEditionIsNilNotMaximal(t *testing.T) {
	ctx := &CurrentEditionContext{Tracks: []EditionTrack{}}
	result := CalculateDiversity([]string{"a"}, fitStrPtr("2010s"), musicaldna.Profile{Mood: fitStrPtr("x")}, ctx)
	if !result.ContextProvided || !result.EditionEmpty {
		t.Fatalf("result = %+v, want ContextProvided=true EditionEmpty=true", result)
	}
	if result.Value != nil {
		t.Errorf("Value = %v, want nil for an empty edition (not 1.0)", *result.Value)
	}
}

// --- 9. Missing current-edition context ---

func TestDiversityMissingContextIsNil(t *testing.T) {
	result := CalculateDiversity([]string{"a"}, fitStrPtr("2010s"), musicaldna.Profile{Mood: fitStrPtr("x")}, nil)
	if result.ContextProvided {
		t.Error("ContextProvided = true, want false")
	}
	if result.Value != nil {
		t.Errorf("Value = %v, want nil when context is missing", *result.Value)
	}
}

// --- 10. Missing individual dimensions renormalize, never treated as 0 ---

func TestDiversityMissingDimensionsRenormalize(t *testing.T) {
	ctx := &CurrentEditionContext{Tracks: []EditionTrack{
		divTrack([]string{"other"}, fitStrPtr("1980s"), musicaldna.Profile{}),
	}}
	// Only Era is available on the candidate; Artist and Sound are not.
	result := CalculateDiversity(nil, fitStrPtr("2010s"), musicaldna.Profile{}, ctx)
	if result.Artist.Available || result.Sound.Available {
		t.Fatalf("Artist/Sound = %+v / %+v, want both unavailable", result.Artist, result.Sound)
	}
	if result.Value == nil || *result.Value != *result.Era.Value {
		t.Errorf("Value = %v, want exactly Era.Value = %v (renormalized over the one available dimension)", result.Value, result.Era.Value)
	}
}

// --- 11. Explicit zero distinguishable from nil ---

func TestDiversityExplicitZeroDistinctFromNil(t *testing.T) {
	// Artist available but concentration drives it arbitrarily close to
	// (never exactly) 0 is not representable by this formula (bounded
	// (0,1]), so verify instead that a present dimension's Value is never
	// nil, distinguishing "assessed" from "unavailable".
	ctx := &CurrentEditionContext{Tracks: []EditionTrack{
		divTrack([]string{"a"}, nil, musicaldna.Profile{}),
	}}
	result := CalculateDiversity([]string{"a"}, nil, musicaldna.Profile{}, ctx)
	if !result.Artist.Available || result.Artist.Value == nil {
		t.Fatalf("Artist = %+v, want Available=true with a non-nil Value", result.Artist)
	}
	notAssessed := CalculateDiversity(nil, nil, musicaldna.Profile{}, ctx)
	if notAssessed.Artist.Available || notAssessed.Artist.Value != nil {
		t.Fatalf("Artist = %+v, want Available=false with a nil Value", notAssessed.Artist)
	}
}

// --- 12/13/14/15/16/17/18/19/20. Independence from other factors/fields ---
// CalculateDiversity structurally takes no candidate.CandidateTrack,
// CandidateType, Category, Source, DiscoveryProvenance/Last.fm match,
// release date-as-Fit/Freshness/DiscoveryBonus/RepetitionPenalty/
// PlaylistFit value, or popularity — none of those can be passed to it at
// all, so independence holds by construction. See diversity_integration_test.go
// for a regression test built on a real CandidateTrack.

// --- 21. Normalization ---

func TestDiversityValuesNormalized(t *testing.T) {
	ctx := &CurrentEditionContext{Tracks: []EditionTrack{
		divTrack([]string{"a"}, fitStrPtr("1990s"), musicaldna.Profile{Mood: fitStrPtr("x"), Energy: fitStrPtr("y")}),
		divTrack([]string{"b"}, fitStrPtr("1990s"), musicaldna.Profile{Mood: fitStrPtr("x")}),
	}}
	result := CalculateDiversity([]string{"a"}, fitStrPtr("1990s"), musicaldna.Profile{Mood: fitStrPtr("x"), Energy: fitStrPtr("y")}, ctx)
	for _, v := range []*float64{result.Value, result.Artist.Value, result.Era.Value, result.Sound.Value} {
		if v == nil {
			t.Fatal("expected a non-nil value to check bounds")
		}
		if *v < 0.0 || *v > 1.0 {
			t.Errorf("value = %v, want in [0,1]", *v)
		}
	}
}

// --- 22. CandidateScore integration ---

func TestDiversityIntegratesWithCandidateScoreFactors(t *testing.T) {
	ctx := &CurrentEditionContext{Tracks: []EditionTrack{
		divTrack([]string{"other"}, nil, musicaldna.Profile{}),
	}}
	result := CalculateDiversity([]string{"a"}, nil, musicaldna.Profile{}, ctx)
	score, err := Calculate("cand-1", Factors{Diversity: result.Value}, DefaultWeights())
	if err != nil {
		t.Fatalf("Calculate() err = %v, want nil", err)
	}
	if score.Factors.Diversity == nil || *score.Factors.Diversity != *result.Value {
		t.Errorf("Factors.Diversity = %v, want %v", score.Factors.Diversity, result.Value)
	}
}

// --- 23. Existing weight unchanged ---

func TestDefaultWeightsDiversityUnchanged(t *testing.T) {
	if got := DefaultWeights().Diversity; got != 0.15 {
		t.Errorf("DefaultWeights().Diversity = %v, want 0.15 (Card #44 must not change the existing weight)", got)
	}
}

// --- 24. Missing-factor renormalization ---

func TestDiversityMissingFactorRenormalizes(t *testing.T) {
	weights := DefaultWeights()
	fitValue := 0.5
	score, err := Calculate("cand-2", Factors{Fit: &fitValue}, weights)
	if err != nil {
		t.Fatalf("Calculate() err = %v, want nil", err)
	}
	wantAvailableWeight := weights.Fit
	if diff := score.AvailableWeight - wantAvailableWeight; diff < -1e-9 || diff > 1e-9 {
		t.Errorf("AvailableWeight = %v, want %v (Diversity excluded, not treated as 0)", score.AvailableWeight, wantAvailableWeight)
	}
}

// --- 25. Artist / Era / Sound independence from each other ---

func TestDiversityDimensionsIndependentOfEachOther(t *testing.T) {
	ctx := &CurrentEditionContext{Tracks: []EditionTrack{
		divTrack([]string{"a"}, fitStrPtr("1990s"), musicaldna.Profile{Mood: fitStrPtr("x")}),
	}}
	base := CalculateDiversity([]string{"a"}, fitStrPtr("1990s"), musicaldna.Profile{Mood: fitStrPtr("x")}, ctx)

	// Changing only the candidate's era must not change Artist's or
	// Sound's raw values.
	changedEra := CalculateDiversity([]string{"a"}, fitStrPtr("2010s"), musicaldna.Profile{Mood: fitStrPtr("x")}, ctx)
	if *changedEra.Artist.Value != *base.Artist.Value {
		t.Errorf("Artist.Value changed from %v to %v after changing Era", *base.Artist.Value, *changedEra.Artist.Value)
	}
	if *changedEra.Sound.Value != *base.Sound.Value {
		t.Errorf("Sound.Value changed from %v to %v after changing Era", *base.Sound.Value, *changedEra.Sound.Value)
	}

	// Changing only the candidate's sound must not change Artist's or
	// Era's raw values.
	changedSound := CalculateDiversity([]string{"a"}, fitStrPtr("1990s"), musicaldna.Profile{Mood: fitStrPtr("y")}, ctx)
	if *changedSound.Artist.Value != *base.Artist.Value {
		t.Errorf("Artist.Value changed from %v to %v after changing Sound", *base.Artist.Value, *changedSound.Artist.Value)
	}
	if *changedSound.Era.Value != *base.Era.Value {
		t.Errorf("Era.Value changed from %v to %v after changing Sound", *base.Era.Value, *changedSound.Era.Value)
	}
}

func TestDiversityDeterministic(t *testing.T) {
	ctx := &CurrentEditionContext{Tracks: []EditionTrack{
		divTrack([]string{"a"}, fitStrPtr("1990s"), musicaldna.Profile{Mood: fitStrPtr("x")}),
	}}
	first := CalculateDiversity([]string{"b"}, fitStrPtr("2010s"), musicaldna.Profile{Mood: fitStrPtr("z")}, ctx)
	second := CalculateDiversity([]string{"b"}, fitStrPtr("2010s"), musicaldna.Profile{Mood: fitStrPtr("z")}, ctx)
	if *first.Value != *second.Value {
		t.Errorf("CalculateDiversity is not deterministic: %v != %v", *first.Value, *second.Value)
	}
}
