// Package candidate defines the Sound Continuum domain model for a
// candidate track: a piece of music discovered and under editorial
// consideration for a future Sound Continuum edition. It is deliberately
// independent of package spotify's response types — a candidate is Sound
// Continuum's own editorial concept, not a copy of external music
// metadata, and not every future candidate will originate from Spotify.
package candidate

import "time"

// Source identifies where a candidate was discovered.
type Source string

const (
	SourceSpotify Source = "Spotify"
	SourceLastFM  Source = "Last.fm"
	SourceManual  Source = "Manual"
)

// Valid reports whether s is a supported discovery source.
func (s Source) Valid() bool {
	switch s {
	case SourceSpotify, SourceLastFM, SourceManual:
		return true
	default:
		return false
	}
}

// Category is the editorial context in which a candidate is being
// considered. It is editorial metadata, not a Spotify or genre
// classification.
type Category string

const (
	CategoryPast       Category = "Past"
	CategoryPresent    Category = "Present"
	CategoryEmerging   Category = "Emerging"
	CategoryNewRelease Category = "New Release"
)

// Valid reports whether c is one of the four supported editorial
// categories.
func (c Category) Valid() bool {
	switch c {
	case CategoryPast, CategoryPresent, CategoryEmerging, CategoryNewRelease:
		return true
	default:
		return false
	}
}

// Type describes the editorial mode through which a candidate entered
// Sound Continuum's editorial process — distinct from Category, which
// describes where the candidate sits editorially. The two are
// independent: e.g. a Discovery-type candidate is not necessarily
// Emerging, and a Classic-type candidate is not necessarily Past.
type Type string

const (
	TypeClassic   Type = "Classic"
	TypeCurrent   Type = "Current"
	TypeDiscovery Type = "Discovery"
)

// Valid reports whether t is one of the three supported candidate types.
func (t Type) Valid() bool {
	switch t {
	case TypeClassic, TypeCurrent, TypeDiscovery:
		return true
	default:
		return false
	}
}

// Status is a candidate's current editorial lifecycle state. It is
// deliberately small — no approval workflow, roles, or voting states.
type Status string

const (
	StatusDiscovered  Status = "discovered"
	StatusUnderReview Status = "under review"
	StatusSelected    Status = "selected"
	StatusRejected    Status = "rejected"
)

// Valid reports whether s is one of the four supported lifecycle states.
func (s Status) Valid() bool {
	switch s {
	case StatusDiscovered, StatusUnderReview, StatusSelected, StatusRejected:
		return true
	default:
		return false
	}
}

// ID is a candidate's internal Sound Continuum identity — distinct from
// SpotifyTrackID, which is an external reference that only applies when
// Source is SourceSpotify.
type ID string

// CandidateTrack is a track discovered as a potential future Sound
// Continuum edition pick. It is not yet selected, approved, or scheduled —
// see Status. It carries only the track metadata Sound Continuum's
// editorial process actually needs (title/artist for identification), not
// a copy of Spotify's full Track/Artist/Album response.
type CandidateTrack struct {
	ID             ID
	SpotifyTrackID string // external reference; empty unless Source == SourceSpotify
	Source         Source
	Category       Category
	Type           Type
	Status         Status

	TrackTitle  string
	TrackArtist string

	DiscoveryReason     string
	EditorialNote       string
	PotentialConnection string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewCandidateTrackParams are the inputs to NewCandidateTrack. Status is
// not settable here — a newly discovered candidate always starts at
// StatusDiscovered.
type NewCandidateTrackParams struct {
	ID             ID
	SpotifyTrackID string
	Source         Source
	Category       Category
	Type           Type

	TrackTitle  string
	TrackArtist string

	DiscoveryReason     string
	EditorialNote       string
	PotentialConnection string
}

// NewCandidateTrack builds a CandidateTrack in its initial discovered
// state, validating it before returning.
func NewCandidateTrack(p NewCandidateTrackParams) (CandidateTrack, error) {
	now := time.Now()
	c := CandidateTrack{
		ID:             p.ID,
		SpotifyTrackID: p.SpotifyTrackID,
		Source:         p.Source,
		Category:       p.Category,
		Type:           p.Type,
		Status:         StatusDiscovered,

		TrackTitle:  p.TrackTitle,
		TrackArtist: p.TrackArtist,

		DiscoveryReason:     p.DiscoveryReason,
		EditorialNote:       p.EditorialNote,
		PotentialConnection: p.PotentialConnection,

		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := c.Validate(); err != nil {
		return CandidateTrack{}, err
	}
	return c, nil
}

// Validate checks c's domain constraints. Exported so it can be run
// against a CandidateTrack built outside NewCandidateTrack (e.g. read back
// from a future persistence layer), independent of any editorial workflow
// this card does not implement.
func (c CandidateTrack) Validate() error {
	if c.ID == "" {
		return ErrEmptyCandidateID
	}
	if !c.Source.Valid() {
		return ErrInvalidSource
	}
	if !c.Category.Valid() {
		return ErrInvalidCategory
	}
	if !c.Type.Valid() {
		return ErrInvalidType
	}
	if !c.Status.Valid() {
		return ErrInvalidStatus
	}
	if c.Source == SourceSpotify && c.SpotifyTrackID == "" {
		return ErrMissingSpotifyTrackID
	}
	return nil
}
