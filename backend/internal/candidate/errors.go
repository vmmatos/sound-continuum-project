package candidate

import "errors"

// Sentinels for CandidateTrack validation failures. Use errors.Is against
// these.
var (
	// ErrEmptyCandidateID is returned when a candidate has no internal
	// identity.
	ErrEmptyCandidateID = errors.New("candidate: id must not be empty")

	// ErrInvalidSource is returned when Source is not one of the supported
	// discovery sources.
	ErrInvalidSource = errors.New("candidate: unsupported source")

	// ErrInvalidCategory is returned when Category is not one of the
	// supported editorial categories.
	ErrInvalidCategory = errors.New("candidate: unsupported editorial category")

	// ErrInvalidStatus is returned when Status is not one of the supported
	// lifecycle states.
	ErrInvalidStatus = errors.New("candidate: unsupported status")

	// ErrMissingSpotifyTrackID is returned when Source is SourceSpotify but
	// SpotifyTrackID is empty — a Spotify-sourced candidate must retain its
	// external reference.
	ErrMissingSpotifyTrackID = errors.New("candidate: spotify track id required when source is Spotify")
)
