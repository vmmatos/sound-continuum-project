// Package selection persists the curator's "Keep" decisions (Card #56) —
// which candidates have been marked selected. It is the smallest viable
// persistence addition consistent with this MVP's existing architecture:
// one new SQLite table, reusing the same *sql.DB and
// CREATE TABLE IF NOT EXISTS pattern already established by
// backend/internal/spotify/store.go, not a new datastore or ORM.
//
// Candidate selection is kept separate from package spotify (which owns
// Spotify-specific persistence only) and from package candidate (a pure
// domain package with no DB dependency) — matching this repo's existing
// convention of a small, flat, feature-named package per concern (see
// candidate/discovery/review/scoring/lastfm and docs/memory/decisions.md).
package selection

import (
	"context"
	"database/sql"
	"time"
)

// schema creates the table backing every candidate's selection state.
// Candidate ID is the primary key — a candidate.ID (for Spotify-sourced
// candidates, the Spotify track ID, see decisions.md's Card #33 entry) is
// never regenerated, so it's a stable, natural key. "selected" (Keep) and
// "under review" (Maybe, Card #57) are the two statuses ever written; a
// cleared decision deletes the row rather than writing a third "neutral"
// status. One row per candidate means writing one status structurally
// overwrites the other — Keep and Maybe are mutually exclusive by
// construction, not by application-level checking.
const schema = `
CREATE TABLE IF NOT EXISTS candidate_selection (
	candidate_id TEXT PRIMARY KEY,
	status       TEXT NOT NULL,
	updated_at   INTEGER NOT NULL
);
`

// Store persists candidate selection state in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates the candidate_selection table if it doesn't exist yet.
func NewStore(db *sql.DB) (*Store, error) {
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// statusSelected and statusUnderReview are the only statuses this package
// ever writes.
const (
	statusSelected    = "selected"
	statusUnderReview = "under review"
)

// setStatus upserts candidateID's status. Idempotent by construction: the
// ON CONFLICT upsert means repeating the same status for the same ID always
// leaves exactly one row in the same end state, never a duplicate and never
// an error. Writing a different status for an already-decided candidate
// overwrites the row — this is what makes Keep and Maybe mutually exclusive
// with no extra check.
func (s *Store) setStatus(ctx context.Context, candidateID, status string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO candidate_selection (candidate_id, status, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(candidate_id) DO UPDATE SET
			status = excluded.status,
			updated_at = excluded.updated_at
	`, candidateID, status, time.Now().Unix())
	return err
}

// Keep marks candidateID as selected.
func (s *Store) Keep(ctx context.Context, candidateID string) error {
	return s.setStatus(ctx, candidateID, statusSelected)
}

// Maybe marks candidateID as under review (Card #57) — the curator is
// undecided but wants to keep it in consideration.
func (s *Store) Maybe(ctx context.Context, candidateID string) error {
	return s.setStatus(ctx, candidateID, statusUnderReview)
}

// Clear removes any persisted decision for candidateID, returning it to the
// neutral/discovered state. Idempotent: deleting a row that doesn't exist is
// a no-op, never an error.
func (s *Store) Clear(ctx context.Context, candidateID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM candidate_selection WHERE candidate_id = ?
	`, candidateID)
	return err
}

// allWithStatus returns the set of every candidate ID currently recorded
// with the given status.
func (s *Store) allWithStatus(ctx context.Context, status string) (map[string]struct{}, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT candidate_id FROM candidate_selection WHERE status = ?
	`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make(map[string]struct{})
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

// AllSelected returns the set of every candidate ID currently marked
// selected. Used by review.Service.ReviewPool to overlay persisted
// selection state onto a freshly-discovered candidate pool (a candidate
// pool is never itself persisted — see docs/memory/decisions.md).
func (s *Store) AllSelected(ctx context.Context) (map[string]struct{}, error) {
	return s.allWithStatus(ctx, statusSelected)
}

// AllUnderReview returns the set of every candidate ID currently marked
// under review (Maybe, Card #57), mirroring AllSelected.
func (s *Store) AllUnderReview(ctx context.Context) (map[string]struct{}, error) {
	return s.allWithStatus(ctx, statusUnderReview)
}
