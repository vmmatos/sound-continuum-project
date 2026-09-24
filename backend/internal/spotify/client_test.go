package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientAuthURL(t *testing.T) {
	c := NewClient("test-client-id", "test-client-secret")

	authURL := c.AuthURL("http://127.0.0.1:8080/api/spotify/callback", "the-state", "user-read-private playlist-read-private")

	for _, want := range []string{
		"client_id=test-client-id",
		"response_type=code",
		"state=the-state",
		"redirect_uri=",
		"scope=",
	} {
		if !strings.Contains(authURL, want) {
			t.Errorf("auth URL missing %q: %s", want, authURL)
		}
	}
	if strings.Contains(authURL, "test-client-secret") {
		t.Fatal("auth URL must never contain the client secret")
	}
}

func TestClientAuthURLNoScope(t *testing.T) {
	c := NewClient("test-client-id", "test-client-secret")

	authURL := c.AuthURL("http://127.0.0.1:8080/api/spotify/callback", "the-state", "")

	if strings.Contains(authURL, "scope=") {
		t.Fatal("auth URL should not include a scope parameter when scope is empty")
	}
}

func TestClientExchangeSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/token" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if user, pass, ok := r.BasicAuth(); !ok || user != "cid" || pass != "secret" {
			t.Error("expected client credentials via HTTP Basic Auth")
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if r.Form.Get("grant_type") != "authorization_code" {
			t.Errorf("expected grant_type=authorization_code, got %q", r.Form.Get("grant_type"))
		}
		if r.Form.Get("code") != "auth-code" {
			t.Errorf("expected code=auth-code, got %q", r.Form.Get("code"))
		}

		json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access-123",
			"refresh_token": "refresh-123",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	}))
	defer server.Close()

	c := NewClient("cid", "secret")
	c.AuthBaseURL = server.URL

	token, err := c.Exchange(context.Background(), "auth-code", "http://127.0.0.1:8080/api/spotify/callback")
	if err != nil {
		t.Fatalf("Exchange returned error: %v", err)
	}
	if token.AccessToken != "access-123" || token.RefreshToken != "refresh-123" {
		t.Errorf("unexpected token: %+v", token)
	}
}

func TestClientExchangeFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"error": "invalid_client"})
	}))
	defer server.Close()

	c := NewClient("cid", "secret")
	c.AuthBaseURL = server.URL

	_, err := c.Exchange(context.Background(), "bad-code", "http://127.0.0.1:8080/api/spotify/callback")
	if err == nil {
		t.Fatal("expected an error from a failed exchange")
	}
	if errors.Is(err, ErrInvalidGrant) {
		t.Fatal("invalid_client should not be reported as ErrInvalidGrant")
	}
}

func TestClientRefreshSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		if r.Form.Get("grant_type") != "refresh_token" {
			t.Errorf("expected grant_type=refresh_token, got %q", r.Form.Get("grant_type"))
		}
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "new-access",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	c := NewClient("cid", "secret")
	c.AuthBaseURL = server.URL

	token, err := c.Refresh(context.Background(), "old-refresh")
	if err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}
	if token.AccessToken != "new-access" {
		t.Errorf("unexpected token: %+v", token)
	}
}

func TestClientRefreshInvalidGrant(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"error": "invalid_grant"})
	}))
	defer server.Close()

	c := NewClient("cid", "secret")
	c.AuthBaseURL = server.URL

	_, err := c.Refresh(context.Background(), "revoked-refresh")
	if !errors.Is(err, ErrInvalidGrant) {
		t.Fatalf("expected ErrInvalidGrant, got %v", err)
	}
}

func TestClientMe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/me" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer access-123" {
			t.Errorf("unexpected Authorization header: %s", r.Header.Get("Authorization"))
		}
		json.NewEncoder(w).Encode(map[string]any{
			"id":           "legacy-id",
			"account_id":   "stable-account-id",
			"display_name": "The Curator",
		})
	}))
	defer server.Close()

	c := NewClient("cid", "secret")
	c.APIBaseURL = server.URL

	profile, err := c.Me(context.Background(), "access-123")
	if err != nil {
		t.Fatalf("Me returned error: %v", err)
	}
	if profile.DisplayName != "The Curator" {
		t.Errorf("unexpected display name: %q", profile.DisplayName)
	}
	if got := profile.UserID(); got != "stable-account-id" {
		t.Errorf("UserID() should prefer account_id, got %q", got)
	}
}

func TestClientPlaylistsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/me/playlists" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer access-123" {
			t.Errorf("unexpected Authorization header: %s", r.Header.Get("Authorization"))
		}
		if r.URL.Query().Get("limit") != "5" || r.URL.Query().Get("offset") != "10" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"items":  []map[string]any{{"id": "pl-1", "name": "Chapter One"}},
			"total":  1,
			"limit":  5,
			"offset": 10,
		})
	}))
	defer server.Close()

	c := NewClient("cid", "secret")
	c.APIBaseURL = server.URL

	page, err := c.Playlists(context.Background(), "access-123", 5, 10)
	if err != nil {
		t.Fatalf("Playlists returned error: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Name != "Chapter One" {
		t.Errorf("unexpected page: %+v", page)
	}
}

func TestClientPlaylistItemsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/playlists/pl-1/items" {
			t.Errorf("unexpected path (must use /items, not /tracks): %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"added_at": "2026-01-01T00:00:00Z", "item": map[string]any{"id": "t-1", "name": "Track One"}},
			},
			"total": 1, "limit": 20, "offset": 0,
		})
	}))
	defer server.Close()

	c := NewClient("cid", "secret")
	c.APIBaseURL = server.URL

	page, err := c.PlaylistItems(context.Background(), "access-123", "pl-1", 0, -1)
	if err != nil {
		t.Fatalf("PlaylistItems returned error: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Track.Name != "Track One" {
		t.Errorf("unexpected page: %+v", page)
	}
}

func TestClientSearchQueryEncoding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("q") != "boards of canada" || q.Get("type") != "track,artist" ||
			q.Get("limit") != "10" || q.Get("offset") != "3" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		json.NewEncoder(w).Encode(map[string]any{})
	}))
	defer server.Close()

	c := NewClient("cid", "secret")
	c.APIBaseURL = server.URL

	if _, err := c.Search(context.Background(), "access-123", "boards of canada", "track,artist", 10, 3); err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
}

func TestClientSearchRejectsLimitAboveTen(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Search must not make a request when limit exceeds 10")
	}))
	defer server.Close()

	c := NewClient("cid", "secret")
	c.APIBaseURL = server.URL

	_, err := c.Search(context.Background(), "access-123", "q", "track", 11, 0)
	if !errors.Is(err, ErrSearchLimitTooHigh) {
		t.Fatalf("expected ErrSearchLimitTooHigh, got %v", err)
	}
}

func TestClientRequestUnauthorized(t *testing.T) {
	c, server := newErrorClient(t, http.StatusUnauthorized, "")
	defer server.Close()

	_, err := c.Me(context.Background(), "access-123")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || !errors.Is(err, ErrUnauthorized) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected ErrUnauthorized APIError, got %v", err)
	}
}

func TestClientRequestForbidden(t *testing.T) {
	c, server := newErrorClient(t, http.StatusForbidden, "")
	defer server.Close()

	_, err := c.Me(context.Background(), "access-123")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestClientRequestNotFound(t *testing.T) {
	c, server := newErrorClient(t, http.StatusNotFound, "")
	defer server.Close()

	_, err := c.Me(context.Background(), "access-123")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestClientRequestRateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"message": "rate limited"}})
	}))
	defer server.Close()

	c := NewClient("cid", "secret")
	c.APIBaseURL = server.URL

	_, err := c.Me(context.Background(), "access-123")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited APIError, got %v", err)
	}
	if apiErr.RetryAfter != 5*time.Second {
		t.Errorf("expected RetryAfter of 5s, got %s", apiErr.RetryAfter)
	}
}

func TestClientRequestGenericServerError(t *testing.T) {
	c, server := newErrorClient(t, http.StatusInternalServerError, "")
	defer server.Close()

	_, err := c.Me(context.Background(), "access-123")
	if !errors.Is(err, ErrAPIFailure) {
		t.Fatalf("expected ErrAPIFailure, got %v", err)
	}
}

func TestClientRequestMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("{not valid json"))
	}))
	defer server.Close()

	c := NewClient("cid", "secret")
	c.APIBaseURL = server.URL

	_, err := c.Me(context.Background(), "access-123")
	if !errors.Is(err, ErrDecode) {
		t.Fatalf("expected ErrDecode, got %v", err)
	}
}

func newErrorClient(t *testing.T, status int, body string) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		if body != "" {
			w.Write([]byte(body))
		}
	}))
	c := NewClient("cid", "secret")
	c.APIBaseURL = server.URL
	return c, server
}
