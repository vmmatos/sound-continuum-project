package spotify

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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
	err := svc.store.Upsert(newTestContext(), Connection{
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
	ctx := newTestContext()
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
	ctx := newTestContext()
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
	ctx := newTestContext()
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
	ctx := newTestContext()
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
	ctx := newTestContext()
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

	_, err := svc.Me(newTestContext())
	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("expected ErrNotConnected, got %v", err)
	}
}

func TestMeHandlerWritesProfileJSON(t *testing.T) {
	fake := &fakeSpotify{meStatus: http.StatusOK, meBody: map[string]any{"id": "u1", "account_id": "acct-1", "display_name": "Curator"}}
	server := fake.server()
	defer server.Close()

	svc := newTestService(t, server.URL)
	ctx := newTestContext()
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
	ctx := newTestContext()
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
	ctx := newTestContext()
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
	ctx := newTestContext()
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
	ctx := newTestContext()
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
	ctx := newTestContext()
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

func newTestContext() context.Context {
	return context.Background()
}
