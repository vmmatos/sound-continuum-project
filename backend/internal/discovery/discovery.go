// Package discovery builds candidate track pools from Sound Continuum's
// editorial reference data via the Spotify catalogue. It is a discovery
// workflow, not a ranking engine — see DiscoverClassic.
package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
	"github.com/vmmatos/sound-continuum-project/internal/spotify"
)

// Config bounds one DiscoverClassic run. Constructor-injected rather than
// env-configured — this repo has no env-var-driven-limit convention, and
// tests need small numbers.
type Config struct {
	MaxAlbumsPerArtist int
	MaxTracksPerAlbum  int
	MaxTotalCandidates int
}

// DefaultConfig bounds a run at 5 albums/artist, 10 tracks/album, and 150
// candidates total. The total cap is the real limiter — it binds well
// before the theoretical 15*5*10=750 raw ceiling across all reference
// artists — keeping the operation predictable.
func DefaultConfig() Config {
	return Config{MaxAlbumsPerArtist: 5, MaxTracksPerAlbum: 10, MaxTotalCandidates: 150}
}

// discoveryReason is recorded on every candidate this workflow produces —
// there is one discovery workflow, so one fixed reason.
const discoveryReason = "Discovered from Past reference artist catalogue."

// errArtistNotResolved is resolveArtist's internal "no exact match"
// signal. It routes the artist into Result.UnresolvedArtists and is never
// returned as DiscoverClassic's own error.
var errArtistNotResolved = errors.New("discovery: artist not found by exact name match")

// spotifyCatalogue is the subset of spotify.Service DiscoverClassic needs.
// *spotify.Service satisfies it, so production wiring is a plain
// spotify.Service — the interface exists only so tests can substitute a
// fake without going through Spotify's real OAuth/HTTP transport, which
// spotify.Service has no exported hook to redirect from outside its
// package.
type spotifyCatalogue interface {
	Search(ctx context.Context, query, types string, limit, offset int) (spotify.SearchResult, error)
	ArtistAlbums(ctx context.Context, artistID string, limit, offset int) (spotify.Paging[spotify.Album], error)
	AlbumTracks(ctx context.Context, albumID string, limit, offset int) (spotify.Paging[spotify.Track], error)
}

// Service builds candidate pools from Sound Continuum's reference artist
// data via a Spotify catalogue.
type Service struct {
	spotify spotifyCatalogue
	cfg     Config
}

// NewService wires a discovery Service to an existing spotify.Service —
// no second Spotify client is created.
func NewService(spotifyService *spotify.Service, cfg Config) *Service {
	return &Service{spotify: spotifyService, cfg: cfg}
}

// Failure records one artist/album/track-level failure that DiscoverClassic
// recorded and continued past, rather than aborting the run.
type Failure struct {
	Artist string
	Stage  string // "resolve", "albums", or "tracks"
	Err    string
}

// Result reports what one DiscoverClassic run did. It is an operational
// discovery result, not analytics.
type Result struct {
	Candidates        []candidate.CandidateTrack
	UnresolvedArtists []string
	Failures          []Failure
	ArtistsInspected  int
	AlbumsInspected   int
	TracksInspected   int
	DuplicatesSkipped int
}

