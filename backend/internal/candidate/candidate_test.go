package candidate

import (
	"errors"
	"testing"
)

func TestNewCandidateTrackValidSpotify(t *testing.T) {
	c, err := NewCandidateTrack(NewCandidateTrackParams{
		ID:             "cand-1",
		SpotifyTrackID: "spotify-track-1",
		Source:         SourceSpotify,
		Category:       CategoryEmerging,
		Type:           TypeDiscovery,
		TrackTitle:     "Example Song",
		TrackArtist:    "Example Artist",
	})
	if err != nil {
		t.Fatalf("NewCandidateTrack returned error: %v", err)
	}
	if c.Status != StatusDiscovered {
		t.Errorf("Status = %q, want %q", c.Status, StatusDiscovered)
	}
	if c.CreatedAt.IsZero() || c.UpdatedAt.IsZero() {
		t.Error("CreatedAt/UpdatedAt were not set")
	}
}

func TestNewCandidateTrackInvalidSource(t *testing.T) {
	for _, source := range []Source{"", "spotify", "SPOTIFY", "Youtube"} {
		_, err := NewCandidateTrack(NewCandidateTrackParams{
			ID:       "cand-1",
			Source:   source,
			Category: CategoryPresent,
		})
		if !errors.Is(err, ErrInvalidSource) {
			t.Errorf("source %q: err = %v, want ErrInvalidSource", source, err)
		}
	}
}

func TestNewCandidateTrackInvalidCategory(t *testing.T) {
	// The card explicitly forbids replacing the four categories with
	// Old/Current/New — confirm they're rejected like any other bad value.
	for _, category := range []Category{"", "Old", "Current", "New", "emerging"} {
		_, err := NewCandidateTrack(NewCandidateTrackParams{
			ID:             "cand-1",
			SpotifyTrackID: "spotify-track-1",
			Source:         SourceSpotify,
			Category:       category,
		})
		if !errors.Is(err, ErrInvalidCategory) {
			t.Errorf("category %q: err = %v, want ErrInvalidCategory", category, err)
		}
	}
}

func TestValidateInvalidStatus(t *testing.T) {
	// NewCandidateTrack always sets StatusDiscovered, so an invalid status
	// can only be exercised via Validate directly.
	c := CandidateTrack{
		ID:             "cand-1",
		SpotifyTrackID: "spotify-track-1",
		Source:         SourceSpotify,
		Category:       CategoryPast,
		Type:           TypeClassic,
		Status:         "approved",
	}
	if err := c.Validate(); !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("Validate() = %v, want ErrInvalidStatus", err)
	}
}

func TestNewCandidateTrackSpotifySourceCarriesSpotifyTrackID(t *testing.T) {
	c, err := NewCandidateTrack(NewCandidateTrackParams{
		ID:             "cand-1",
		SpotifyTrackID: "spotify-track-1",
		Source:         SourceSpotify,
		Category:       CategoryNewRelease,
		Type:           TypeCurrent,
	})
	if err != nil {
		t.Fatalf("NewCandidateTrack returned error: %v", err)
	}
	if c.SpotifyTrackID != "spotify-track-1" {
		t.Errorf("SpotifyTrackID = %q, want %q", c.SpotifyTrackID, "spotify-track-1")
	}
}

func TestNewCandidateTrackManualSourceRequiresNoSpotifyTrackID(t *testing.T) {
	c, err := NewCandidateTrack(NewCandidateTrackParams{
		ID:          "cand-1",
		Source:      SourceManual,
		Category:    CategoryPast,
		Type:        TypeDiscovery,
		TrackTitle:  "A Rediscovered Classic",
		TrackArtist: "Some Artist",
	})
	if err != nil {
		t.Fatalf("NewCandidateTrack returned error: %v", err)
	}
	if c.SpotifyTrackID != "" {
		t.Errorf("SpotifyTrackID = %q, want empty for a manual candidate", c.SpotifyTrackID)
	}
}

func TestNewCandidateTrackSpotifySourceMissingSpotifyTrackID(t *testing.T) {
	_, err := NewCandidateTrack(NewCandidateTrackParams{
		ID:       "cand-1",
		Source:   SourceSpotify,
		Category: CategoryPast,
		Type:     TypeCurrent,
	})
	if !errors.Is(err, ErrMissingSpotifyTrackID) {
		t.Errorf("err = %v, want ErrMissingSpotifyTrackID", err)
	}
}

