package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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

// AuthURL builds the Spotify authorization redirect URL. scope is a
// space-separated list of Spotify OAuth scopes (empty for none).
func (c *Client) AuthURL(redirectURI, state, scope string) string {
	q := url.Values{
		"client_id":     {c.ClientID},
		"response_type": {"code"},
		"redirect_uri":  {redirectURI},
		"state":         {state},
	}
	if scope != "" {
		q.Set("scope", scope)
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
	var p Profile
	err := c.request(ctx, http.MethodGet, "/v1/me", nil, nil, accessToken, &p)
	return p, err
}

// Playlists fetches a page of the curator's own Spotify playlists.
// limit <= 0 and offset < 0 fall back to Spotify's own defaults (20, 0).
func (c *Client) Playlists(ctx context.Context, accessToken string, limit, offset int) (Paging[Playlist], error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	query := url.Values{
		"limit":  {strconv.Itoa(limit)},
		"offset": {strconv.Itoa(offset)},
	}

	var page Paging[Playlist]
	err := c.request(ctx, http.MethodGet, "/v1/me/playlists", query, nil, accessToken, &page)
	return page, err
}

// Playlist fetches metadata for a single playlist by ID. Playlist name
// discovery (finding "the" Sound Continuum playlist) is application-level
// logic and deliberately does not live here — see decisions.md.
func (c *Client) Playlist(ctx context.Context, accessToken, playlistID string) (Playlist, error) {
	var p Playlist
	path := "/v1/playlists/" + url.PathEscape(playlistID)
	err := c.request(ctx, http.MethodGet, path, nil, nil, accessToken, &p)
	return p, err
}

// PlaylistItems fetches a page of items from a playlist via the current
// /items endpoint (the historical /tracks endpoint was removed).
// limit <= 0 and offset < 0 fall back to Spotify's own defaults (20, 0).
func (c *Client) PlaylistItems(ctx context.Context, accessToken, playlistID string, limit, offset int) (Paging[PlaylistItem], error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	query := url.Values{
		"limit":  {strconv.Itoa(limit)},
		"offset": {strconv.Itoa(offset)},
	}

	var page Paging[PlaylistItem]
	path := "/v1/playlists/" + url.PathEscape(playlistID) + "/items"
	err := c.request(ctx, http.MethodGet, path, query, nil, accessToken, &page)
	return page, err
}

// Search queries the Spotify catalog. limit must not exceed 10 — Spotify's
// Development Mode maximum — and is rejected rather than silently clamped.
// limit <= 0 falls back to Spotify's own default (5); offset < 0 falls
// back to 0.
func (c *Client) Search(ctx context.Context, accessToken, query, types string, limit, offset int) (SearchResult, error) {
	if limit > 10 {
		return SearchResult{}, ErrSearchLimitTooHigh
	}
	if limit <= 0 {
		limit = 5
	}
	if offset < 0 {
		offset = 0
	}
	q := url.Values{
		"q":      {query},
		"type":   {types},
		"limit":  {strconv.Itoa(limit)},
		"offset": {strconv.Itoa(offset)},
	}

	var result SearchResult
	err := c.request(ctx, http.MethodGet, "/v1/search", q, nil, accessToken, &result)
	return result, err
}

// Track fetches metadata for a single track by ID. No market parameter —
// this client always uses a user access token, and Spotify infers market
// from the authenticated user's account when market is omitted; see
// decisions.md.
func (c *Client) Track(ctx context.Context, accessToken, trackID string) (Track, error) {
	if trackID == "" {
		return Track{}, ErrEmptyTrackID
	}
	var t Track
	path := "/v1/tracks/" + url.PathEscape(trackID)
	err := c.request(ctx, http.MethodGet, path, nil, nil, accessToken, &t)
	return t, err
}

// Artist fetches metadata for a single artist by ID. There is no bulk
// equivalent — Spotify removed GET /artists?ids= for Development Mode —
// so a caller needing several artists must call this once per artist.
func (c *Client) Artist(ctx context.Context, accessToken, artistID string) (Artist, error) {
	if artistID == "" {
		return Artist{}, ErrEmptyArtistID
	}
	var a Artist
	path := "/v1/artists/" + url.PathEscape(artistID)
	err := c.request(ctx, http.MethodGet, path, nil, nil, accessToken, &a)
	return a, err
}

// request performs an authenticated Spotify Web API call and decodes a
// JSON response into out (nil to discard the body). method/body support
// POST/PUT/DELETE for a future write operation — every Card #26 operation
// is GET.
func (c *Client) request(ctx context.Context, method, path string, query url.Values, body io.Reader, accessToken string, out any) error {
	u := c.APIBaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return &APIError{err: ErrTransport, Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return newAPIError(resp)
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return &APIError{StatusCode: resp.StatusCode, err: ErrDecode, Message: err.Error()}
	}
	return nil
}
