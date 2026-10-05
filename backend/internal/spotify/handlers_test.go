package spotify

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

const testFrontendOrigin = "http://localhost:5173"

// fakeSpotify is a minimal stand-in for accounts.spotify.com + api.spotify.com,
// configurable per test so the handler tests never call the real Spotify API.
type fakeSpotify struct {
	tokenStatus int
	tokenBody   map[string]any
	meStatus    int
	meBody      map[string]any

	// meResponses, if set, overrides meStatus/meBody: one entry is consumed
	// per /v1/me call, sticking on the last entry once exhausted.
	meResponses []fakeResponse
	meCalls     int
	tokenCalls  int
}

type fakeResponse struct {
	status int
	body   map[string]any
}

func (f *fakeSpotify) server() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/token", func(w http.ResponseWriter, r *http.Request) {
		f.tokenCalls++
		w.WriteHeader(f.tokenStatus)
		json.NewEncoder(w).Encode(f.tokenBody)
	})
	mux.HandleFunc("/v1/me", func(w http.ResponseWriter, r *http.Request) {
		f.meCalls++
		if len(f.meResponses) > 0 {
			i := f.meCalls - 1
			if i >= len(f.meResponses) {
				i = len(f.meResponses) - 1
			}
			resp := f.meResponses[i]
			w.WriteHeader(resp.status)
			json.NewEncoder(w).Encode(resp.body)
			return
		}
		w.WriteHeader(f.meStatus)
		json.NewEncoder(w).Encode(f.meBody)
	})
	return httptest.NewServer(mux)
}

func newTestService(t *testing.T, spotifyServerURL string) *Service {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	svc, err := NewService(db, Config{
		ClientID:     "cid",
		ClientSecret: "secret",
		RedirectURI:  "http://127.0.0.1:8080/api/spotify/callback",
	}, testFrontendOrigin)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	if spotifyServerURL != "" {
		svc.client.AuthBaseURL = spotifyServerURL
		svc.client.APIBaseURL = spotifyServerURL
	}
	return svc
}

