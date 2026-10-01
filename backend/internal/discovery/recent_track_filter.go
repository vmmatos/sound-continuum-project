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
	index, err := s.PlaylistTrackHistory(ctx)
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

// PlaylistTrackHistory returns the official Sound Continuum playlist's
// full track-appearance history: every distinct Spotify track ID found in
// it, mapped to its most recent added_at. This is the single source of
// playlist-history retrieval in the codebase — FilterRecentTracks (the
// 28-day hard eligibility cutoff, Card #37) and any factor needing
// playlist-history recency as a softer signal (e.g. scoring.Freshness,
// Card #42; scoring.RepetitionPenalty, Card #45) both build on this one
// method rather than each retrieving and paginating the playlist
// independently.
//
// Returns an error — never an empty map — if the official playlist isn't
// configured yet or its items can't be retrieved, so a Spotify outage or
// missing playlist can never be mistaken for "nothing has ever been
// played."
func (s *Service) PlaylistTrackHistory(ctx context.Context) (map[string]time.Time, error) {
	trackIndex, _, err := s.playlistHistoryIndexes(ctx)
	if err != nil {
		return nil, err
	}
	return trackIndex, nil
}

// PlaylistArtistHistory returns the official Sound Continuum playlist's
// full artist-appearance history: every distinct Spotify artist ID found
// across all tracked items, mapped to the most recent added_at of any
// track by that artist. Built from the same playlist walk as
// PlaylistTrackHistory, reused rather than duplicated — see that method's
// doc comment. Returns an error — never an empty map — under the same
// conditions as PlaylistTrackHistory.
func (s *Service) PlaylistArtistHistory(ctx context.Context) (map[string]time.Time, error) {
	_, artistIndex, err := s.playlistHistoryIndexes(ctx)
	if err != nil {
		return nil, err
	}
	return artistIndex, nil
}

// playlistHistoryIndexes resolves the official playlist and walks it once
// via recentTrackIndex's underlying logic, building both the track and
// artist history indexes in the same pagination pass.
func (s *Service) playlistHistoryIndexes(ctx context.Context) (trackIndex, artistIndex map[string]time.Time, err error) {
	playlist, err := s.spotify.OfficialPlaylist(ctx)
	if err != nil {
		return nil, nil, err
	}
	return s.recentTrackAndArtistIndex(ctx, playlist.SpotifyPlaylistID)
}

// recentTrackIndex walks every item of the official playlist (following
// pagination in full — the card explicitly forbids assuming the first
// page is enough) and returns each distinct Spotify track ID's most
// recent added_at. Episodes, unavailable items, and items with no track
// ID or an unparsable added_at are skipped individually rather than
// aborting the scan; a duplicate track ID keeps the maximum added_at.
func (s *Service) recentTrackIndex(ctx context.Context, playlistID string) (map[string]time.Time, error) {
	trackIndex, _, err := s.recentTrackAndArtistIndex(ctx, playlistID)
	if err != nil {
		return nil, err
	}
	return trackIndex, nil
}

// recentTrackAndArtistIndex is recentTrackIndex's shared implementation,
// extended (Card #45) to also build an artist-ID -> most-recent-added_at
// index from the same walk, so Repetition Penalty's artist dimension
// needs no second playlist retrieval. The track-index behavior is
// byte-for-byte unchanged from Card #37/#42.
func (s *Service) recentTrackAndArtistIndex(ctx context.Context, playlistID string) (trackIndex, artistIndex map[string]time.Time, err error) {
	trackIndex = make(map[string]time.Time)
	artistIndex = make(map[string]time.Time)
	offset := 0
	for {
		page, err := s.spotify.PlaylistItems(ctx, playlistID, playlistItemsPageSize, offset)
		if err != nil {
			return nil, nil, err
		}

		for _, item := range page.Items {
			if item.ItemType != "track" || item.Track == nil || item.Track.ID == "" {
				continue
			}
			addedAt, err := time.Parse(time.RFC3339, item.AddedAt)
			if err != nil {
				continue
			}
			if existing, ok := trackIndex[item.Track.ID]; !ok || addedAt.After(existing) {
				trackIndex[item.Track.ID] = addedAt
			}
			for _, artist := range item.Track.Artists {
				if artist.ID == "" {
					continue
				}
				if existing, ok := artistIndex[artist.ID]; !ok || addedAt.After(existing) {
					artistIndex[artist.ID] = addedAt
				}
			}
		}

		offset += len(page.Items)
		if len(page.Items) == 0 || offset >= page.Total {
			break
		}
	}
	return trackIndex, artistIndex, nil
}

// writeRecentTrackFilterError maps a FilterRecentTracks error to an HTTP
// status, mirroring writeDiscoveryError/writeSpotifyError. A playlist-
// retrieval failure must never look like a successful empty pool.
func writeRecentTrackFilterError(w http.ResponseWriter, err error) {
	if writeConnectionError(w, err) {
		return
	}
	switch {
	case errors.Is(err, spotify.ErrOfficialPlaylistNotConfigured):
		http.Error(w, "Sound Continuum playlist is not initialized", http.StatusServiceUnavailable)
	default:
		http.Error(w, "recent track filtering failed", http.StatusBadGateway)
	}
}
