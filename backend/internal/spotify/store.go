package spotify

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// schema creates the single-row table backing the one Spotify connection
// this MVP supports. needs_reauth is a flag rather than a row deletion on
// invalid_grant, because the frontend status contract distinguishes "never
// connected" from "connection needs reauthorization" — deleting the row
// would collapse those into the same state.
const schema = `
CREATE TABLE IF NOT EXISTS spotify_connection (
	id              INTEGER PRIMARY KEY CHECK (id = 1),
	access_token    TEXT NOT NULL,
	refresh_token   TEXT NOT NULL,
	token_type      TEXT NOT NULL,
	expires_at      INTEGER NOT NULL,
	spotify_user_id TEXT NOT NULL,
	display_name    TEXT NOT NULL DEFAULT '',
	needs_reauth    INTEGER NOT NULL DEFAULT 0,
	updated_at      INTEGER NOT NULL
);
`

// officialPlaylistSchema creates the single-row table backing the one
// official Sound Continuum Spotify playlist (Card #30). A separate table
// from spotify_connection, which is OAuth-connection-specific.
const officialPlaylistSchema = `
CREATE TABLE IF NOT EXISTS official_playlist (
	id                  INTEGER PRIMARY KEY CHECK (id = 1),
	spotify_playlist_id TEXT NOT NULL,
	name                TEXT NOT NULL,
	url                 TEXT NOT NULL,
	created_at          INTEGER NOT NULL
);
`

// Connection is the persisted Spotify token/identity state for the curator.
type Connection struct {
	AccessToken   string
	RefreshToken  string
	TokenType     string
	ExpiresAt     time.Time
	SpotifyUserID string
	DisplayName   string
	NeedsReauth   bool
}

// Store persists the single Spotify connection row in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates the spotify_connection and official_playlist tables if
// they don't exist yet.
func NewStore(db *sql.DB) (*Store, error) {
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}
	if _, err := db.Exec(officialPlaylistSchema); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// Get returns the stored connection, or nil if the curator has never
// connected.
func (s *Store) Get(ctx context.Context) (*Connection, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT access_token, refresh_token, token_type, expires_at,
		       spotify_user_id, display_name, needs_reauth
		FROM spotify_connection WHERE id = 1
	`)

	var c Connection
	var expiresAt int64
	var needsReauth int
	err := row.Scan(&c.AccessToken, &c.RefreshToken, &c.TokenType, &expiresAt,
		&c.SpotifyUserID, &c.DisplayName, &needsReauth)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.ExpiresAt = time.Unix(expiresAt, 0)
	c.NeedsReauth = needsReauth != 0
	return &c, nil
}

// Upsert saves a fresh or refreshed connection, clearing any prior
// needs_reauth flag — a successful token exchange/refresh means the
// connection is healthy again.
func (s *Store) Upsert(ctx context.Context, c Connection) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO spotify_connection
			(id, access_token, refresh_token, token_type, expires_at,
			 spotify_user_id, display_name, needs_reauth, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, 0, ?)
		ON CONFLICT(id) DO UPDATE SET
			access_token = excluded.access_token,
			refresh_token = excluded.refresh_token,
			token_type = excluded.token_type,
			expires_at = excluded.expires_at,
			spotify_user_id = excluded.spotify_user_id,
			display_name = excluded.display_name,
			needs_reauth = 0,
			updated_at = excluded.updated_at
	`, c.AccessToken, c.RefreshToken, c.TokenType, c.ExpiresAt.Unix(),
		c.SpotifyUserID, c.DisplayName, time.Now().Unix())
	return err
}

// MarkNeedsReauth flags the stored connection as requiring the curator to
// authorize again, without discarding the row (see the schema comment).
func (s *Store) MarkNeedsReauth(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE spotify_connection SET needs_reauth = 1, updated_at = ? WHERE id = 1
	`, time.Now().Unix())
	return err
}

// OfficialPlaylist is the persisted identity of the one official Sound
// Continuum Spotify playlist (Card #30), created once on Spotify and never
// recreated or edited by this application — see decisions.md.
type OfficialPlaylist struct {
	SpotifyPlaylistID string
	Name              string
	URL               string
	CreatedAt         time.Time
}

// GetOfficialPlaylist returns the persisted official playlist, or nil if
// it hasn't been created yet.
func (s *Store) GetOfficialPlaylist(ctx context.Context) (*OfficialPlaylist, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT spotify_playlist_id, name, url, created_at
		FROM official_playlist WHERE id = 1
	`)

	var p OfficialPlaylist
	var createdAt int64
	err := row.Scan(&p.SpotifyPlaylistID, &p.Name, &p.URL, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.CreatedAt = time.Unix(createdAt, 0)
	return &p, nil
}

// SaveOfficialPlaylist persists the official playlist row exactly once. A
// plain INSERT (no upsert) is deliberate: the row is never updated, and a
// second INSERT attempt fails on the id=1 PRIMARY KEY as a DB-level
// backstop against a duplicate — Service already guards against calling
// this twice; this is belt-and-suspenders.
func (s *Store) SaveOfficialPlaylist(ctx context.Context, p OfficialPlaylist) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO official_playlist (id, spotify_playlist_id, name, url, created_at)
		VALUES (1, ?, ?, ?, ?)
	`, p.SpotifyPlaylistID, p.Name, p.URL, time.Now().Unix())
	return err
}