// DiscoverClassic explores each PastReferenceArtists' Spotify catalogue
// (artist albums -> album tracks) and returns Classic/Past candidates,
// deduplicated by Spotify track ID. Popularity is never used to select or
// order candidates — Spotify's own catalogue order is left as returned.
//
// Aborts the whole run only on a Spotify connection failure
// (ErrNotConnected / ErrInvalidGrant); every other failure (an unresolved
// artist, an album or track fetch error) is recorded on Result and the
// run continues. Stops early, across all remaining artists, once
// Config.MaxTotalCandidates is reached — the operation is bounded by
// construction, not by exhausting the reference list.
func (s *Service) DiscoverClassic(ctx context.Context) (Result, error) {
	var result Result
	seen := make(map[string]bool)

artists:
	for _, name := range PastReferenceArtists {
		if len(result.Candidates) >= s.cfg.MaxTotalCandidates {
			break
		}
		result.ArtistsInspected++

		artistID, err := s.resolveArtist(ctx, name)
		switch {
		case errors.Is(err, errArtistNotResolved):
			result.UnresolvedArtists = append(result.UnresolvedArtists, name)
			continue
		case isConnectionError(err):
			return result, err
		case err != nil:
			result.Failures = append(result.Failures, Failure{Artist: name, Stage: "resolve", Err: err.Error()})
			continue
		}

		albums, err := walkPages(s.cfg.MaxAlbumsPerArtist, func(limit, offset int) (spotify.Paging[spotify.Album], error) {
			return s.spotify.ArtistAlbums(ctx, artistID, limit, offset)
		})
		if isConnectionError(err) {
			return result, err
		}
		if err != nil {
			result.Failures = append(result.Failures, Failure{Artist: name, Stage: "albums", Err: err.Error()})
			continue
		}
		result.AlbumsInspected += len(albums)

		for _, album := range albums {
			if len(result.Candidates) >= s.cfg.MaxTotalCandidates {
				break artists
			}

			tracks, err := walkPages(s.cfg.MaxTracksPerAlbum, func(limit, offset int) (spotify.Paging[spotify.Track], error) {
				return s.spotify.AlbumTracks(ctx, album.ID, limit, offset)
			})
			if isConnectionError(err) {
				return result, err
			}
			if err != nil {
				result.Failures = append(result.Failures, Failure{Artist: name, Stage: "tracks", Err: err.Error()})
				continue
			}
			result.TracksInspected += len(tracks)

			for _, track := range tracks {
				if len(result.Candidates) >= s.cfg.MaxTotalCandidates {
					break artists
				}
				if track.ID == "" {
					continue
				}
				if seen[track.ID] {
					result.DuplicatesSkipped++
					continue
				}
				seen[track.ID] = true

				c, err := candidate.NewCandidateTrack(candidate.NewCandidateTrackParams{
					ID:              candidate.ID(track.ID),
					SpotifyTrackID:  track.ID,
					Source:          candidate.SourceSpotify,
					Category:        candidate.CategoryPast,
					Type:            candidate.TypeClassic,
					TrackTitle:      track.Name,
					TrackArtist:     name,
					DiscoveryReason: discoveryReason,
				})
				if err != nil {
					// Fabricated/invalid metadata never becomes a
					// candidate rather than crashing the whole run.
					continue
				}
				result.Candidates = append(result.Candidates, c)
			}
		}
	}

	return result, nil
}

// resolveArtist looks up name via Search(type=artist) and returns the
// Spotify ID of an exact, case-insensitive, trimmed name match among the
// top 5 results. No fuzzy matching — an unresolved artist is surfaced,
// never guessed.
func (s *Service) resolveArtist(ctx context.Context, name string) (string, error) {
	result, err := s.spotify.Search(ctx, name, "artist", 5, 0)
	if err != nil {
		return "", err
	}
	if result.Artists == nil {
		return "", errArtistNotResolved
	}
	want := strings.ToLower(strings.TrimSpace(name))
	for _, a := range result.Artists.Items {
		if strings.ToLower(strings.TrimSpace(a.Name)) == want {
			return a.ID, nil
		}
	}
	return "", errArtistNotResolved
}

// walkPages bounded-walks a Spotify paginated resource by offset,
// accumulating up to maxItems items, stopping early once Spotify returns
// fewer items than requested (no more pages). It never follows
// Paging.Next — every existing spotify.Client method re-requests by
// offset, not by URL, and this matches that. maxItems <= 0 returns
// immediately with no request made.
func walkPages[T any](maxItems int, fetch func(limit, offset int) (spotify.Paging[T], error)) ([]T, error) {
	if maxItems <= 0 {
		return nil, nil
	}

	var items []T
	offset := 0
	for len(items) < maxItems {
		pageSize := maxItems - len(items)
		if pageSize > 50 {
			pageSize = 50
		}

		page, err := fetch(pageSize, offset)
		if err != nil {
			return nil, err
		}
		items = append(items, page.Items...)
		offset += len(page.Items)

		if len(page.Items) < pageSize {
			break
		}
	}
	if len(items) > maxItems {
		items = items[:maxItems]
	}
	return items, nil
}

// isConnectionError reports whether err means Spotify authentication is
// unavailable — the only failure that aborts an entire DiscoverClassic run.
func isConnectionError(err error) bool {
	return errors.Is(err, spotify.ErrNotConnected) || errors.Is(err, spotify.ErrInvalidGrant)
}

// ClassicHandler exposes POST /api/discovery/classic. Runs DiscoverClassic
// once per call — no persistence, no request body, no query parameters.
// It never modifies the official Spotify playlist and never selects a
// candidate.
func (s *Service) ClassicHandler(w http.ResponseWriter, r *http.Request) {
	result, err := s.DiscoverClassic(r.Context())
	if err != nil {
		log.Printf("classic discovery failed: %v", err)
		writeDiscoveryError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// writeDiscoveryError maps DiscoverClassic's top-level error to an HTTP
// status. DiscoverClassic only ever returns a connection-level error here
// (every other failure is recorded on Result instead), so this is a small,
// local equivalent of spotify's own writeSpotifyError rather than a shared
// export.
func writeDiscoveryError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, spotify.ErrNotConnected):
		http.Error(w, "Spotify is not connected", http.StatusServiceUnavailable)
	case errors.Is(err, spotify.ErrInvalidGrant):
		http.Error(w, "Spotify authorization required", http.StatusUnauthorized)
	default:
		http.Error(w, "classic discovery failed", http.StatusBadGateway)
	}
}
