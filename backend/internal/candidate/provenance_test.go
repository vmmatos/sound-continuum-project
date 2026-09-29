package candidate

import (
	"errors"
	"reflect"
	"testing"
)

func TestDiscoveryMethodValid(t *testing.T) {
	valid := []DiscoveryMethod{
		DiscoveryMethodClassicReferenceArtist,
		DiscoveryMethodCurrentReferenceArtist,
		DiscoveryMethodLastFMSimilarArtist,
		DiscoveryMethodManual,
	}
	for _, m := range valid {
		if !m.Valid() {
			t.Errorf("DiscoveryMethod(%q).Valid() = false, want true", m)
		}
	}
	for _, m := range []DiscoveryMethod{"", "spotify_search", "unknown"} {
		if m.Valid() {
			t.Errorf("DiscoveryMethod(%q).Valid() = true, want false", m)
		}
	}
}

func TestCandidateTrackValidateInvalidDiscoveryMethod(t *testing.T) {
	c := CandidateTrack{
		ID:             "cand-1",
		SpotifyTrackID: "spotify-track-1",
		Source:         SourceSpotify,
		Category:       CategoryPast,
		Type:           TypeClassic,
		Status:         StatusDiscovered,
		Provenance:     []DiscoveryProvenance{{Method: "not_a_real_method"}},
	}
	if err := c.Validate(); !errors.Is(err, ErrInvalidDiscoveryMethod) {
		t.Errorf("Validate() = %v, want ErrInvalidDiscoveryMethod", err)
	}
}

func TestNewCandidateTrackManualProvenanceNoFabricatedData(t *testing.T) {
	// A manually added candidate can carry manual provenance without any
	// provider/seed info being invented for it.
	c, err := NewCandidateTrack(NewCandidateTrackParams{
		ID:             "cand-1",
		SpotifyTrackID: "spotify-track-1",
		Source:         SourceSpotify,
		Category:       CategoryPresent,
		Type:           TypeCurrent,
		Provenance:     []DiscoveryProvenance{{Method: DiscoveryMethodManual}},
	})
	if err != nil {
		t.Fatalf("NewCandidateTrack returned error: %v", err)
	}
	if len(c.Provenance) != 1 || c.Provenance[0].Method != DiscoveryMethodManual {
		t.Fatalf("Provenance = %+v, want one manual entry", c.Provenance)
	}
	if c.Provenance[0].Provider != "" || c.Provenance[0].Seed != nil || c.Provenance[0].DiscoveredArtist != nil {
		t.Errorf("manual provenance has fabricated data: %+v", c.Provenance[0])
	}
}

func TestCandidateTrackNoProvenanceIsValid(t *testing.T) {
	// Provenance is optional — not every caller sets it (matches
	// DiscoveryReason's own optionality).
	c, err := NewCandidateTrack(NewCandidateTrackParams{
		ID:             "cand-1",
		SpotifyTrackID: "spotify-track-1",
		Source:         SourceSpotify,
		Category:       CategoryPast,
		Type:           TypeClassic,
	})
	if err != nil {
		t.Fatalf("NewCandidateTrack returned error: %v", err)
	}
	if c.Provenance != nil {
		t.Errorf("Provenance = %+v, want nil", c.Provenance)
	}
}

func classicProvenance(artistID, name string) DiscoveryProvenance {
	return DiscoveryProvenance{
		Method:   DiscoveryMethodClassicReferenceArtist,
		Provider: ProvenanceProviderSpotify,
		Seed:     &SeedArtist{Provider: ProvenanceProviderSpotify, ProviderArtistID: artistID, Name: name},
	}
}

func lastfmProvenance(seed, discoveredID, discoveredName string, match float64) DiscoveryProvenance {
	return DiscoveryProvenance{
		Method:           DiscoveryMethodLastFMSimilarArtist,
		Provider:         ProvenanceProviderLastFM,
		Seed:             &SeedArtist{Name: seed},
		DiscoveredArtist: &SeedArtist{Provider: ProvenanceProviderSpotify, ProviderArtistID: discoveredID, Name: discoveredName},
		LastFMMatch:      &match,
	}
}

func TestMergeProvenanceUnion(t *testing.T) {
	a := []DiscoveryProvenance{classicProvenance("bowie-id", "David Bowie")}
	b := []DiscoveryProvenance{lastfmProvenance("Rosalía", "bowie-id", "David Bowie", 0.8)}

	got := MergeProvenance(a, b)
	if len(got) != 2 {
		t.Fatalf("MergeProvenance returned %d entries, want 2: %+v", len(got), got)
	}
	if got[0].Method != DiscoveryMethodClassicReferenceArtist || got[1].Method != DiscoveryMethodLastFMSimilarArtist {
		t.Errorf("MergeProvenance did not preserve both entries in order: %+v", got)
	}
}

func TestMergeProvenanceDedupesExactDuplicates(t *testing.T) {
	a := []DiscoveryProvenance{classicProvenance("bowie-id", "David Bowie")}
	b := []DiscoveryProvenance{classicProvenance("bowie-id", "David Bowie")}

	got := MergeProvenance(a, b)
	if len(got) != 1 {
		t.Fatalf("MergeProvenance returned %d entries, want 1 (duplicate collapsed): %+v", len(got), got)
	}
}

func TestMergeProvenanceEmptyInputs(t *testing.T) {
	if got := MergeProvenance(nil, nil); len(got) != 0 {
		t.Errorf("MergeProvenance(nil, nil) = %+v, want empty", got)
	}
	a := []DiscoveryProvenance{classicProvenance("bowie-id", "David Bowie")}
	if got := MergeProvenance(a, nil); !reflect.DeepEqual(got, a) {
		t.Errorf("MergeProvenance(a, nil) = %+v, want %+v", got, a)
	}
	if got := MergeProvenance(nil, a); !reflect.DeepEqual(got, a) {
		t.Errorf("MergeProvenance(nil, a) = %+v, want %+v", got, a)
	}
}
