package discovery

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
	"github.com/vmmatos/sound-continuum-project/internal/spotify"
)

func newTestSvcMetadata(f *fakeCatalogue) *Service {
	return &Service{spotify: f}
}

func candidateFor(t *testing.T, spotifyTrackID string) candidate.CandidateTrack {
	t.Helper()
	c, err := candidate.NewCandidateTrack(candidate.NewCandidateTrackParams{
		ID:             candidate.ID(spotifyTrackID),
		SpotifyTrackID: spotifyTrackID,
		Source:         candidate.SourceSpotify,
		Category:       candidate.CategoryPast,
		Type:           candidate.TypeClassic,
		TrackTitle:     "original title",
		TrackArtist:    "original artist",
	})
	if err != nil {
		t.Fatalf("candidateFor: NewCandidateTrack: %v", err)
	}
	return c
}

func TestEnrichCandidateMetadataCompleteTrack(t *testing.T) {
	f := &fakeCatalogue{trackByID: map[string]spotify.Track{
		"track-1": {
			ID:         "track-1",
			Name:       "Heroes",
			URI:        "spotify:track:track-1",
			DurationMS: 371333,
			Explicit:   false,
			ExternalURLs: spotify.ExternalURLs{
				Spotify: "https://open.spotify.com/track/track-1",
			},
			Artists: []spotify.Artist{
				{ID: "artist-1", Name: "David Bowie", ExternalURLs: spotify.ExternalURLs{Spotify: "https://open.spotify.com/artist/artist-1"}},
			},
			Album: spotify.Album{
				ID:                   "album-1",
				Name:                 "\"Heroes\"",
				AlbumType:            "album",
				ReleaseDate:          "1977-10-14",
				ReleaseDatePrecision: "day",
				ExternalURLs:         spotify.ExternalURLs{Spotify: "https://open.spotify.com/album/album-1"},
				Images: []spotify.Image{
					{URL: "https://example.com/art-640.jpg", Width: 640, Height: 640},
				},
			},
		},
	}}
	svc := newTestSvcMetadata(f)

	result, err := svc.EnrichCandidateMetadata(context.Background(), []candidate.CandidateTrack{candidateFor(t, "track-1")})
	if err != nil {
		t.Fatalf("EnrichCandidateMetadata returned error: %v", err)
	}
	if result.EnrichedCount != 1 || len(result.Failures) != 0 {
		t.Fatalf("result = %+v, want 1 enriched, 0 failures", result)
	}

	m := result.EnrichedCandidates[0].Metadata
	if m == nil {
		t.Fatal("Metadata is nil, want populated")
	}
	if m.Title != "Heroes" {
		t.Errorf("Title = %q, want %q", m.Title, "Heroes")
	}
	if m.DurationMS != 371333 {
		t.Errorf("DurationMS = %d, want 371333", m.DurationMS)
	}
	if m.Explicit != false {
		t.Errorf("Explicit = %v, want false", m.Explicit)
	}
	if m.SpotifyURL != "https://open.spotify.com/track/track-1" {
		t.Errorf("SpotifyURL = %q", m.SpotifyURL)
	}
	if m.SpotifyURI != "spotify:track:track-1" {
		t.Errorf("SpotifyURI = %q", m.SpotifyURI)
	}
	if m.Album.SpotifyAlbumID != "album-1" || m.Album.Name != "\"Heroes\"" || m.Album.AlbumType != "album" {
		t.Errorf("Album = %+v", m.Album)
	}
	if m.Album.ReleaseDate != "1977-10-14" || m.Album.ReleaseDatePrecision != "day" {
		t.Errorf("Album release date = %q/%q, want 1977-10-14/day", m.Album.ReleaseDate, m.Album.ReleaseDatePrecision)
	}
	if m.Album.SpotifyURL != "https://open.spotify.com/album/album-1" {
		t.Errorf("Album.SpotifyURL = %q", m.Album.SpotifyURL)
	}
	if len(m.Artists) != 1 || m.Artists[0].SpotifyArtistID != "artist-1" || m.Artists[0].Name != "David Bowie" {
		t.Errorf("Artists = %+v", m.Artists)
	}
	if len(m.Album.Artwork) != 1 || m.Album.Artwork[0].URL != "https://example.com/art-640.jpg" || m.Album.Artwork[0].Width != 640 {
		t.Errorf("Artwork = %+v", m.Album.Artwork)
	}
}

