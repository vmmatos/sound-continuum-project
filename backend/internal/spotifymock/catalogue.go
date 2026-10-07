// Package spotifymock is a deterministic, in-memory stand-in for Spotify,
// used only when SPOTIFY_MOCK_MODE=true (Card #56, cmd/server/main.go).
// It exists because Spotify Development Mode is rate-limited (documented
// since Card #36, still in effect as of Card #53/#126), blocking local
// Candidate Review development.
//
// Catalogue implements the same method set as
// backend/internal/discovery's unexported spotifyCatalogue interface
// (Search/ArtistAlbums/AlbumTracks/PlaylistItems/OfficialPlaylist/Track) —
// confirmed to cover every Spotify touchpoint anywhere in the discovery
// pipeline (DiscoverClassic/DiscoverCurrent/DiscoverEmerging/
// FilterRecentTracks/EnrichCandidateMetadata all funnel through exactly
// these six methods, no more). Swapping Catalogue in at that one seam
// (discovery.NewService's first parameter, widened in Card #56 — see
// decisions.md) mocks the entire Discover -> Pool -> Score -> Rank ->
// Review path with no `if mock` checks scattered through discovery itself.
//
// Every method here is a pure function of its input — no shared mutable
// state, no cache, no clock, no randomness, and critically, no net/http
// client anywhere in this package. Same input always produces the same
// output, forever, which is what makes the dataset deterministic across
// runs as the card requires.
package spotifymock

import (
	"context"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"

	"github.com/vmmatos/sound-continuum-project/internal/spotify"
)

// Catalogue is a stateless, deterministic Spotify replacement. The zero
// value is ready to use.
type Catalogue struct{}

// NewCatalogue returns a ready-to-use mock catalogue.
func NewCatalogue() *Catalogue {
	return &Catalogue{}
}

// mockOfficialPlaylistID is the fixed ID Catalogue reports for the one
// official Sound Continuum playlist. Clearly a dummy value ("mock-"
// prefix), per the card's requirement.
const mockOfficialPlaylistID = "mock-playlist-001"

// hashSeed deterministically derives a uint32 from one or more strings —
// the one primitive every other generator in this file builds on. Same
// input, same output, every call, forever: no time, no randomness.
func hashSeed(parts ...string) uint32 {
	h := fnv.New32a()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0}) // separator, so ("ab","c") != ("a","bc")
	}
	return h.Sum32()
}

// mockID builds a clearly-identifiable dummy Spotify-style ID
// (e.g. "mock-artist-a1b2c3d4") — prefixed per the card's "use clearly
// identifiable dummy/test values" requirement, deterministic from seed.
func mockID(kind string, seed ...string) string {
	return fmt.Sprintf("mock-%s-%08x", kind, hashSeed(seed...))
}

// mockTitle builds a deterministic, clearly-dummy title from seed — not an
// attempt at realistic song-naming, just enough variation (via seed) to
// tell candidates apart in the UI.
func mockTitle(seed string) string {
	return fmt.Sprintf("Mock %08x", hashSeed(seed))
}

// Search synthesizes exactly one artist whose Name echoes query exactly.
// This is required, not cosmetic: discovery.resolveArtist only accepts an
// exact, case-insensitive name match, so a fixed-roster mock (distinct
// names from the query) would silently fail every resolution. Echoing the
// query back means every one of Sound Continuum's canonical reference
// artists — and any other artist name a caller searches for — resolves
// deterministically, with no per-name data to hand-author.
func (c *Catalogue) Search(ctx context.Context, query, types string, limit, offset int) (spotify.SearchResult, error) {
	name := strings.TrimSpace(query)
	if name == "" {
		return spotify.SearchResult{}, nil
	}
	artistID := mockID("artist", strings.ToLower(name))
	artist := spotify.Artist{
		ID:   artistID,
		Name: name,
		URI:  "spotify:artist:" + artistID,
		Type: "artist",
		ExternalURLs: spotify.ExternalURLs{
			Spotify: "https://open.spotify.com/artist/" + artistID,
		},
	}
	return spotify.SearchResult{
		Artists: &spotify.Paging[spotify.Artist]{
			Items: []spotify.Artist{artist},
			Total: 1,
			Limit: limit,
		},
	}, nil
}

// mockAlbumCount and mockTrackCount bound how many albums/tracks each
// generator produces per artist/album — small, fixed, enough to exercise
// Candidate Review without an unbounded dataset.
const (
	mockAlbumCount = 2
	mockTrackCount = 4
)

// mockReleaseDate reports a date 7 days before now, formatted
// "YYYY-MM-DD" — always inside DiscoverCurrent/DiscoverEmerging's 90-day
// recency window (see discovery.DefaultCurrentConfig/DefaultEmergingConfig)
// no matter when mock mode is run, so Current and Emerging candidates
// aren't silently filtered out as "outside the window." This is the only
// non-pure-hash value in this package — a relative "recently released"
// release date is inherent to any recency filter, real Spotify data
// included, not a determinism violation: every other field (IDs, titles,
// durations) is still a pure function of its input, stable across runs on
// any given day.
func mockReleaseDate() string {
	return time.Now().AddDate(0, 0, -7).Format("2006-01-02")
}

