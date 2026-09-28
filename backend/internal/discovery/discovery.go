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
	"sort"
	"strings"
	"time"

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

// discoveryReason is recorded on every candidate DiscoverClassic produces.
const discoveryReason = "Discovered from Past reference artist catalogue."

// currentDiscoveryReason is recorded on every candidate DiscoverCurrent
// produces.
const currentDiscoveryReason = "Discovered from Present reference artist recent catalogue."

// CurrentConfig bounds one DiscoverCurrent run. Kept separate from Config
// rather than merged into one shared struct — DiscoverCurrent has a
// filter-by-recency-then-sort step DiscoverClassic doesn't, so a shared
// struct would carry fields each workflow ignores.
type CurrentConfig struct {
	// LookbackDays is the recent-catalogue window: a release is only
	// considered "current" if its release date falls within this many
	// days of now. See parseReleaseDate for how partial release dates
	// are resolved against this window.
	LookbackDays int
	// MaxAlbumsScannedPerArtist bounds the raw albums/singles fetched per
	// artist before date filtering — independent of how many of them end
	// up recent enough to keep.
	MaxAlbumsScannedPerArtist int
	// MaxAlbumsPerArtist bounds how many of the most recent qualifying
	// releases (after filtering and sorting) are inspected for tracks.
	MaxAlbumsPerArtist int
	MaxTracksPerAlbum  int
	MaxTotalCandidates int
}

// DefaultCurrentConfig bounds a DiscoverCurrent run at a 90-day recent
// catalogue window, 50 raw releases scanned per artist (Spotify's max page
// size — one request), the 5 most recent qualifying releases inspected per
// artist, 10 tracks per release, and 150 candidates total.
//
// 90 days (~one quarter) comfortably covers a contemporary artist's latest
// single/EP/album cycle without reaching into last year's material. This is
// a discovery-window default, not an editorial rule — the boundary between
// Present, Emerging and New Release remains a human decision made later,
// not encoded here.
func DefaultCurrentConfig() CurrentConfig {
	return CurrentConfig{
		LookbackDays:              90,
		MaxAlbumsScannedPerArtist: 50,
		MaxAlbumsPerArtist:        5,
		MaxTracksPerAlbum:         10,
		MaxTotalCandidates:        150,
	}
}

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
	spotify    spotifyCatalogue
	classicCfg Config
	currentCfg CurrentConfig
}

