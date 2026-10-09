// Package edition defines the Edition domain model — one weekly Sound
// Continuum chapter, from the curator's confirmed track selection through
// publishing to Spotify and eventual archival (Card #139).
//
// It is a new, flat, feature-named package, sibling to candidate/selection/
// review/spotify — matching this repo's existing convention of one small
// package per concern rather than an internal/domain layer. It reuses
// candidate.CandidateMetadata for confirmed-track metadata instead of a
// third track-metadata shape.
//
// This is deliberately not the same concept as scoring.EditionTrack/
// CurrentEditionContext (Card #44) or musicaldna.WeeklyDirection.EditionID
// (Card #41) — those are transient, non-persisted, function-scoped inputs
// to the scoring factors, documented in decisions.md as intentionally not a
// real Edition entity ahead of need. This package is that entity, now that
// a real need (Card #61's confirmed playlist having nowhere durable to
// live) exists. Neither existing concept is modified by this package.
package edition

import (
	"fmt"
	"slices"
	"time"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
)

// Status is an Edition's lifecycle stage.
type Status string

const (
	StatusDraft      Status = "draft"
	StatusConfirmed  Status = "confirmed"
	StatusPublishing Status = "publishing"
	StatusPublished  Status = "published"
	StatusArchived   Status = "archived"
)

// Valid reports whether s is one of the five supported lifecycle stages.
func (s Status) Valid() bool {
	switch s {
	case StatusDraft, StatusConfirmed, StatusPublishing, StatusPublished, StatusArchived:
		return true
	default:
		return false
	}
}

// transitions enumerates every allowed Status change. Publishing can return
// to Confirmed on a recoverable failure — a retry, never a dead end — but
// every other edge is one-directional and explicit; anything not listed
// here is rejected. Confirmed does not list itself: re-confirming an
// already-Confirmed edition (Card #61's Edit → reorder → reconfirm flow) is
// a snapshot update, not a Status transition, and is handled separately by
// Service.ConfirmFromReview.
var transitions = map[Status][]Status{
	StatusDraft:      {StatusConfirmed},
	StatusConfirmed:  {StatusPublishing},
	StatusPublishing: {StatusPublished, StatusConfirmed},
	StatusPublished:  {StatusArchived},
	StatusArchived:   {},
}

// CanTransitionTo reports whether moving from s to target is a valid
// lifecycle transition.
func (s Status) CanTransitionTo(target Status) bool {
	return slices.Contains(transitions[s], target)
}

// ConfirmedTrack is one track in an Edition's confirmed, ordered snapshot.
// It reuses candidate.CandidateMetadata (Card #38's shape) rather than a
// third track-metadata type — the "essential metadata to reconstruct the
// confirmed selection" this card calls for, nothing more. SpotifyTrackID is
// carried alongside Metadata (not solely inside it) because it is the
// stable identity a future publishing step needs, independent of whether
// enrichment metadata was ever available for this track.
type ConfirmedTrack struct {
	SpotifyTrackID string
	Metadata       candidate.CandidateMetadata
}

// Edition is one weekly Sound Continuum chapter.
type Edition struct {
	ID        string
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time

	// ConfirmedTracks is the exact ordered track snapshot captured at
	// confirmation time. Nil until Confirmed. Order is array order — never
	// re-derived from Rank/FinalScore, discovery order, or artist name.
	ConfirmedTracks []ConfirmedTrack
	ConfirmedAt     *time.Time

	// SpotifyPlaylistID/URL are recorded once a publish attempt succeeds —
	// the reference a retry needs to reconcile remote state instead of
	// blindly creating a duplicate (Step 7). Empty until then.
	SpotifyPlaylistID  string
	SpotifyPlaylistURL string
	PublishedAt        *time.Time

	// LastPublishAttemptAt/LastPublishError record the most recent
	// publishing attempt's outcome without losing Edition identity or the
	// confirmed snapshot on failure (Step 7). LastPublishError is nil
	// whenever the last recorded attempt did not fail.
	LastPublishAttemptAt *time.Time
	LastPublishError     *string

	ArchivedAt *time.Time
}

// newEditionID generates a plain, collision-safe-in-practice ID with no new
// dependency (this repo has deliberately avoided adding a UUID library —
// see decisions.md's Card #33 entry on reusing candidate.ID instead of
// generating one). Editions are created at most a few times a week by a
// single curator against a single process, so nanosecond-resolution
// timestamps are sufficient; the single-active-edition UNIQUE index is a
// second backstop against any collision regardless.
func newEditionID() string {
	return fmt.Sprintf("edition-%d", time.Now().UnixNano())
}