// mockAlbum deterministically builds one album for artistID, indexed by i.
func mockAlbum(artistID string, i int) spotify.Album {
	albumID := mockID("album", artistID, strconv.Itoa(i))
	name := mockTitle(albumID) + " (Mock Album)"
	return spotify.Album{
		ID:                   albumID,
		Name:                 name,
		AlbumType:            "album",
		TotalTracks:          mockTrackCount,
		ReleaseDate:          mockReleaseDate(),
		ReleaseDatePrecision: "day",
		URI:                  "spotify:album:" + albumID,
		ExternalURLs:         spotify.ExternalURLs{Spotify: "https://open.spotify.com/album/" + albumID},
		Images: []spotify.Image{
			{URL: "https://i.scdn.co/image/" + mockID("artwork", albumID), Height: 300, Width: 300},
		},
	}
}

// ArtistAlbums deterministically derives mockAlbumCount albums from
// artistID — same artistID always yields the same albums.
func (c *Catalogue) ArtistAlbums(ctx context.Context, artistID string, limit, offset int) (spotify.Paging[spotify.Album], error) {
	var albums []spotify.Album
	for i := offset; i < mockAlbumCount && len(albums) < limit; i++ {
		albums = append(albums, mockAlbum(artistID, i))
	}
	return spotify.Paging[spotify.Album]{Items: albums, Total: mockAlbumCount, Limit: limit, Offset: offset}, nil
}

// mockTrack deterministically builds one track for albumID, indexed by i —
// the single source of truth both AlbumTracks and Track(id) build on, so a
// track looked up directly by ID always matches what AlbumTracks already
// produced for it.
func mockTrack(albumID string, i int) spotify.Track {
	trackID := mockID("track", albumID, strconv.Itoa(i))
	return trackFromID(trackID)
}

// trackFromID reconstructs a full mock track purely from its own ID — no
// shared state with mockTrack/AlbumTracks, by construction, since every
// field is derived from hashing trackID itself. This is what guarantees
// Track(id) is always consistent with whatever AlbumTracks minted that ID
// from in the first place.
func trackFromID(trackID string) spotify.Track {
	title := mockTitle(trackID)
	artistID := mockID("artist-of-track", trackID)
	albumID := mockID("album-of-track", trackID)
	durationMS := 120000 + int(hashSeed(trackID)%180000) // 2:00–5:00, deterministic

	return spotify.Track{
		ID:         trackID,
		Name:       title,
		URI:        "spotify:track:" + trackID,
		Type:       "track",
		DurationMS: durationMS,
		Explicit:   hashSeed(trackID, "explicit")%5 == 0,
		Artists: []spotify.Artist{{
			ID:           artistID,
			Name:         "Mock Artist " + trackID[len(trackID)-4:],
			URI:          "spotify:artist:" + artistID,
			Type:         "artist",
			ExternalURLs: spotify.ExternalURLs{Spotify: "https://open.spotify.com/artist/" + artistID},
		}},
		Album: spotify.Album{
			ID:                   albumID,
			Name:                 mockTitle(albumID) + " (Mock Album)",
			AlbumType:            "album",
			ReleaseDate:          mockReleaseDate(),
			ReleaseDatePrecision: "day",
			URI:                  "spotify:album:" + albumID,
			ExternalURLs:         spotify.ExternalURLs{Spotify: "https://open.spotify.com/album/" + albumID},
			Images: []spotify.Image{
				{URL: "https://i.scdn.co/image/" + mockID("artwork", albumID), Height: 300, Width: 300},
			},
		},
		ExternalURLs: spotify.ExternalURLs{Spotify: "https://open.spotify.com/track/" + trackID},
	}
}

// AlbumTracks deterministically derives mockTrackCount tracks from
// albumID — same albumID always yields the same tracks.
func (c *Catalogue) AlbumTracks(ctx context.Context, albumID string, limit, offset int) (spotify.Paging[spotify.Track], error) {
	var tracks []spotify.Track
	for i := offset; i < mockTrackCount && len(tracks) < limit; i++ {
		tracks = append(tracks, mockTrack(albumID, i))
	}
	return spotify.Paging[spotify.Track]{Items: tracks, Total: mockTrackCount, Limit: limit, Offset: offset}, nil
}

// Track reconstructs the same track EnrichCandidateMetadata already saw via
// AlbumTracks, purely from trackID — see trackFromID.
func (c *Catalogue) Track(ctx context.Context, trackID string) (spotify.Track, error) {
	return trackFromID(trackID), nil
}

// OfficialPlaylist reports a fixed, clearly-dummy official playlist — mock
// mode never requires the curator to have initialized a real one.
func (c *Catalogue) OfficialPlaylist(ctx context.Context) (*spotify.OfficialPlaylist, error) {
	return &spotify.OfficialPlaylist{
		SpotifyPlaylistID: mockOfficialPlaylistID,
		Name:              "Mock Sound Continuum Playlist",
		URL:               "https://open.spotify.com/playlist/" + mockOfficialPlaylistID,
	}, nil
}

// PlaylistItems always reports an empty playlist: every mock candidate is
// therefore "new" (Freshness = 1.0, RepetitionPenalty = 0) — the simplest
// correct starting point, and exactly the state the card's own reference
// screenshot shows. Mocking a populated playlist history is not required
// to exercise Keep or the rest of Candidate Review.
func (c *Catalogue) PlaylistItems(ctx context.Context, playlistID string, limit, offset int) (spotify.Paging[spotify.PlaylistItem], error) {
	return spotify.Paging[spotify.PlaylistItem]{Limit: limit, Offset: offset}, nil
}
