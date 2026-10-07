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
// never regenerated, so it's a stable, natural key. Only "selected" is ever
// written today (Keep has no "unselect"/"reject" counterpart yet) — status
// is still a column, not an implicit boolean, so a future state doesn't
// require a schema migration.
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

// statusSelected is the only status this package ever writes.
const statusSelected = "selected"

// Keep marks candidateID as selected. Idempotent by construction: the
// ON CONFLICT upsert means repeating Keep for the same ID always leaves
// exactly one row in the same end state, never a duplicate and never an
// error.
func (s *Store) Keep(ctx context.Context, candidateID string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO candidate_selection (candidate_id, status, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(candidate_id) DO UPDATE SET
			status = excluded.status,
			updated_at = excluded.updated_at
	`, candidateID, statusSelected, time.Now().Unix())
	return err
}

// AllSelected returns the set of every candidate ID currently marked
// selected. Used by review.Service.ReviewPool to overlay persisted
// selection state onto a freshly-discovered candidate pool (a candidate
// pool is never itself persisted — see docs/memory/decisions.md).
func (s *Store) AllSelected(ctx context.Context) (map[string]struct{}, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT candidate_id FROM candidate_selection WHERE status = ?
	`, statusSelected)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	selected := make(map[string]struct{})
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		selected[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return selected, nil
}