// NewService wires a discovery Service to an existing spotify.Service —
// no second Spotify client is created.
func NewService(spotifyService *spotify.Service, classicCfg Config, currentCfg CurrentConfig) *Service {
	return &Service{spotify: spotifyService, classicCfg: classicCfg, currentCfg: currentCfg}
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
	// ReleasesOutsideWindow counts releases scanned but excluded by
	// CurrentConfig.LookbackDays. Always 0 for DiscoverClassic, which has
	// no recency window.
	ReleasesOutsideWindow int
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
		if len(result.Candidates) >= s.classicCfg.MaxTotalCandidates {
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

		albums, err := walkPages(s.classicCfg.MaxAlbumsPerArtist, maxArtistAlbumsPageSize, func(limit, offset int) (spotify.Paging[spotify.Album], error) {
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
			if len(result.Candidates) >= s.classicCfg.MaxTotalCandidates {
				break artists
			}

			tracks, err := walkPages(s.classicCfg.MaxTracksPerAlbum, maxAlbumTracksPageSize, func(limit, offset int) (spotify.Paging[spotify.Track], error) {
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
				if len(result.Candidates) >= s.classicCfg.MaxTotalCandidates {
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

// DiscoverCurrent explores each PresentReferenceArtists' recent Spotify
// catalogue (artist albums/singles released within CurrentConfig.LookbackDays
// -> album tracks) and returns Current/Present candidates, deduplicated by
// Spotify track ID. Popularity is never used to select or order
// candidates; within the recency window, releases are ordered newest-first
// (release date descending, Spotify album ID as a stable tiebreaker) before
// being truncated to CurrentConfig.MaxAlbumsPerArtist — Spotify's own
// artist-albums ordering is not documented to be chronological, so this
// ordering is explicit rather than assumed.
//
// Aborts the whole run only on a Spotify connection failure
// (ErrNotConnected / ErrInvalidGrant); every other failure (an unresolved
// artist, an album or track fetch error) is recorded on Result and the run
// continues. Stops early, across all remaining artists, once
// CurrentConfig.MaxTotalCandidates is reached.
func (s *Service) DiscoverCurrent(ctx context.Context) (Result, error) {
	var result Result
	seen := make(map[string]bool)
	cutoff := time.Now().AddDate(0, 0, -s.currentCfg.LookbackDays)

artists:
	for _, name := range PresentReferenceArtists {
		if len(result.Candidates) >= s.currentCfg.MaxTotalCandidates {
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

		scanned, err := walkPages(s.currentCfg.MaxAlbumsScannedPerArtist, maxArtistAlbumsPageSize, func(limit, offset int) (spotify.Paging[spotify.Album], error) {
			return s.spotify.ArtistAlbums(ctx, artistID, limit, offset)
		})
		if isConnectionError(err) {
			return result, err
		}
		if err != nil {
			result.Failures = append(result.Failures, Failure{Artist: name, Stage: "albums", Err: err.Error()})
			continue
		}

		var recent []spotify.Album
		for _, album := range scanned {
			released, ok := parseReleaseDate(album.ReleaseDate, album.ReleaseDatePrecision)
			if !ok || released.Before(cutoff) {
				result.ReleasesOutsideWindow++
				continue
			}
			recent = append(recent, album)
		}
		sort.Slice(recent, func(i, j int) bool {
			di, _ := parseReleaseDate(recent[i].ReleaseDate, recent[i].ReleaseDatePrecision)
			dj, _ := parseReleaseDate(recent[j].ReleaseDate, recent[j].ReleaseDatePrecision)
			if !di.Equal(dj) {
				return di.After(dj)
			}
			return recent[i].ID < recent[j].ID
		})
		if len(recent) > s.currentCfg.MaxAlbumsPerArtist {
			recent = recent[:s.currentCfg.MaxAlbumsPerArtist]
		}
		result.AlbumsInspected += len(recent)

		for _, album := range recent {
			if len(result.Candidates) >= s.currentCfg.MaxTotalCandidates {
				break artists
			}

			tracks, err := walkPages(s.currentCfg.MaxTracksPerAlbum, maxAlbumTracksPageSize, func(limit, offset int) (spotify.Paging[spotify.Track], error) {
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
				if len(result.Candidates) >= s.currentCfg.MaxTotalCandidates {
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
					Category:        candidate.CategoryPresent,
					Type:            candidate.TypeCurrent,
					TrackTitle:      track.Name,
					TrackArtist:     name,
					DiscoveryReason: currentDiscoveryReason,
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

// parseReleaseDate interprets a Spotify release_date according to its
// release_date_precision ("day", "month", or "year" — Spotify does not
// always know the exact release day). A partial date resolves to the
// EARLIEST instant consistent with its precision (the 1st of the month/
// year): this is the conservative reading for a recency filter, since it
// never treats a release as more recent than Spotify actually reported —
// an ambiguous release only passes CurrentConfig.LookbackDays under its
// least generous interpretation. Unknown/empty precision falls back to a
// full-date parse; a value that doesn't parse under its precision (or the
// fallback) is reported as ok=false and excluded by the caller.
func parseReleaseDate(raw, precision string) (time.Time, bool) {
	layout := "2006-01-02"
	switch precision {
	case "month":
		layout = "2006-01"
	case "year":
		layout = "2006"
	}
	t, err := time.Parse(layout, raw)
	return t, err == nil
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

// maxArtistAlbumsPageSize and maxAlbumTracksPageSize bound how large a
// single page walkPages requests from each endpoint. Spotify documents a
// default max `limit` of 50 for both, but this app's Development Mode
// access empirically rejects `limit>10` on GET /artists/{id}/albums with
// "400 Invalid limit" (confirmed live; undocumented) while
// GET /albums/{id}/tracks accepts the full 50. walkPages still fetches as
// many pages as needed to reach maxItems — this only shrinks each
// individual request.
const (
	maxArtistAlbumsPageSize = 10
	maxAlbumTracksPageSize  = 50
)

// walkPages bounded-walks a Spotify paginated resource by offset,
// accumulating up to maxItems items, stopping early once Spotify returns
// fewer items than requested (no more pages). It never follows
// Paging.Next — every existing spotify.Client method re-requests by
// offset, not by URL, and this matches that. maxItems <= 0 returns
// immediately with no request made. maxPageSize caps each individual
// request's `limit` — see maxArtistAlbumsPageSize/maxAlbumTracksPageSize.
func walkPages[T any](maxItems, maxPageSize int, fetch func(limit, offset int) (spotify.Paging[T], error)) ([]T, error) {
	if maxItems <= 0 {
		return nil, nil
	}

	var items []T
	offset := 0
	for len(items) < maxItems {
		pageSize := maxItems - len(items)
		if pageSize > maxPageSize {
			pageSize = maxPageSize
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

// CurrentHandler exposes POST /api/discovery/current. Runs DiscoverCurrent
// once per call — no persistence, no request body, no query parameters.
// It never modifies the official Spotify playlist and never selects a
// candidate.
func (s *Service) CurrentHandler(w http.ResponseWriter, r *http.Request) {
	result, err := s.DiscoverCurrent(r.Context())
	if err != nil {
		log.Printf("current discovery failed: %v", err)
		writeDiscoveryError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// writeDiscoveryError maps a discovery run's top-level error to an HTTP
// status. DiscoverClassic/DiscoverCurrent only ever return a
// connection-level error here (every other failure is recorded on Result
// instead), so this is a small,
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