func TestAuthHandlerUnconfigured(t *testing.T) {
	svc := newTestService(t, "")
	svc.cfg = Config{}

	rec := httptest.NewRecorder()
	svc.AuthHandler(rec, httptest.NewRequest(http.MethodGet, "/api/spotify/auth", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestAuthHandlerRedirects(t *testing.T) {
	svc := newTestService(t, "")

	rec := httptest.NewRecorder()
	svc.AuthHandler(rec, httptest.NewRequest(http.MethodGet, "/api/spotify/auth", nil))

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	location := rec.Header().Get("Location")
	for _, want := range []string{"client_id=cid", "response_type=code", "state="} {
		if !strings.Contains(location, want) {
			t.Errorf("redirect location missing %q: %s", want, location)
		}
	}
	if strings.Contains(location, "secret") {
		t.Fatal("redirect location must never contain the client secret")
	}
}

func TestCallbackHandlerDenied(t *testing.T) {
	svc := newTestService(t, "")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/spotify/callback?error=access_denied", nil)
	svc.CallbackHandler(rec, req)

	assertRedirectTo(t, rec, testFrontendOrigin+"/?spotify=denied")
}

func TestCallbackHandlerInvalidState(t *testing.T) {
	svc := newTestService(t, "")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/spotify/callback?code=abc&state=never-issued", nil)
	svc.CallbackHandler(rec, req)

	assertRedirectTo(t, rec, testFrontendOrigin+"/?spotify=error")
}

func TestCallbackHandlerMissingCode(t *testing.T) {
	svc := newTestService(t, "")
	state, err := svc.state.generate()
	if err != nil {
		t.Fatalf("generate returned error: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/spotify/callback?state="+state, nil)
	svc.CallbackHandler(rec, req)

	assertRedirectTo(t, rec, testFrontendOrigin+"/?spotify=error")
}

func TestCallbackHandlerExchangeFailure(t *testing.T) {
	fake := &fakeSpotify{tokenStatus: http.StatusBadRequest, tokenBody: map[string]any{"error": "invalid_client"}}
	server := fake.server()
	defer server.Close()

	svc := newTestService(t, server.URL)
	state, err := svc.state.generate()
	if err != nil {
		t.Fatalf("generate returned error: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/spotify/callback?code=abc&state="+state, nil)
	svc.CallbackHandler(rec, req)

	assertRedirectTo(t, rec, testFrontendOrigin+"/?spotify=error")
}

func TestCallbackHandlerSuccess(t *testing.T) {
	fake := &fakeSpotify{
		tokenStatus: http.StatusOK,
		tokenBody: map[string]any{
			"access_token": "access-123", "refresh_token": "refresh-123",
			"token_type": "Bearer", "expires_in": 3600,
		},
		meStatus: http.StatusOK,
		meBody:   map[string]any{"id": "legacy", "account_id": "acct-1", "display_name": "The Curator"},
	}
	server := fake.server()
	defer server.Close()

	svc := newTestService(t, server.URL)
	state, err := svc.state.generate()
	if err != nil {
		t.Fatalf("generate returned error: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/spotify/callback?code=abc&state="+state, nil)
	svc.CallbackHandler(rec, req)

	assertRedirectTo(t, rec, testFrontendOrigin+"/?spotify=connected")

	conn, err := svc.store.Get(req.Context())
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if conn == nil || conn.AccessToken != "access-123" || conn.SpotifyUserID != "acct-1" {
		t.Fatalf("expected the connection to be stored, got %+v", conn)
	}
}

func TestStatusHandlerDisconnected(t *testing.T) {
	svc := newTestService(t, "")

	rec := httptest.NewRecorder()
	svc.StatusHandler(rec, httptest.NewRequest(http.MethodGet, "/api/spotify/status", nil))

	body := decodeStatus(t, rec)
	if body.Status != "disconnected" {
		t.Fatalf("expected disconnected, got %+v", body)
	}
}

func TestStatusHandlerConnected(t *testing.T) {
	svc := newTestService(t, "")
	err := svc.store.Upsert(context.Background(), Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	})
	if err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	rec := httptest.NewRecorder()
	svc.StatusHandler(rec, httptest.NewRequest(http.MethodGet, "/api/spotify/status", nil))

	body := decodeStatus(t, rec)
	if body.Status != "connected" || body.DisplayName != "Curator" {
		t.Fatalf("unexpected status response: %+v", body)
	}
}

func TestStatusHandlerAuthorizationRequired(t *testing.T) {
	svc := newTestService(t, "")
	ctx := context.Background()
	err := svc.store.Upsert(ctx, Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	})
	if err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}
	if err := svc.store.MarkNeedsReauth(ctx); err != nil {
		t.Fatalf("MarkNeedsReauth returned error: %v", err)
	}

	rec := httptest.NewRecorder()
	svc.StatusHandler(rec, httptest.NewRequest(http.MethodGet, "/api/spotify/status", nil))

	body := decodeStatus(t, rec)
	if body.Status != "authorization_required" {
		t.Fatalf("expected authorization_required, got %+v", body)
	}
}

func TestStatusHandlerRefreshesExpiredToken(t *testing.T) {
	fake := &fakeSpotify{
		tokenStatus: http.StatusOK,
		tokenBody:   map[string]any{"access_token": "fresh-access", "token_type": "Bearer", "expires_in": 3600},
	}
	server := fake.server()
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	err := svc.store.Upsert(ctx, Connection{
		AccessToken: "stale-access", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(-time.Minute), SpotifyUserID: "user-1", DisplayName: "Curator",
	})
	if err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	rec := httptest.NewRecorder()
	svc.StatusHandler(rec, httptest.NewRequest(http.MethodGet, "/api/spotify/status", nil))

	body := decodeStatus(t, rec)
	if body.Status != "connected" {
		t.Fatalf("expected connected after refresh, got %+v", body)
	}

	conn, err := svc.store.Get(ctx)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if conn.AccessToken != "fresh-access" {
		t.Fatalf("expected the refreshed access token to be persisted, got %+v", conn)
	}
}

func TestStatusHandlerInvalidGrantOnRefresh(t *testing.T) {
	fake := &fakeSpotify{tokenStatus: http.StatusBadRequest, tokenBody: map[string]any{"error": "invalid_grant"}}
	server := fake.server()
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	err := svc.store.Upsert(ctx, Connection{
		AccessToken: "stale-access", RefreshToken: "revoked-refresh", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(-time.Minute), SpotifyUserID: "user-1", DisplayName: "Curator",
	})
	if err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	rec := httptest.NewRecorder()
	svc.StatusHandler(rec, httptest.NewRequest(http.MethodGet, "/api/spotify/status", nil))

	body := decodeStatus(t, rec)
	if body.Status != "authorization_required" {
		t.Fatalf("expected authorization_required, got %+v", body)
	}

	conn, err := svc.store.Get(ctx)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if !conn.NeedsReauth {
		t.Fatal("expected the stored connection to be flagged needs_reauth")
	}
}

func TestServiceMeRefreshesOnUnauthorizedThenSucceeds(t *testing.T) {
	fake := &fakeSpotify{
		tokenStatus: http.StatusOK,
		tokenBody:   map[string]any{"access_token": "fresh-access", "token_type": "Bearer", "expires_in": 3600},
		meResponses: []fakeResponse{
			{status: http.StatusUnauthorized, body: map[string]any{}},
			{status: http.StatusOK, body: map[string]any{"id": "u1", "account_id": "acct-1", "display_name": "Curator"}},
		},
	}
	server := fake.server()
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	if err := svc.store.Upsert(ctx, Connection{
		AccessToken: "stale-access", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	profile, err := svc.Me(ctx)
	if err != nil {
		t.Fatalf("Me returned error: %v", err)
	}
	if profile.DisplayName != "Curator" {
		t.Errorf("unexpected profile: %+v", profile)
	}
	if fake.meCalls != 2 {
		t.Errorf("expected the original request to be retried exactly once (2 calls), got %d", fake.meCalls)
	}
	if fake.tokenCalls != 1 {
		t.Errorf("expected exactly one refresh, got %d", fake.tokenCalls)
	}
}

func TestServiceMeRefreshFailsWithInvalidGrantNoRetryLoop(t *testing.T) {
	fake := &fakeSpotify{
		tokenStatus: http.StatusBadRequest,
		tokenBody:   map[string]any{"error": "invalid_grant"},
		meResponses: []fakeResponse{
			{status: http.StatusUnauthorized, body: map[string]any{}},
		},
	}
	server := fake.server()
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	if err := svc.store.Upsert(ctx, Connection{
		AccessToken: "stale-access", RefreshToken: "revoked-refresh", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	_, err := svc.Me(ctx)
	if !errors.Is(err, ErrInvalidGrant) {
		t.Fatalf("expected ErrInvalidGrant, got %v", err)
	}
	if fake.meCalls != 1 {
		t.Errorf("expected no retry after a failed refresh, got %d /v1/me calls", fake.meCalls)
	}

	conn, err := svc.store.Get(ctx)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if !conn.NeedsReauth {
		t.Fatal("expected the stored connection to be flagged needs_reauth")
	}
}

func TestServiceMeReturnsErrNotConnectedWhenNeverConnected(t *testing.T) {
	svc := newTestService(t, "")

	_, err := svc.Me(context.Background())
	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("expected ErrNotConnected, got %v", err)
	}
}

func TestMeHandlerWritesProfileJSON(t *testing.T) {
	fake := &fakeSpotify{meStatus: http.StatusOK, meBody: map[string]any{"id": "u1", "account_id": "acct-1", "display_name": "Curator"}}
	server := fake.server()
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	if err := svc.store.Upsert(ctx, Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	rec := httptest.NewRecorder()
	svc.MeHandler(rec, httptest.NewRequest(http.MethodGet, "/api/spotify/me", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var profile Profile
	if err := json.NewDecoder(rec.Body).Decode(&profile); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if profile.DisplayName != "Curator" {
		t.Errorf("unexpected profile: %+v", profile)
	}
}

func TestMeHandlerReturnsServiceUnavailableWhenNotConnected(t *testing.T) {
	svc := newTestService(t, "")

	rec := httptest.NewRecorder()
	svc.MeHandler(rec, httptest.NewRequest(http.MethodGet, "/api/spotify/me", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestPlaylistsHandlerPassesQueryParams(t *testing.T) {
	var gotQuery string
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/me/playlists", func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		json.NewEncoder(w).Encode(map[string]any{"items": []any{}, "total": 0, "limit": 5, "offset": 10})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	if err := svc.store.Upsert(ctx, Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	rec := httptest.NewRecorder()
	svc.PlaylistsHandler(rec, httptest.NewRequest(http.MethodGet, "/api/spotify/playlists?limit=5&offset=10", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(gotQuery, "limit=5") || !strings.Contains(gotQuery, "offset=10") {
		t.Errorf("expected limit/offset to reach Spotify, got query %q", gotQuery)
	}
}

func TestPlaylistHandlerUsesPathID(t *testing.T) {
	var gotPath string
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/playlists/abc123", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewEncoder(w).Encode(map[string]any{"id": "abc123", "name": "Chapter One", "items": map[string]any{"total": 3}})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	if err := svc.store.Upsert(ctx, Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	mux2 := http.NewServeMux()
	mux2.HandleFunc("GET /api/spotify/playlists/{id}", svc.PlaylistHandler)

	rec := httptest.NewRecorder()
	mux2.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/spotify/playlists/abc123", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if gotPath != "/v1/playlists/abc123" {
		t.Errorf("expected the path ID to reach Spotify as /v1/playlists/abc123, got %q", gotPath)
	}
	var playlist Playlist
	if err := json.NewDecoder(rec.Body).Decode(&playlist); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if playlist.Name != "Chapter One" || playlist.Items.Total != 3 {
		t.Errorf("unexpected playlist: %+v", playlist)
	}
}

func TestPlaylistHandlerForbidden(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/playlists/abc123", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"message": "inaccessible"}})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	if err := svc.store.Upsert(ctx, Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	mux2 := http.NewServeMux()
	mux2.HandleFunc("GET /api/spotify/playlists/{id}", svc.PlaylistHandler)

	rec := httptest.NewRecorder()
	mux2.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/spotify/playlists/abc123", nil))

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected a Spotify-forbidden playlist to map to 502 (not 404), got %d", rec.Code)
	}
}

func TestPlaylistItemsHandlerUsesPathID(t *testing.T) {
	var gotPath string
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/playlists/abc123/items", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewEncoder(w).Encode(map[string]any{"items": []any{}, "total": 0, "limit": 20, "offset": 0})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	if err := svc.store.Upsert(ctx, Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	mux2 := http.NewServeMux()
	mux2.HandleFunc("GET /api/spotify/playlists/{id}/items", svc.PlaylistItemsHandler)

	rec := httptest.NewRecorder()
	mux2.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/spotify/playlists/abc123/items", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if gotPath != "/v1/playlists/abc123/items" {
		t.Errorf("expected the path ID to reach Spotify as /v1/playlists/abc123/items, got %q", gotPath)
	}
}

func TestSearchHandlerRejectsLimitAboveTen(t *testing.T) {
	svc := newTestService(t, "")
	ctx := context.Background()
	if err := svc.store.Upsert(ctx, Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	rec := httptest.NewRecorder()
	svc.SearchHandler(rec, httptest.NewRequest(http.MethodGet, "/api/spotify/search?q=x&type=track&limit=11", nil))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestTrackHandlerUsesPathID(t *testing.T) {
	var gotPath string
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/tracks/t-1", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewEncoder(w).Encode(map[string]any{"id": "t-1", "name": "Track One", "duration_ms": 210000})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	if err := svc.store.Upsert(ctx, Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	mux2 := http.NewServeMux()
	mux2.HandleFunc("GET /api/spotify/tracks/{id}", svc.TrackHandler)

	rec := httptest.NewRecorder()
	mux2.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/spotify/tracks/t-1", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if gotPath != "/v1/tracks/t-1" {
		t.Errorf("expected the path ID to reach Spotify as /v1/tracks/t-1, got %q", gotPath)
	}
	var track Track
	if err := json.NewDecoder(rec.Body).Decode(&track); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if track.Name != "Track One" || track.DurationMS != 210000 {
		t.Errorf("unexpected track: %+v", track)
	}
}

func TestTrackHandlerForbidden(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/tracks/t-1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"message": "inaccessible"}})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	if err := svc.store.Upsert(ctx, Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	mux2 := http.NewServeMux()
	mux2.HandleFunc("GET /api/spotify/tracks/{id}", svc.TrackHandler)

	rec := httptest.NewRecorder()
	mux2.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/spotify/tracks/t-1", nil))

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected a Spotify-forbidden track to map to 502 (not 403), got %d", rec.Code)
	}
}

func TestTrackHandlerEmptyID(t *testing.T) {
	svc := newTestService(t, "")
	ctx := context.Background()
	if err := svc.store.Upsert(ctx, Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/spotify/tracks/", nil)
	req.SetPathValue("id", "")

	rec := httptest.NewRecorder()
	svc.TrackHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestArtistHandlerUsesPathID(t *testing.T) {
	var gotPath string
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/artists/a-1", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewEncoder(w).Encode(map[string]any{"id": "a-1", "name": "Artist One"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	if err := svc.store.Upsert(ctx, Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	mux2 := http.NewServeMux()
	mux2.HandleFunc("GET /api/spotify/artists/{id}", svc.ArtistHandler)

	rec := httptest.NewRecorder()
	mux2.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/spotify/artists/a-1", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if gotPath != "/v1/artists/a-1" {
		t.Errorf("expected the path ID to reach Spotify as /v1/artists/a-1, got %q", gotPath)
	}
	var artist Artist
	if err := json.NewDecoder(rec.Body).Decode(&artist); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if artist.Name != "Artist One" {
		t.Errorf("unexpected artist: %+v", artist)
	}
}

func TestArtistHandlerForbidden(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/artists/a-1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"message": "inaccessible"}})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	if err := svc.store.Upsert(ctx, Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	mux2 := http.NewServeMux()
	mux2.HandleFunc("GET /api/spotify/artists/{id}", svc.ArtistHandler)

	rec := httptest.NewRecorder()
	mux2.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/spotify/artists/a-1", nil))

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected a Spotify-forbidden artist to map to 502 (not 403), got %d", rec.Code)
	}
}

func TestArtistHandlerEmptyID(t *testing.T) {
	svc := newTestService(t, "")
	ctx := context.Background()
	if err := svc.store.Upsert(ctx, Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/spotify/artists/", nil)
	req.SetPathValue("id", "")

	rec := httptest.NewRecorder()
	svc.ArtistHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestServiceArtistAlbumsPassesArgs(t *testing.T) {
	var gotPath, gotQuery string
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/artists/a-1/albums", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{{"id": "al-1", "name": "Album One"}}, "total": 1, "limit": 5, "offset": 0,
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	if err := svc.store.Upsert(ctx, Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	page, err := svc.ArtistAlbums(ctx, "a-1", 5, 0)
	if err != nil {
		t.Fatalf("ArtistAlbums returned error: %v", err)
	}
	if gotPath != "/v1/artists/a-1/albums" {
		t.Errorf("unexpected path: %q", gotPath)
	}
	if !strings.Contains(gotQuery, "limit=5") {
		t.Errorf("expected limit to reach Spotify, got query %q", gotQuery)
	}
	if len(page.Items) != 1 || page.Items[0].Name != "Album One" {
		t.Errorf("unexpected page: %+v", page)
	}
}

func TestServiceAlbumTracksPassesArgs(t *testing.T) {
	var gotPath string
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/albums/al-1/tracks", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{{"id": "t-1", "name": "Track One"}}, "total": 1, "limit": 10, "offset": 0,
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	if err := svc.store.Upsert(ctx, Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	page, err := svc.AlbumTracks(ctx, "al-1", 10, 0)
	if err != nil {
		t.Fatalf("AlbumTracks returned error: %v", err)
	}
	if gotPath != "/v1/albums/al-1/tracks" {
		t.Errorf("unexpected path: %q", gotPath)
	}
	if len(page.Items) != 1 || page.Items[0].Name != "Track One" {
		t.Errorf("unexpected page: %+v", page)
	}
}

func TestInitializeOfficialPlaylistCreatesOnFirstCall(t *testing.T) {
	var createCalls int
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/me", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"id": "user-1"})
	})
	mux.HandleFunc("/v1/me/playlists", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(map[string]any{"items": []any{}, "total": 0})
			return
		}
		createCalls++
		json.NewEncoder(w).Encode(map[string]any{
			"id": "pl-official", "name": "Sound Continuum — Weekly Journey",
			"external_urls": map[string]any{"spotify": "https://open.spotify.com/playlist/pl-official"},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	if err := svc.store.Upsert(ctx, Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	official, err := svc.InitializeOfficialPlaylist(ctx)
	if err != nil {
		t.Fatalf("InitializeOfficialPlaylist returned error: %v", err)
	}
	if official.SpotifyPlaylistID != "pl-official" || official.Name != "Sound Continuum — Weekly Journey" ||
		official.URL != "https://open.spotify.com/playlist/pl-official" {
		t.Errorf("unexpected official playlist: %+v", official)
	}
	if createCalls != 1 {
		t.Errorf("expected exactly one create call, got %d", createCalls)
	}

	stored, err := svc.store.GetOfficialPlaylist(ctx)
	if err != nil {
		t.Fatalf("GetOfficialPlaylist returned error: %v", err)
	}
	if stored == nil || stored.SpotifyPlaylistID != "pl-official" {
		t.Fatalf("expected the created playlist to be persisted, got %+v", stored)
	}
}

func TestInitializeOfficialPlaylistRequestBody(t *testing.T) {
	var gotBody map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/me", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"id": "user-1"})
	})
	mux.HandleFunc("/v1/me/playlists", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(map[string]any{"items": []any{}, "total": 0})
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		json.NewEncoder(w).Encode(map[string]any{
			"id": "pl-official", "name": "Sound Continuum — Weekly Journey",
			"external_urls": map[string]any{"spotify": "https://open.spotify.com/playlist/pl-official"},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	if err := svc.store.Upsert(ctx, Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	if _, err := svc.InitializeOfficialPlaylist(ctx); err != nil {
		t.Fatalf("InitializeOfficialPlaylist returned error: %v", err)
	}

	if gotBody["name"] != officialPlaylistName {
		t.Errorf("unexpected name: %v", gotBody["name"])
	}
	if gotBody["description"] != officialPlaylistDescription {
		t.Errorf("unexpected description: %v", gotBody["description"])
	}
	if gotBody["public"] != true {
		t.Errorf("expected public: true, got %v", gotBody["public"])
	}
	if v, ok := gotBody["collaborative"]; ok && v != false {
		t.Errorf("expected collaborative false or absent, got %v", v)
	}
}

func TestInitializeOfficialPlaylistIdempotentReturnsCachedWithoutCallingSpotify(t *testing.T) {
	var createCalls int
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/me/playlists", func(w http.ResponseWriter, r *http.Request) {
		createCalls++
		json.NewEncoder(w).Encode(map[string]any{"id": "should-not-be-created"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	if err := svc.store.Upsert(ctx, Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}
	if err := svc.store.SaveOfficialPlaylist(ctx, OfficialPlaylist{
		SpotifyPlaylistID: "pl-existing", Name: "Sound Continuum — Weekly Journey",
		URL: "https://open.spotify.com/playlist/pl-existing",
	}); err != nil {
		t.Fatalf("SaveOfficialPlaylist returned error: %v", err)
	}

	official, err := svc.InitializeOfficialPlaylist(ctx)
	if err != nil {
		t.Fatalf("InitializeOfficialPlaylist returned error: %v", err)
	}
	if official.SpotifyPlaylistID != "pl-existing" {
		t.Errorf("expected the cached playlist to be returned, got %+v", official)
	}
	if createCalls != 0 {
		t.Fatalf("expected no Spotify create call when a local playlist already exists, got %d", createCalls)
	}
}

func TestInitializeOfficialPlaylistCalledTwiceCreatesOnlyOnce(t *testing.T) {
	var createCalls int
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/me", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"id": "user-1"})
	})
	mux.HandleFunc("/v1/me/playlists", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(map[string]any{"items": []any{}, "total": 0})
			return
		}
		createCalls++
		json.NewEncoder(w).Encode(map[string]any{
			"id": "pl-official", "name": "Sound Continuum — Weekly Journey",
			"external_urls": map[string]any{"spotify": "https://open.spotify.com/playlist/pl-official"},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	if err := svc.store.Upsert(ctx, Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	first, err := svc.InitializeOfficialPlaylist(ctx)
	if err != nil {
		t.Fatalf("first InitializeOfficialPlaylist returned error: %v", err)
	}
	second, err := svc.InitializeOfficialPlaylist(ctx)
	if err != nil {
		t.Fatalf("second InitializeOfficialPlaylist returned error: %v", err)
	}
	if first.SpotifyPlaylistID != second.SpotifyPlaylistID {
		t.Errorf("expected both calls to return the same playlist, got %+v and %+v", first, second)
	}
	if createCalls != 1 {
		t.Fatalf("expected exactly one Spotify create call across two InitializeOfficialPlaylist calls, got %d", createCalls)
	}
}

func TestOfficialPlaylistReturnsPersistedPlaylist(t *testing.T) {
	svc := newTestService(t, "")
	ctx := context.Background()
	if err := svc.store.SaveOfficialPlaylist(ctx, OfficialPlaylist{
		SpotifyPlaylistID: "pl-existing", Name: "Sound Continuum — Weekly Journey",
		URL: "https://open.spotify.com/playlist/pl-existing",
	}); err != nil {
		t.Fatalf("SaveOfficialPlaylist returned error: %v", err)
	}

	playlist, err := svc.OfficialPlaylist(ctx)
	if err != nil {
		t.Fatalf("OfficialPlaylist returned error: %v", err)
	}
	if playlist.SpotifyPlaylistID != "pl-existing" {
		t.Errorf("OfficialPlaylist = %+v, want SpotifyPlaylistID pl-existing", playlist)
	}
}

func TestOfficialPlaylistNotConfigured(t *testing.T) {
	svc := newTestService(t, "")

	_, err := svc.OfficialPlaylist(context.Background())
	if !errors.Is(err, ErrOfficialPlaylistNotConfigured) {
		t.Fatalf("err = %v, want ErrOfficialPlaylistNotConfigured", err)
	}
}

func TestInitializeOfficialPlaylistNotConnected(t *testing.T) {
	svc := newTestService(t, "")

	_, err := svc.InitializeOfficialPlaylist(context.Background())
	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("expected ErrNotConnected, got %v", err)
	}
}

func TestInitializeOfficialPlaylistAuthorizationRequired(t *testing.T) {
	svc := newTestService(t, "")
	ctx := context.Background()
	if err := svc.store.Upsert(ctx, Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}
	if err := svc.store.MarkNeedsReauth(ctx); err != nil {
		t.Fatalf("MarkNeedsReauth returned error: %v", err)
	}

	_, err := svc.InitializeOfficialPlaylist(ctx)
	if !errors.Is(err, ErrInvalidGrant) {
		t.Fatalf("expected ErrInvalidGrant, got %v", err)
	}
}

func TestInitializeOfficialPlaylistSpotifyErrorNotPersisted(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/me", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"id": "user-1"})
	})
	mux.HandleFunc("/v1/me/playlists", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"message": "not allowed"}})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	if err := svc.store.Upsert(ctx, Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	_, err := svc.InitializeOfficialPlaylist(ctx)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}

	stored, err := svc.store.GetOfficialPlaylist(ctx)
	if err != nil {
		t.Fatalf("GetOfficialPlaylist returned error: %v", err)
	}
	if stored != nil {
		t.Fatalf("expected nothing persisted after a failed Spotify creation, got %+v", stored)
	}
}

func TestInitializePlaylistHandlerHTTP(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/me", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"id": "user-1"})
	})
	mux.HandleFunc("/v1/me/playlists", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id": "pl-official", "name": "Sound Continuum — Weekly Journey",
			"external_urls": map[string]any{"spotify": "https://open.spotify.com/playlist/pl-official"},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	if err := svc.store.Upsert(ctx, Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	mux2 := http.NewServeMux()
	mux2.HandleFunc("POST /api/spotify/playlist", svc.InitializePlaylistHandler)

	rec := httptest.NewRecorder()
	mux2.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/spotify/playlist", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var first officialPlaylistResponse
	if err := json.NewDecoder(rec.Body).Decode(&first); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if first.SpotifyPlaylistID != "pl-official" {
		t.Errorf("unexpected response: %+v", first)
	}

	rec2 := httptest.NewRecorder()
	mux2.ServeHTTP(rec2, httptest.NewRequest(http.MethodPost, "/api/spotify/playlist", nil))
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 on second call, got %d", rec2.Code)
	}
	var second officialPlaylistResponse
	if err := json.NewDecoder(rec2.Body).Decode(&second); err != nil {
		t.Fatalf("failed to decode second response: %v", err)
	}
	if second.SpotifyPlaylistID != first.SpotifyPlaylistID {
		t.Errorf("expected the same playlist across calls, got %+v and %+v", first, second)
	}
}

// upsertConnection is the Connection every InitializeOfficialPlaylist
// existing-playlist-search test below needs, with a fixed SpotifyUserID
// ("user-1") matched against Playlist.Owner.ID in test fixtures.
func upsertConnection(t *testing.T, svc *Service, ctx context.Context) {
	t.Helper()
	if err := svc.store.Upsert(ctx, Connection{
		AccessToken: "access-1", RefreshToken: "refresh-1", TokenType: "Bearer",
		ExpiresAt: time.Now().Add(time.Hour), SpotifyUserID: "user-1", DisplayName: "Curator",
	}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}
}

func TestInitializeOfficialPlaylistAdoptsSingleExistingMatch(t *testing.T) {
	var createCalls int
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/me", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"id": "user-1"})
	})
	mux.HandleFunc("/v1/me/playlists", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{
						"id": "pl-existing", "name": "Sound Continuum — Weekly Journey",
						"owner":          map[string]any{"id": "user-1"},
						"external_urls": map[string]any{"spotify": "https://open.spotify.com/playlist/pl-existing"},
					},
				},
				"total": 1,
			})
			return
		}
		createCalls++
		w.WriteHeader(http.StatusTeapot) // must never be reached
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	upsertConnection(t, svc, ctx)

	official, err := svc.InitializeOfficialPlaylist(ctx)
	if err != nil {
		t.Fatalf("InitializeOfficialPlaylist returned error: %v", err)
	}
	if official.SpotifyPlaylistID != "pl-existing" {
		t.Errorf("expected the existing Spotify playlist to be adopted, got %+v", official)
	}
	if createCalls != 0 {
		t.Fatalf("expected no create call when an existing playlist matches, got %d", createCalls)
	}

	stored, err := svc.store.GetOfficialPlaylist(ctx)
	if err != nil {
		t.Fatalf("GetOfficialPlaylist returned error: %v", err)
	}
	if stored == nil || stored.SpotifyPlaylistID != "pl-existing" {
		t.Fatalf("expected the adopted playlist to be persisted, got %+v", stored)
	}
}

func TestInitializeOfficialPlaylistAmbiguousMatchesFailClosed(t *testing.T) {
	var createCalls int
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/me", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"id": "user-1"})
	})
	mux.HandleFunc("/v1/me/playlists", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"id": "pl-dup-1", "name": "Sound Continuum — Weekly Journey", "owner": map[string]any{"id": "user-1"}},
					{"id": "pl-dup-2", "name": "Sound Continuum — Weekly Journey", "owner": map[string]any{"id": "user-1"}},
				},
				"total": 2,
			})
			return
		}
		createCalls++
		w.WriteHeader(http.StatusTeapot) // must never be reached
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	upsertConnection(t, svc, ctx)

	_, err := svc.InitializeOfficialPlaylist(ctx)
	var ambiguous *AmbiguousOfficialPlaylistError
	if !errors.As(err, &ambiguous) {
		t.Fatalf("expected *AmbiguousOfficialPlaylistError, got %v", err)
	}
	if len(ambiguous.Matches) != 2 {
		t.Errorf("expected 2 matches, got %d: %+v", len(ambiguous.Matches), ambiguous.Matches)
	}
	if createCalls != 0 {
		t.Fatalf("expected no create call when matches are ambiguous, got %d", createCalls)
	}
	if stored, err := svc.store.GetOfficialPlaylist(ctx); err != nil || stored != nil {
		t.Fatalf("expected nothing persisted when ambiguous, got stored=%+v err=%v", stored, err)
	}
}

