package spotify

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	// modernc.org/sqlite's :memory: database is per-connection; a single
	// connection keeps the whole test on the same in-memory database.
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore returned error: %v", err)
	}
	return store
}

func TestStoreGetWithNoConnection(t *testing.T) {
	store := newTestStore(t)

	conn, err := store.Get(context.Background())
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if conn != nil {
		t.Fatalf("expected no connection, got %+v", conn)
	}
}

func TestStoreUpsertAndGet(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	expiresAt := time.Now().Add(time.Hour).Truncate(time.Second)
	err := store.Upsert(ctx, Connection{
		AccessToken:   "access-1",
		RefreshToken:  "refresh-1",
		TokenType:     "Bearer",
		ExpiresAt:     expiresAt,
		SpotifyUserID: "user-1",
		DisplayName:   "Curator",
	})
	if err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	conn, err := store.Get(ctx)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected a stored connection")
	}
	if conn.AccessToken != "access-1" || conn.RefreshToken != "refresh-1" ||
		conn.SpotifyUserID != "user-1" || conn.DisplayName != "Curator" {
		t.Errorf("unexpected connection: %+v", conn)
	}
	if !conn.ExpiresAt.Equal(expiresAt) {
		t.Errorf("expected expires_at %v, got %v", expiresAt, conn.ExpiresAt)
	}
	if conn.NeedsReauth {
		t.Error("freshly upserted connection should not need reauth")
	}
}

func TestStoreUpsertReplacesExistingRow(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	base := Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}
	if err := store.Upsert(ctx, base); err != nil {
		t.Fatalf("first Upsert returned error: %v", err)
	}

	base.AccessToken = "access-2"
	if err := store.Upsert(ctx, base); err != nil {
		t.Fatalf("second Upsert returned error: %v", err)
	}

	conn, err := store.Get(ctx)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if conn.AccessToken != "access-2" {
		t.Errorf("expected the row to be replaced, got %+v", conn)
	}
}

func TestStoreMarkNeedsReauth(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	err := store.Upsert(ctx, Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	})
	if err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	if err := store.MarkNeedsReauth(ctx); err != nil {
		t.Fatalf("MarkNeedsReauth returned error: %v", err)
	}

	conn, err := store.Get(ctx)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if !conn.NeedsReauth {
		t.Fatal("expected NeedsReauth to be true")
	}
	if conn.RefreshToken != "refresh-1" {
		t.Fatal("MarkNeedsReauth must not discard the stored tokens")
	}
}

func TestStoreUpsertClearsNeedsReauth(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	conn := Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}
	if err := store.Upsert(ctx, conn); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}
	if err := store.MarkNeedsReauth(ctx); err != nil {
		t.Fatalf("MarkNeedsReauth returned error: %v", err)
	}

	if err := store.Upsert(ctx, conn); err != nil {
		t.Fatalf("second Upsert returned error: %v", err)
	}

	got, err := store.Get(ctx)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.NeedsReauth {
		t.Fatal("a successful Upsert should clear needs_reauth")
	}
}

func TestStoreGetOfficialPlaylistWithNone(t *testing.T) {
	store := newTestStore(t)

	p, err := store.GetOfficialPlaylist(context.Background())
	if err != nil {
		t.Fatalf("GetOfficialPlaylist returned error: %v", err)
	}
	if p != nil {
		t.Fatalf("expected no official playlist, got %+v", p)
	}
}

func TestStoreSaveAndGetOfficialPlaylist(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	err := store.SaveOfficialPlaylist(ctx, OfficialPlaylist{
		SpotifyPlaylistID: "pl-official",
		Name:              "Sound Continuum — Weekly Journey",
		URL:               "https://open.spotify.com/playlist/pl-official",
	})
	if err != nil {
		t.Fatalf("SaveOfficialPlaylist returned error: %v", err)
	}

	p, err := store.GetOfficialPlaylist(ctx)
	if err != nil {
		t.Fatalf("GetOfficialPlaylist returned error: %v", err)
	}
	if p == nil {
		t.Fatal("expected a stored official playlist")
	}
	if p.SpotifyPlaylistID != "pl-official" || p.Name != "Sound Continuum — Weekly Journey" ||
		p.URL != "https://open.spotify.com/playlist/pl-official" {
		t.Errorf("unexpected official playlist: %+v", p)
	}
	if p.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestStoreSaveOfficialPlaylistTwiceFails(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	p := OfficialPlaylist{SpotifyPlaylistID: "pl-official", Name: "Sound Continuum — Weekly Journey", URL: "https://open.spotify.com/playlist/pl-official"}
	if err := store.SaveOfficialPlaylist(ctx, p); err != nil {
		t.Fatalf("first SaveOfficialPlaylist returned error: %v", err)
	}

	if err := store.SaveOfficialPlaylist(ctx, p); err == nil {
		t.Fatal("expected a second SaveOfficialPlaylist to fail on the id=1 primary key")
	}
}
