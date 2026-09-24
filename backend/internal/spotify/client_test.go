package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientAuthURL(t *testing.T) {
	c := NewClient("test-client-id", "test-client-secret")

	authURL := c.AuthURL("http://127.0.0.1:8080/api/spotify/callback", "the-state")

	for _, want := range []string{
		"client_id=test-client-id",
		"response_type=code",
		"state=the-state",
		"redirect_uri=",
	} {
		if !strings.Contains(authURL, want) {
			t.Errorf("auth URL missing %q: %s", want, authURL)
		}
	}
	if strings.Contains(authURL, "test-client-secret") {
		t.Fatal("auth URL must never contain the client secret")
	}
	if strings.Contains(authURL, "scope=") {
		t.Fatal("auth URL should not request any scope for this card")
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
