package spotify

import (
	"context"
	"database/sql"
	"encoding/json"
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
}

func (f *fakeSpotify) server() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/token", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(f.tokenStatus)
		json.NewEncoder(w).Encode(f.tokenBody)
	})
	mux.HandleFunc("/v1/me", func(w http.ResponseWriter, r *http.Request) {
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
