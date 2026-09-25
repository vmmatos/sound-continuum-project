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