func TestEnrichCandidateMetadataMultipleArtistsPreserved(t *testing.T) {
	f := &fakeCatalogue{trackByID: map[string]spotify.Track{
		"track-2": {
			ID:   "track-2",
			Name: "Collab Song",
			Artists: []spotify.Artist{
				{ID: "artist-a", Name: "Artist A"},
				{ID: "artist-b", Name: "Artist B"},
				{ID: "artist-c", Name: "Artist C"},
			},
		},
	}}
	svc := newTestSvcMetadata(f)

	result, err := svc.EnrichCandidateMetadata(context.Background(), []candidate.CandidateTrack{candidateFor(t, "track-2")})
	if err != nil {
		t.Fatalf("EnrichCandidateMetadata returned error: %v", err)
	}

	artists := result.EnrichedCandidates[0].Metadata.Artists
	if len(artists) != 3 {
		t.Fatalf("Artists = %+v, want 3", artists)
	}
	wantIDs := []string{"artist-a", "artist-b", "artist-c"}
	wantNames := []string{"Artist A", "Artist B", "Artist C"}
	for i, a := range artists {
		if a.SpotifyArtistID != wantIDs[i] || a.Name != wantNames[i] {
			t.Errorf("artist[%d] = %+v, want ID=%q Name=%q", i, a, wantIDs[i], wantNames[i])
		}
	}
}

func TestEnrichCandidateMetadataAlbumArtworkMapped(t *testing.T) {
	f := &fakeCatalogue{trackByID: map[string]spotify.Track{
		"track-3": {
			ID:   "track-3",
			Name: "Song",
			Album: spotify.Album{
				Images: []spotify.Image{
					{URL: "https://example.com/640.jpg", Width: 640, Height: 640},
					{URL: "https://example.com/300.jpg", Width: 300, Height: 300},
					{URL: "https://example.com/64.jpg", Width: 64, Height: 64},
				},
			},
		},
	}}
	svc := newTestSvcMetadata(f)

	result, err := svc.EnrichCandidateMetadata(context.Background(), []candidate.CandidateTrack{candidateFor(t, "track-3")})
	if err != nil {
		t.Fatalf("EnrichCandidateMetadata returned error: %v", err)
	}

	artwork := result.EnrichedCandidates[0].Metadata.Album.Artwork
	if len(artwork) != 3 {
		t.Fatalf("Artwork = %+v, want 3 images", artwork)
	}
	if artwork[0].Width != 640 || artwork[1].Width != 300 || artwork[2].Width != 64 {
		t.Errorf("Artwork dimensions not preserved in order: %+v", artwork)
	}
}

func TestEnrichCandidateMetadataReleaseDatePrecision(t *testing.T) {
	cases := []struct {
		name      string
		date      string
		precision string
	}{
		{"year only", "1981", "year"},
		{"year and month", "1981-12", "month"},
		{"full date", "1981-12-15", "day"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeCatalogue{trackByID: map[string]spotify.Track{
				"track-x": {ID: "track-x", Name: "Song", Album: spotify.Album{
					ReleaseDate:          tc.date,
					ReleaseDatePrecision: tc.precision,
				}},
			}}
			svc := newTestSvcMetadata(f)

			result, err := svc.EnrichCandidateMetadata(context.Background(), []candidate.CandidateTrack{candidateFor(t, "track-x")})
			if err != nil {
				t.Fatalf("EnrichCandidateMetadata returned error: %v", err)
			}
			album := result.EnrichedCandidates[0].Metadata.Album
			if album.ReleaseDate != tc.date || album.ReleaseDatePrecision != tc.precision {
				t.Errorf("Album release date = %q/%q, want %q/%q", album.ReleaseDate, album.ReleaseDatePrecision, tc.date, tc.precision)
			}
		})
	}
}

func TestEnrichCandidateMetadataExplicitTrue(t *testing.T) {
	f := &fakeCatalogue{trackByID: map[string]spotify.Track{
		"track-e": {ID: "track-e", Name: "Song", Explicit: true},
	}}
	svc := newTestSvcMetadata(f)

	result, err := svc.EnrichCandidateMetadata(context.Background(), []candidate.CandidateTrack{candidateFor(t, "track-e")})
	if err != nil {
		t.Fatalf("EnrichCandidateMetadata returned error: %v", err)
	}
	if !result.EnrichedCandidates[0].Metadata.Explicit {
		t.Error("Explicit = false, want true")
	}
}

