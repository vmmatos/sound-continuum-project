package discovery

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
	"github.com/vmmatos/sound-continuum-project/internal/spotify"
)

// DefaultRecentTrackLookbackDays is Card #37's documented default: a track
// is recently used if it was added to the official playlist within the
// last 28 days. Overridable via RECENT_TRACK_LOOKBACK_DAYS (see
// cmd/server/main.go), following this repo's PORT/SQLITE_PATH precedent —
// unlike Config/CurrentConfig/EmergingConfig, this is a single,
// curator-tunable editorial knob, not an internal crawl bound, so it's a
// plain int rather than its own config struct.
const DefaultRecentTrackLookbackDays = 28

// ReasonRecentlyUsed is the only exclusion reason FilterRecentTracks
// produces — this is a single-purpose filter, not a general rule engine.
const ReasonRecentlyUsed = "recently_used"

// RecentlyUsedCandidate is a candidate FilterRecentTracks excluded because
// it was found in the official playlist within the configured lookback
// window. The original CandidateTrack is kept intact — Status stays
// StatusDiscovered; this is a temporary filtering decision, not an
// editorial rejection.
type RecentlyUsedCandidate struct {
	Candidate  candidate.CandidateTrack
	LastUsedAt time.Time
	Reason     string
}

// RecentTrackFilterResult is FilterRecentTracks' output: every input
// candidate lands in exactly one of EligibleCandidates or
// RecentlyUsedCandidates, plus the operational counts the API surfaces.
type RecentTrackFilterResult struct {
	EligibleCandidates     []candidate.CandidateTrack
	RecentlyUsedCandidates []RecentlyUsedCandidate

	TotalCandidates   int
	EligibleCount     int
	RecentlyUsedCount int
	LookbackDays      int
	// PlaylistTracksInspected is the number of distinct Spotify track IDs
	// found across the official playlist's items (episodes, unavailable
	// items, and duplicate occurrences of the same track are not counted
	// separately).
	PlaylistTracksInspected int
}

// playlistItemsPageSize is the page size FilterRecentTracks requests per
// call to PlaylistItems. Spotify's current documented maximum for this
// endpoint is 50.
const playlistItemsPageSize = 50

// FilterRecentTracks splits candidates into eligible and recently-used
// editorial pools by comparing each candidate's Spotify track ID against
// the official Sound Continuum playlist's own added_at history — the
// source of truth for what was recently published. A candidate is
// "recently used" when its Spotify track ID's most recent added_at in the
// playlist is on or after (now - LookbackDays); see recentTrackIndex for
// the exact boundary.
//
// Returns an error — never a fallback empty/all-eligible result — if the
// official playlist isn't configured yet or its items can't be retrieved
// (Spotify connection failure or API error), so a temporary Spotify
// outage can never silently bypass the repetition guardrail. A candidate
// with no Spotify track ID (e.g. a future non-Spotify source) can never
// match the playlist and always stays eligible.
func (s *Service) FilterRecentTracks(ctx context.Context, candidates []candidate.CandidateTrack) (RecentTrackFilterResult, error) {
	playlist, err := s.spotify.OfficialPlaylist(ctx)
	if err != nil {
		return RecentTrackFilterResult{}, err
	}

	index, err := s.recentTrackIndex(ctx, playlist.SpotifyPlaylistID)
	if err != nil {
		return RecentTrackFilterResult{}, err
	}

	cutoff := s.now().AddDate(0, 0, -s.recentTrackLookbackDays)

	result := RecentTrackFilterResult{
		TotalCandidates:         len(candidates),
		LookbackDays:            s.recentTrackLookbackDays,
		PlaylistTracksInspected: len(index),
	}
	for _, c := range candidates {
		lastUsedAt, ok := index[c.SpotifyTrackID]
		if ok && c.SpotifyTrackID != "" && !lastUsedAt.Before(cutoff) {
			result.RecentlyUsedCandidates = append(result.RecentlyUsedCandidates, RecentlyUsedCandidate{
				Candidate:  c,
				LastUsedAt: lastUsedAt,
				Reason:     ReasonRecentlyUsed,
			})
			continue
		}
		result.EligibleCandidates = append(result.EligibleCandidates, c)
	}
	result.EligibleCount = len(result.EligibleCandidates)
	result.RecentlyUsedCount = len(result.RecentlyUsedCandidates)
	return result, nil
}

// recentTrackIndex walks every item of the official playlist (following
// pagination in full — the card explicitly forbids assuming the first
// page is enough) and returns each distinct Spotify track ID's most
// recent added_at. Episodes, unavailable items, and items with no track
// ID or an unparsable added_at are skipped individually rather than
// aborting the scan; a duplicate track ID keeps the maximum added_at.
func (s *Service) recentTrackIndex(ctx context.Context, playlistID string) (map[string]time.Time, error) {
	index := make(map[string]time.Time)
	offset := 0
	for {
		page, err := s.spotify.PlaylistItems(ctx, playlistID, playlistItemsPageSize, offset)
		if err != nil {
			return nil, err
		}

		for _, item := range page.Items {
			if item.ItemType != "track" || item.Track == nil || item.Track.ID == "" {
				continue
			}
			addedAt, err := time.Parse(time.RFC3339, item.AddedAt)
			if err != nil {
				continue
			}
			if existing, ok := index[item.Track.ID]; !ok || addedAt.After(existing) {
				index[item.Track.ID] = addedAt
			}
		}

		offset += len(page.Items)
		if len(page.Items) == 0 || offset >= page.Total {
			break
		}
	}
	return index, nil
}

// writeRecentTrackFilterError maps a FilterRecentTracks error to an HTTP
// status, mirroring writeDiscoveryError/writeSpotifyError. A playlist-
// retrieval failure must never look like a successful empty pool.
func writeRecentTrackFilterError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, spotify.ErrNotConnected):
		http.Error(w, "Spotify is not connected", http.StatusServiceUnavailable)
	case errors.Is(err, spotify.ErrInvalidGrant):
		http.Error(w, "Spotify authorization required", http.StatusUnauthorized)
	case errors.Is(err, spotify.ErrOfficialPlaylistNotConfigured):
		http.Error(w, "Sound Continuum playlist is not initialized", http.StatusServiceUnavailable)
	default:
		http.Error(w, "recent track filtering failed", http.StatusBadGateway)
	}
}