func TestInitializeOfficialPlaylistSearchWalksAllPages(t *testing.T) {
	var createCalls, listCalls int
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/me", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"id": "user-1"})
	})
	mux.HandleFunc("/v1/me/playlists", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			listCalls++
			offset := r.URL.Query().Get("offset")
			if offset == "0" {
				// A full page of unrelated playlists — forces a second page.
				items := make([]map[string]any, 50)
				for i := range items {
					items[i] = map[string]any{"id": fmt.Sprintf("other-%d", i), "name": "Unrelated", "owner": map[string]any{"id": "user-1"}}
				}
				json.NewEncoder(w).Encode(map[string]any{"items": items, "total": 51})
				return
			}
			json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{
						"id": "pl-on-page-2", "name": "Sound Continuum — Weekly Journey",
						"owner":          map[string]any{"id": "user-1"},
						"external_urls": map[string]any{"spotify": "https://open.spotify.com/playlist/pl-on-page-2"},
					},
				},
				"total": 51,
			})
			return
		}
		createCalls++
		w.WriteHeader(http.StatusTeapot)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	upsertConnection(t, svc, ctx)

	official, err := svc.InitializeOfficialPlaylist(ctx)
	if err != nil {
		t.Fatalf("InitializeOfficialPlaylist returned error: %v", err)
	}
	if official.SpotifyPlaylistID != "pl-on-page-2" {
		t.Errorf("expected the match on page 2 to be found, got %+v", official)
	}
	if listCalls != 2 {
		t.Fatalf("expected the search to walk 2 pages, got %d", listCalls)
	}
	if createCalls != 0 {
		t.Fatalf("expected no create call once page 2's match was found, got %d", createCalls)
	}
}

