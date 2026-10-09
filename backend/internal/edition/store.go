package edition

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// schema creates the editions table. Unlike spotify_connection/
// official_playlist (singleton id=1 rows), this is a keyed-row table — one
// row per weekly chapter over time — matching selection's candidate_
// selection table shape. confirmed_tracks is a single JSON TEXT column
// (encoding/json, no new dependency): the simplest representation that
// preserves exact order by construction (array order), versus a child
// edition_tracks(position, ...) table, which would add a second table, a
// join, and a position column to keep in sync for no consumer that needs
// SQL-level querying over individual confirmed tracks today — see
// docs/memory/decisions.md.
const schema = `
CREATE TABLE IF NOT EXISTS editions (
	id                      TEXT PRIMARY KEY,
	status                  TEXT NOT NULL,
	created_at              INTEGER NOT NULL,
	updated_at              INTEGER NOT NULL,
	confirmed_tracks        TEXT,
	confirmed_at            INTEGER,
	spotify_playlist_id     TEXT NOT NULL DEFAULT '',
	spotify_playlist_url    TEXT NOT NULL DEFAULT '',
	published_at            INTEGER,
	last_publish_attempt_at INTEGER,
	last_publish_error      TEXT,
	archived_at             INTEGER
);
`

// indexSchema enforces the single-active-edition invariant (Step 4) at the
// database level, not only in application code: every non-archived row
// indexes to the same constant expression ((1)), so SQLite's own UNIQUE
// constraint rejects a second one outright. This needs no application-level
// locking or transaction (this repo's backend has neither anywhere) — a
// second concurrent INSERT is serialized by SQLite's own single-writer
// behavior and fails the constraint exactly like a sequential one would.
// Archived rows are excluded by the WHERE clause, so an archived edition
// never blocks the next one.
const indexSchema = `
CREATE UNIQUE INDEX IF NOT EXISTS idx_editions_single_active
ON editions ((1)) WHERE status != 'archived';
`

// Store persists Edition state in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates the editions table and its single-active-edition index
// if they don't exist yet.
func NewStore(db *sql.DB) (*Store, error) {
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}
	if _, err := db.Exec(indexSchema); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// CreateDraft starts a new Edition in StatusDraft. Fails with
