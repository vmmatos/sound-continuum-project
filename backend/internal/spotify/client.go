package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrInvalidGrant indicates Spotify rejected a refresh token as expired or
// revoked (HTTP 400, error=invalid_grant). Callers must not retry — the
// curator has to authorize again.
var ErrInvalidGrant = errors.New("spotify: invalid_grant")

// Token is Spotify's token endpoint response, reduced to what this MVP
// tracks (access token, refresh token, type, expiry).
type Token struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    time.Duration
}

// Profile is the subset of GET /v1/me this MVP needs to identify the
// curator. AccountID is Spotify's stable, pseudoanonymous identifier
// (added May 2026); ID is the fallback for older/edge-case responses.
type Profile struct {
	ID          string `json:"id"`
	AccountID   string `json:"account_id"`
	DisplayName string `json:"display_name"`
}

// UserID prefers the stable account_id over id, per Spotify's own May 2026
// guidance for external linking.
func (p Profile) UserID() string {
	if p.AccountID != "" {
		return p.AccountID
	}
	return p.ID
}

// Client talks to Spotify's accounts/API hosts. AuthBaseURL and APIBaseURL
// are fields (not constants) so tests can point them at an httptest.Server.
type Client struct {
	ClientID     string
	ClientSecret string
	AuthBaseURL  string // default https://accounts.spotify.com
	APIBaseURL   string // default https://api.spotify.com
	HTTPClient   *http.Client
}

// NewClient builds a Client pointed at the real Spotify hosts.
func NewClient(clientID, clientSecret string) *Client {
	return &Client{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthBaseURL:  "https://accounts.spotify.com",
		APIBaseURL:   "https://api.spotify.com",
		HTTPClient:   http.DefaultClient,
	}
}

// AuthURL builds the Spotify authorization redirect URL. No scope is
// requested: GET /v1/me returns id/display_name without one, and this card
// only needs to establish the curator's identity, not act on their behalf.
func (c *Client) AuthURL(redirectURI, state string) string {
	q := url.Values{
		"client_id":     {c.ClientID},
		"response_type": {"code"},
		"redirect_uri":  {redirectURI},
		"state":         {state},
	}
	return c.AuthBaseURL + "/authorize?" + q.Encode()
}

// Exchange trades an authorization code for an access/refresh token pair.
func (c *Client) Exchange(ctx context.Context, code, redirectURI string) (Token, error) {
	return c.tokenRequest(ctx, url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {redirectURI},
	})
}

// Refresh obtains a new access token using a stored refresh token. Returns
// ErrInvalidGrant if Spotify reports the refresh token as expired/revoked.
func (c *Client) Refresh(ctx context.Context, refreshToken string) (Token, error) {
	return c.tokenRequest(ctx, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	})
}

func (c *Client) tokenRequest(ctx context.Context, form url.Values) (Token, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.AuthBaseURL+"/api/token",
		strings.NewReader(form.Encode()))
	if err != nil {
		return Token{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.ClientID, c.ClientSecret)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return Token{}, fmt.Errorf("spotify: token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		if errBody.Error == "invalid_grant" {
			return Token{}, ErrInvalidGrant
		}
		return Token{}, fmt.Errorf("spotify: token endpoint returned %d", resp.StatusCode)
	}

	var body struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return Token{}, fmt.Errorf("spotify: malformed token response: %w", err)
	}

	return Token{
		AccessToken:  body.AccessToken,
		RefreshToken: body.RefreshToken,
		TokenType:    body.TokenType,
		ExpiresIn:    time.Duration(body.ExpiresIn) * time.Second,
	}, nil
}

// Me fetches the curator's Spotify profile using a valid access token.
func (c *Client) Me(ctx context.Context, accessToken string) (Profile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.APIBaseURL+"/v1/me", nil)
	if err != nil {
		return Profile{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return Profile{}, fmt.Errorf("spotify: /v1/me request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Profile{}, fmt.Errorf("spotify: /v1/me returned %d", resp.StatusCode)
	}

	var p Profile
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return Profile{}, fmt.Errorf("spotify: malformed profile response: %w", err)
	}
	return p, nil
}