func TestEnrichCandidateMetadataExplicitFalse(t *testing.T) {
	f := &fakeCatalogue{trackByID: map[string]spotify.Track{
		"track-ne": {ID: "track-ne", Name: "Song", Explicit: false},
	}}
	svc := newTestSvcMetadata(f)

	result, err := svc.EnrichCandidateMetadata(context.Background(), []candidate.CandidateTrack{candidateFor(t, "track-ne")})
	if err != nil {
		t.Fatalf("EnrichCandidateMetadata returned error: %v", err)
	}
	if result.EnrichedCandidates[0].Metadata.Explicit {
		t.Error("Explicit = true, want false")
	}
}

func TestEnrichCandidateMetadataSkipsCandidateWithNoSpotifyID(t *testing.T) {
	f := &fakeCatalogue{}
	svc := newTestSvcMetadata(f)

	// A candidate with no Spotify track ID (e.g. a future Manual source) —
	// built directly, bypassing NewCandidateTrack's Spotify-source
	// requirement, since no non-Spotify Source value exists yet.
	c := candidate.CandidateTrack{
		ID:       "manual-1",
		Source:   candidate.SourceSpotify,
		Category: candidate.CategoryPast,
		Type:     candidate.TypeClassic,
		Status:   candidate.StatusDiscovered,
		// SpotifyTrackID intentionally empty.
	}

	result, err := svc.EnrichCandidateMetadata(context.Background(), []candidate.CandidateTrack{c})
	if err != nil {
		t.Fatalf("EnrichCandidateMetadata returned error: %v", err)
	}
	if result.SkippedCount != 1 || result.EnrichedCount != 0 {
		t.Fatalf("result = %+v, want 1 skipped, 0 enriched", result)
	}
	if result.EnrichedCandidates[0].Metadata != nil {
		t.Error("Metadata should stay nil for a skipped candidate")
	}
	if len(f.trackCalls) != 0 {
		t.Error("no Spotify Track call should be attempted for a candidate with no Spotify ID")
	}
}

func TestEnrichCandidateMetadataPreservesClassification(t *testing.T) {
	f := &fakeCatalogue{trackByID: map[string]spotify.Track{
		"track-1": {ID: "track-1", Name: "Song"},
	}}
	svc := newTestSvcMetadata(f)
	c := candidateFor(t, "track-1")

	result, err := svc.EnrichCandidateMetadata(context.Background(), []candidate.CandidateTrack{c})
	if err != nil {
		t.Fatalf("EnrichCandidateMetadata returned error: %v", err)
	}
	enriched := result.EnrichedCandidates[0]
	if enriched.Type != c.Type || enriched.Category != c.Category || enriched.Status != c.Status {
		t.Errorf("classification changed: got Type=%q Category=%q Status=%q, want Type=%q Category=%q Status=%q",
			enriched.Type, enriched.Category, enriched.Status, c.Type, c.Category, c.Status)
	}
}

func TestEnrichCandidateMetadataPreservesProvenance(t *testing.T) {
	f := &fakeCatalogue{trackByID: map[string]spotify.Track{
		"track-1": {ID: "track-1", Name: "Song"},
	}}
	svc := newTestSvcMetadata(f)
	c := candidateFor(t, "track-1")
	c.Provenance = []candidate.DiscoveryProvenance{{
		Method:   candidate.DiscoveryMethodClassicReferenceArtist,
		Provider: candidate.ProvenanceProviderSpotify,
		Seed:     &candidate.SeedArtist{Provider: candidate.ProvenanceProviderSpotify, ProviderArtistID: "a-1", Name: "original artist"},
	}}

	result, err := svc.EnrichCandidateMetadata(context.Background(), []candidate.CandidateTrack{c})
	if err != nil {
		t.Fatalf("EnrichCandidateMetadata returned error: %v", err)
	}
	enriched := result.EnrichedCandidates[0]
	if !reflect.DeepEqual(enriched.Provenance, c.Provenance) {
		t.Errorf("EnrichCandidateMetadata did not preserve Provenance: got %+v, want %+v", enriched.Provenance, c.Provenance)
	}
}