func TestInitializeOfficialPlaylistNearNameDoesNotMatch(t *testing.T) {
	var createCalls int
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/me", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"id": "user-1"})
	})
	mux.HandleFunc("/v1/me/playlists", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"id": "pl-similar", "name": "Sound Continuum - Weekly Journey (old)", "owner": map[string]any{"id": "user-1"}},
				},
				"total": 1,
			})
			return
		}
		createCalls++
		json.NewEncoder(w).Encode(map[string]any{
			"id": "pl-official", "name": "Sound Continuum — Weekly Journey",
			"external_urls": map[string]any{"spotify": "https://open.spotify.com/playlist/pl-official"},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	upsertConnection(t, svc, ctx)

	official, err := svc.InitializeOfficialPlaylist(ctx)
	if err != nil {
		t.Fatalf("InitializeOfficialPlaylist returned error: %v", err)
	}
	if official.SpotifyPlaylistID != "pl-official" {
		t.Errorf("expected a new playlist to be created since no exact match exists, got %+v", official)
	}
	if createCalls != 1 {
		t.Fatalf("expected exactly one create call, got %d", createCalls)
	}
}

func TestInitializeOfficialPlaylistOtherOwnerDoesNotMatch(t *testing.T) {
	var createCalls int
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/me", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"id": "user-1"})
	})
	mux.HandleFunc("/v1/me/playlists", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"id": "pl-someone-elses", "name": "Sound Continuum — Weekly Journey", "owner": map[string]any{"id": "someone-else"}},
				},
				"total": 1,
			})
			return
		}
		createCalls++
		json.NewEncoder(w).Encode(map[string]any{
			"id": "pl-official", "name": "Sound Continuum — Weekly Journey",
			"external_urls": map[string]any{"spotify": "https://open.spotify.com/playlist/pl-official"},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := context.Background()
	upsertConnection(t, svc, ctx)

	official, err := svc.InitializeOfficialPlaylist(ctx)
	if err != nil {
		t.Fatalf("InitializeOfficialPlaylist returned error: %v", err)
	}
	if official.SpotifyPlaylistID != "pl-official" {
		t.Errorf("expected a new playlist to be created since the name match is owned by someone else, got %+v", official)
	}
	if createCalls != 1 {
		t.Fatalf("expected exactly one create call, got %d", createCalls)
	}
}

func assertRedirectTo(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != want {
		t.Fatalf("expected redirect to %q, got %q", want, got)
	}
}

func decodeStatus(t *testing.T, rec *httptest.ResponseRecorder) statusResponse {
	t.Helper()
	var body statusResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode status response: %v", err)
	}
	return body
}