func TestNewCandidateTrackEditorialContextWithoutSpotifyData(t *testing.T) {
	c, err := NewCandidateTrack(NewCandidateTrackParams{
		ID:                  "cand-1",
		Source:              SourceLastFM,
		Category:            CategoryEmerging,
		Type:                TypeDiscovery,
		TrackTitle:          "A Track From Last.fm",
		TrackArtist:         "An Artist",
		DiscoveryReason:     "Surfaced via similar-artist lookup",
		EditorialNote:       "Could provide a transition into contemporary soul",
		PotentialConnection: "Bridges atmospheric electronic into soul",
	})
	if err != nil {
		t.Fatalf("NewCandidateTrack returned error: %v", err)
	}
	if c.DiscoveryReason == "" || c.EditorialNote == "" || c.PotentialConnection == "" {
		t.Error("editorial fields were not retained")
	}
	if c.SpotifyTrackID != "" {
		t.Errorf("SpotifyTrackID = %q, want empty for a Last.fm candidate", c.SpotifyTrackID)
	}
}

func TestNewCandidateTrackEmptyID(t *testing.T) {
	_, err := NewCandidateTrack(NewCandidateTrackParams{
		Source:   SourceManual,
		Category: CategoryPresent,
	})
	if !errors.Is(err, ErrEmptyCandidateID) {
		t.Errorf("err = %v, want ErrEmptyCandidateID", err)
	}
}

func TestNewCandidateTrackValidTypeClassic(t *testing.T) {
	c, err := NewCandidateTrack(NewCandidateTrackParams{
		ID:       "cand-1",
		Source:   SourceManual,
		Category: CategoryPast,
		Type:     TypeClassic,
	})
	if err != nil {
		t.Fatalf("NewCandidateTrack returned error: %v", err)
	}
	if c.Type != TypeClassic {
		t.Errorf("Type = %q, want %q", c.Type, TypeClassic)
	}
}

func TestNewCandidateTrackValidTypeCurrent(t *testing.T) {
	c, err := NewCandidateTrack(NewCandidateTrackParams{
		ID:       "cand-1",
		Source:   SourceManual,
		Category: CategoryNewRelease,
		Type:     TypeCurrent,
	})
	if err != nil {
		t.Fatalf("NewCandidateTrack returned error: %v", err)
	}
	if c.Type != TypeCurrent {
		t.Errorf("Type = %q, want %q", c.Type, TypeCurrent)
	}
}

func TestNewCandidateTrackValidTypeDiscovery(t *testing.T) {
	c, err := NewCandidateTrack(NewCandidateTrackParams{
		ID:       "cand-1",
		Source:   SourceManual,
		Category: CategoryEmerging,
		Type:     TypeDiscovery,
	})
	if err != nil {
		t.Fatalf("NewCandidateTrack returned error: %v", err)
	}
	if c.Type != TypeDiscovery {
		t.Errorf("Type = %q, want %q", c.Type, TypeDiscovery)
	}
}

func TestNewCandidateTrackInvalidType(t *testing.T) {
	for _, typ := range []Type{"", "classic", "CLASSIC", "New", "Vintage", "Emerging"} {
		_, err := NewCandidateTrack(NewCandidateTrackParams{
			ID:       "cand-1",
			Source:   SourceManual,
			Category: CategoryPresent,
			Type:     typ,
		})
		if !errors.Is(err, ErrInvalidType) {
			t.Errorf("type %q: err = %v, want ErrInvalidType", typ, err)
		}
	}
}

func TestNewCandidateTrackCategoryValidationUnaffectedByType(t *testing.T) {
	// A valid Type must not mask or short-circuit Category validation.
	_, err := NewCandidateTrack(NewCandidateTrackParams{
		ID:       "cand-1",
		Source:   SourceManual,
		Category: "Old",
		Type:     TypeCurrent,
	})
	if !errors.Is(err, ErrInvalidCategory) {
		t.Errorf("err = %v, want ErrInvalidCategory", err)
	}
}

func TestNewCandidateTrackTypeAndCategoryAreIndependent(t *testing.T) {
	// Type and Category are two unrelated dimensions: no combination is
	// rejected or rewritten based on the other. Discovery does not imply
	// Emerging, Current does not imply Present, Classic does not imply Past.
	cases := []struct {
		typ      Type
		category Category
	}{
		{TypeDiscovery, CategoryPast},
		{TypeDiscovery, CategoryEmerging},
		{TypeCurrent, CategoryNewRelease},
		{TypeClassic, CategoryPast},
	}
	for _, tc := range cases {
		c, err := NewCandidateTrack(NewCandidateTrackParams{
			ID:       "cand-1",
			Source:   SourceManual,
			Category: tc.category,
			Type:     tc.typ,
		})
		if err != nil {
			t.Fatalf("Type %q + Category %q: NewCandidateTrack returned error: %v", tc.typ, tc.category, err)
		}
		if c.Type != tc.typ {
			t.Errorf("Type = %q, want %q", c.Type, tc.typ)
		}
		if c.Category != tc.category {
			t.Errorf("Category = %q, want %q", c.Category, tc.category)
		}
	}
}