// ErrActiveEditionExists if a non-archived Edition already exists —
// enforced by idx_editions_single_active, not a prior SELECT, so it holds
// under concurrent callers too.
func (s *Store) CreateDraft(ctx context.Context) (Edition, error) {
	now := time.Now()
	e := Edition{
		ID:        newEditionID(),
		Status:    StatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO editions (id, status, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`, e.ID, string(e.Status), now.Unix(), now.Unix())
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return Edition{}, ErrActiveEditionExists
		}
		return Edition{}, err
	}
	return e, nil
}

// GetActive returns the one non-archived Edition, or nil if none exists.
func (s *Store) GetActive(ctx context.Context) (*Edition, error) {
	row := s.db.QueryRowContext(ctx, selectColumns+`FROM editions WHERE status != 'archived'`)
	return scanEdition(row)
}

// GetByID returns the Edition with the given ID, or nil if it doesn't exist.
func (s *Store) GetByID(ctx context.Context, id string) (*Edition, error) {
	row := s.db.QueryRowContext(ctx, selectColumns+`FROM editions WHERE id = ?`, id)
	return scanEdition(row)
}

const selectColumns = `
	SELECT id, status, created_at, updated_at, confirmed_tracks, confirmed_at,
	       spotify_playlist_id, spotify_playlist_url, published_at,
	       last_publish_attempt_at, last_publish_error, archived_at
`

func scanEdition(row *sql.Row) (*Edition, error) {
	var e Edition
	var status string
	var createdAt, updatedAt int64
	var confirmedTracksJSON sql.NullString
	var confirmedAt, publishedAt, lastPublishAttemptAt, archivedAt sql.NullInt64
	var lastPublishError sql.NullString

	err := row.Scan(&e.ID, &status, &createdAt, &updatedAt, &confirmedTracksJSON, &confirmedAt,
		&e.SpotifyPlaylistID, &e.SpotifyPlaylistURL, &publishedAt,
		&lastPublishAttemptAt, &lastPublishError, &archivedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	e.Status = Status(status)
	e.CreatedAt = time.Unix(createdAt, 0)
	e.UpdatedAt = time.Unix(updatedAt, 0)
	if confirmedTracksJSON.Valid {
		if err := json.Unmarshal([]byte(confirmedTracksJSON.String), &e.ConfirmedTracks); err != nil {
			return nil, fmt.Errorf("edition: decode confirmed tracks: %w", err)
		}
	}
	if confirmedAt.Valid {
		t := time.Unix(confirmedAt.Int64, 0)
		e.ConfirmedAt = &t
	}
	if publishedAt.Valid {
		t := time.Unix(publishedAt.Int64, 0)
		e.PublishedAt = &t
	}
	if lastPublishAttemptAt.Valid {
		t := time.Unix(lastPublishAttemptAt.Int64, 0)
		e.LastPublishAttemptAt = &t
	}
	if lastPublishError.Valid {
		e.LastPublishError = &lastPublishError.String
	}
	if archivedAt.Valid {
		t := time.Unix(archivedAt.Int64, 0)
		e.ArchivedAt = &t
	}
	return &e, nil
}

// save overwrites every column of an Edition in one statement, guarded by
// WHERE status = fromStatus — an optimistic check against whatever Status
// the caller read just before calling save. It reports whether the row was
// actually updated (false means the Edition no longer has fromStatus, or no
// longer exists — a concurrent change the caller must treat as a failed
// transition, not retry blindly). One shared helper for every write beyond
// CreateDraft, instead of a bespoke UPDATE per transition.
func (s *Store) save(ctx context.Context, e Edition, fromStatus Status) (bool, error) {
	tracksJSON, err := marshalTracks(e.ConfirmedTracks)
	if err != nil {
		return false, err
	}

	res, err := s.db.ExecContext(ctx, `
		UPDATE editions SET
			status = ?, updated_at = ?, confirmed_tracks = ?, confirmed_at = ?,
			spotify_playlist_id = ?, spotify_playlist_url = ?, published_at = ?,
			last_publish_attempt_at = ?, last_publish_error = ?, archived_at = ?
		WHERE id = ? AND status = ?
	`,
		string(e.Status), e.UpdatedAt.Unix(), tracksJSON, unixOrNil(e.ConfirmedAt),
		e.SpotifyPlaylistID, e.SpotifyPlaylistURL, unixOrNil(e.PublishedAt),
		unixOrNil(e.LastPublishAttemptAt), e.LastPublishError, unixOrNil(e.ArchivedAt),
		e.ID, string(fromStatus),
	)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

// marshalTracks returns nil (SQL NULL) for an empty/nil snapshot, rather
// than the literal string "null" or "[]" — there is no confirmed snapshot
// yet, not an empty one.
func marshalTracks(tracks []ConfirmedTrack) (any, error) {
	if len(tracks) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(tracks)
	if err != nil {
		return nil, fmt.Errorf("edition: encode confirmed tracks: %w", err)
	}
	return string(b), nil
}

// unixOrNil converts an optional timestamp to SQL NULL or a Unix-seconds
// integer, matching every other nullable-timestamp column in this package.
func unixOrNil(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Unix()
}

// Confirm captures tracks as id's confirmed, ordered snapshot (Step 2/5).
// Valid from StatusDraft (the first confirmation) or StatusConfirmed itself
// (re-confirming after an Edit → reorder, Card #61's flow — a snapshot
// overwrite, not a Status transition, so it's handled here rather than via
// Status.CanTransitionTo). Any other status — Publishing, Published,
// Archived — returns ErrSnapshotLocked: once publishing has started, the
// confirmed boundary is final (Step 3/6).
func (s *Store) Confirm(ctx context.Context, id string, tracks []ConfirmedTrack) (Edition, error) {
	if len(tracks) == 0 {
		return Edition{}, ErrEmptySnapshot
	}

	current, err := s.GetByID(ctx, id)
	if err != nil {
		return Edition{}, err
	}
	if current == nil {
		return Edition{}, ErrEditionNotFound
	}
	if current.Status != StatusDraft && current.Status != StatusConfirmed {
		return Edition{}, ErrSnapshotLocked
	}

	fromStatus := current.Status
	now := time.Now()
	next := *current
	next.Status = StatusConfirmed
	next.UpdatedAt = now
	next.ConfirmedTracks = tracks
	next.ConfirmedAt = &now

	applied, err := s.save(ctx, next, fromStatus)
	if err != nil {
		return Edition{}, err
	}
	if !applied {
		return Edition{}, ErrSnapshotLocked
	}
	return next, nil
}

// transition is the shared body for every pure Status change beyond
// Confirm: validate via Status.CanTransitionTo, mutate a copy, save it
// guarded by the status just read.
func (s *Store) transition(ctx context.Context, id string, to Status, mutate func(e *Edition, now time.Time)) (Edition, error) {
	current, err := s.GetByID(ctx, id)
	if err != nil {
		return Edition{}, err
	}
	if current == nil {
		return Edition{}, ErrEditionNotFound
	}
	if !current.Status.CanTransitionTo(to) {
		return Edition{}, ErrInvalidTransition
	}

	fromStatus := current.Status
	now := time.Now()
	next := *current
	next.Status = to
	next.UpdatedAt = now
	mutate(&next, now)

	applied, err := s.save(ctx, next, fromStatus)
	if err != nil {
		return Edition{}, err
	}
	if !applied {
		return Edition{}, ErrInvalidTransition
	}
	return next, nil
}

// StartPublishing moves id from Confirmed to Publishing, recording the
// attempt timestamp. ConfirmedTracks is untouched — publishing operates on
// the confirmed snapshot, never a fresh one (Step 3).
func (s *Store) StartPublishing(ctx context.Context, id string) (Edition, error) {
	return s.transition(ctx, id, StatusPublishing, func(e *Edition, now time.Time) {
		e.LastPublishAttemptAt = &now
		e.LastPublishError = nil
	})
}

// RecordPublishSuccess moves id from Publishing to Published, recording the
// resulting Spotify playlist reference and publication timestamp.
// ConfirmedTracks/ConfirmedAt are untouched — publication metadata never
// changes the confirmed order (Step 3/7).
func (s *Store) RecordPublishSuccess(ctx context.Context, id, spotifyPlaylistID, spotifyPlaylistURL string) (Edition, error) {
	return s.transition(ctx, id, StatusPublished, func(e *Edition, now time.Time) {
		e.SpotifyPlaylistID = spotifyPlaylistID
		e.SpotifyPlaylistURL = spotifyPlaylistURL
		e.PublishedAt = &now
	})
}

// RecordPublishFailure moves id from Publishing back to Confirmed — a
// recoverable failure, never a terminal one (Step 3) — recording the
// error. ConfirmedTracks/ConfirmedAt/ID are untouched, so a retry can start
// publishing again without losing the confirmed snapshot or Edition
// identity (Step 7).
func (s *Store) RecordPublishFailure(ctx context.Context, id, errMsg string) (Edition, error) {
	return s.transition(ctx, id, StatusConfirmed, func(e *Edition, now time.Time) {
		e.LastPublishError = &errMsg
	})
}

// Archive moves id from Published to Archived, recording the archival
// timestamp. Only reachable from Published — Status.CanTransitionTo
// rejects every other source status, satisfying Step 8's "only a
// successfully published Edition may be archived."
func (s *Store) Archive(ctx context.Context, id string) (Edition, error) {
	return s.transition(ctx, id, StatusArchived, func(e *Edition, now time.Time) {
		e.ArchivedAt = &now
	})
}
