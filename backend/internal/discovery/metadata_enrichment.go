package discovery

import (
	"context"
	"errors"
	"net/http"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
	"github.com/vmmatos/sound-continuum-project/internal/spotify"
)

// EnrichmentFailure records one candidate whose Spotify metadata could not
// be retrieved. The candidate itself is still kept (see
// EnrichCandidateMetadata) with a nil Metadata — this failure is what makes
// that visible rather than silent.
type EnrichmentFailure struct {
	CandidateID    candidate.ID
	SpotifyTrackID string
	Reason         string // "not_found", "rate_limited", "unauthorized", "malformed_response", or "api_failure"
}

// EnrichmentResult reports what one EnrichCandidateMetadata call did.
type EnrichmentResult struct {
	EnrichedCandidates []candidate.CandidateTrack
	Failures           []EnrichmentFailure

	EnrichedCount int
	// SkippedCount counts candidates with no Spotify track ID to enrich
	// from (e.g. a future Manual-source candidate) — no Spotify request is
	// attempted for these, and no fake metadata is generated.
	SkippedCount int
}

// EnrichCandidateMetadata attaches CandidateMetadata to every candidate that
// carries a Spotify track ID, using the existing Spotify client
// (spotifyCatalogue.Track) — no new Spotify client, auth, or endpoint.
// Candidates keep their Type/Category/Status/identity unchanged; only
// Metadata is set.
//
// A candidate with no SpotifyTrackID is passed through unmodified and
// counted on SkippedCount, never given fabricated metadata.
//
// Aborts the whole call only on a Spotify connection failure
// (ErrNotConnected/ErrInvalidGrant), matching every Discover* method's own
// convention. Any other per-track failure (not found, rate limited,
// unauthorized, malformed response, or a generic API failure) is recorded
// on EnrichmentResult.Failures; that candidate is still returned, with
// Metadata left nil rather than a fabricated/partial value, and the run
// continues to the next candidate.
func (s *Service) EnrichCandidateMetadata(ctx context.Context, candidates []candidate.CandidateTrack) (EnrichmentResult, error) {
	var result EnrichmentResult

	for _, c := range candidates {
		if c.SpotifyTrackID == "" {
			result.SkippedCount++
			result.EnrichedCandidates = append(result.EnrichedCandidates, c)
			continue
		}

		track, err := retryOn429(s.sleep, func() (spotify.Track, error) {
			return s.spotify.Track(ctx, c.SpotifyTrackID)
		})
		if isConnectionError(err) {
			return result, err
		}
		if err != nil {
			result.Failures = append(result.Failures, EnrichmentFailure{
				CandidateID:    c.ID,
				SpotifyTrackID: c.SpotifyTrackID,
				Reason:         enrichmentFailureReason(err),
			})
			result.EnrichedCandidates = append(result.EnrichedCandidates, c)
			continue
		}

		metadata := mapSpotifyTrack(track)
		c.Metadata = &metadata
		result.EnrichedCount++
		result.EnrichedCandidates = append(result.EnrichedCandidates, c)
	}

	return result, nil
}

// mapSpotifyTrack maps a Spotify track response into Sound Continuum's own
// candidate metadata. This is the one place a provider response shape
// becomes a domain shape — Spotify's Track/Artist/Album/Image types never
// leak past this function. No popularity, followers, or genres are mapped:
// those fields are deprecated in the current Spotify API and are not
// ranking/editorial signals this card introduces.
func mapSpotifyTrack(t spotify.Track) candidate.CandidateMetadata {
	artists := make([]candidate.CandidateArtist, 0, len(t.Artists))
	for _, a := range t.Artists {
		artists = append(artists, candidate.CandidateArtist{
			SpotifyArtistID: a.ID,
			Name:            a.Name,
			SpotifyURL:      a.ExternalURLs.Spotify,
		})
	}

	images := make([]candidate.CandidateImage, 0, len(t.Album.Images))
	for _, img := range t.Album.Images {
		images = append(images, candidate.CandidateImage{
			URL:    img.URL,
			Width:  img.Width,
			Height: img.Height,
		})
	}

	return candidate.CandidateMetadata{
		Title:      t.Name,
		Artists:    artists,
		DurationMS: t.DurationMS,
		Explicit:   t.Explicit,
		SpotifyURL: t.ExternalURLs.Spotify,
		SpotifyURI: t.URI,
		Album: candidate.CandidateAlbum{
			SpotifyAlbumID:       t.Album.ID,
			Name:                 t.Album.Name,
			AlbumType:            t.Album.AlbumType,
			ReleaseDate:          t.Album.ReleaseDate,
			ReleaseDatePrecision: t.Album.ReleaseDatePrecision,
			Artwork:              images,
			SpotifyURL:           t.Album.ExternalURLs.Spotify,
		},
	}
}

// enrichmentFailureReason classifies a non-connection Track lookup failure
// for EnrichmentFailure.Reason, reusing package spotify's existing error
// taxonomy rather than a second error model.
func enrichmentFailureReason(err error) string {
	switch {
	case errors.Is(err, spotify.ErrNotFound):
		return "not_found"
	case errors.Is(err, spotify.ErrRateLimited):
		return "rate_limited"
	case errors.Is(err, spotify.ErrUnauthorized), errors.Is(err, spotify.ErrForbidden):
		return "unauthorized"
	case errors.Is(err, spotify.ErrDecode):
		return "malformed_response"
	default:
		return "api_failure"
	}
}

// writeEnrichmentError maps EnrichCandidateMetadata's top-level error (a
// Spotify connection failure — every other failure is recorded on
// EnrichmentResult instead) to an HTTP status via the shared
// writeConnectionError.
func writeEnrichmentError(w http.ResponseWriter, err error) {
	if writeConnectionError(w, err) {
		return
	}
	http.Error(w, "candidate metadata enrichment failed", http.StatusBadGateway)
}