func TestEnrichCandidateMetadataPreservesIdentity(t *testing.T) {
	f := &fakeCatalogue{trackByID: map[string]spotify.Track{
		"track-1": {ID: "track-1", Name: "Song"},
	}}
	svc := newTestSvcMetadata(f)
	c := candidateFor(t, "track-1")

	result, err := svc.EnrichCandidateMetadata(context.Background(), []candidate.CandidateTrack{c})
	if err != nil {
		t.Fatalf("EnrichCandidateMetadata returned error: %v", err)
	}
	enriched := result.EnrichedCandidates[0]
	if enriched.ID != c.ID || enriched.SpotifyTrackID != c.SpotifyTrackID {
		t.Errorf("identity changed: got ID=%q SpotifyTrackID=%q, want ID=%q SpotifyTrackID=%q",
			enriched.ID, enriched.SpotifyTrackID, c.ID, c.SpotifyTrackID)
	}
}

func TestEnrichCandidateMetadataConnectionFailureAborts(t *testing.T) {
	f := &fakeCatalogue{trackErr: map[string]error{"track-1": spotify.ErrNotConnected}}
	svc := newTestSvcMetadata(f)

	_, err := svc.EnrichCandidateMetadata(context.Background(), []candidate.CandidateTrack{candidateFor(t, "track-1")})
	if !errors.Is(err, spotify.ErrNotConnected) {
		t.Errorf("err = %v, want ErrNotConnected", err)
	}
}

func TestEnrichCandidateMetadataPerCandidateFailureContinues(t *testing.T) {
	f := &fakeCatalogue{
		trackErr:  map[string]error{"track-missing": spotify.ErrNotFound},
		trackByID: map[string]spotify.Track{"track-ok": {ID: "track-ok", Name: "Song"}},
	}
	svc := newTestSvcMetadata(f)

	result, err := svc.EnrichCandidateMetadata(context.Background(), []candidate.CandidateTrack{
		candidateFor(t, "track-missing"),
		candidateFor(t, "track-ok"),
	})
	if err != nil {
		t.Fatalf("EnrichCandidateMetadata returned error: %v", err)
	}
	if len(result.EnrichedCandidates) != 2 {
		t.Fatalf("EnrichedCandidates = %+v, want both candidates kept", result.EnrichedCandidates)
	}
	if result.EnrichedCandidates[0].Metadata != nil {
		t.Error("failed candidate's Metadata should stay nil")
	}
	if result.EnrichedCandidates[1].Metadata == nil {
		t.Error("surviving candidate should be enriched")
	}
	if len(result.Failures) != 1 || result.Failures[0].SpotifyTrackID != "track-missing" || result.Failures[0].Reason != "not_found" {
		t.Errorf("Failures = %+v, want one not_found failure for track-missing", result.Failures)
	}
	if result.EnrichedCount != 1 {
		t.Errorf("EnrichedCount = %d, want 1", result.EnrichedCount)
	}
}

func TestEnrichCandidateMetadataMissingOptionalFieldsStaySafe(t *testing.T) {
	f := &fakeCatalogue{trackByID: map[string]spotify.Track{
		// Minimal track: no artists, no album images, no external URLs.
		"track-bare": {ID: "track-bare", Name: "Bare Song"},
	}}
	svc := newTestSvcMetadata(f)

	result, err := svc.EnrichCandidateMetadata(context.Background(), []candidate.CandidateTrack{candidateFor(t, "track-bare")})
	if err != nil {
		t.Fatalf("EnrichCandidateMetadata returned error: %v", err)
	}
	m := result.EnrichedCandidates[0].Metadata
	if m == nil {
		t.Fatal("Metadata is nil, want a safe empty-ish result")
	}
	if len(m.Artists) != 0 {
		t.Errorf("Artists = %+v, want empty", m.Artists)
	}
	if len(m.Album.Artwork) != 0 {
		t.Errorf("Artwork = %+v, want empty", m.Album.Artwork)
	}
	if m.SpotifyURL != "" || m.Album.SpotifyURL != "" {
		t.Errorf("expected empty URLs, got track=%q album=%q", m.SpotifyURL, m.Album.SpotifyURL)
	}
}
