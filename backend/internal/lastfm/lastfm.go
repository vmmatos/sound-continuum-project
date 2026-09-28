// Package lastfm is a small, focused client for the Last.fm REST API. It
// is used by internal/discovery only as an external discovery signal
// (artist.getsimilar) — Spotify remains the authoritative source for
// artist/track identity and catalogue data. See docs/memory/decisions.md.
package lastfm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// Sentinels for Last.fm API request failures. Use errors.Is against these,
// or errors.As(&apiErr) for Last.fm's own error code/message.
var (
	// ErrMissingAPIKey is returned by SimilarArtists without making a
	// request when the client has no API key configured — this is what
	// makes a missing LASTFM_API_KEY fail loudly rather than silently
	// returning an empty result.
	ErrMissingAPIKey = errors.New("lastfm: missing API key")

	// ErrRateLimited corresponds to Last.fm's documented error code 29
	// (rate limit exceeded).
	ErrRateLimited = errors.New("lastfm: rate limited")

	// ErrAPIFailure is any other Last.fm API error (invalid API key,
	// invalid parameters, artist not found, service unavailable, ...).
	ErrAPIFailure = errors.New("lastfm: api request failed")

	ErrTransport = errors.New("lastfm: request failed")
	ErrDecode    = errors.New("lastfm: malformed response")
)

// rateLimitErrorCode is Last.fm's documented error code for "rate limit
// exceeded".
const rateLimitErrorCode = 29

// APIError carries Last.fm's own error code and message (safe to surface —
// never a secret).
type APIError struct {
	Code    int
	Message string
	err     error // one of the sentinels above
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%v (code %d): %s", e.err, e.Code, e.Message)
}

func (e *APIError) Unwrap() error { return e.err }

// defaultUserAgent identifies this application to Last.fm, as their API
// guidelines recommend, instead of an empty/default Go User-Agent.
const defaultUserAgent = "SoundContinuum/0.1 (+https://github.com/vmmatos/sound-continuum-project)"

// SimilarArtist is one artist returned by artist.getsimilar. Match is
// Last.fm's own 0..1 similarity score, kept only as discovery provenance —
// never an editorial ranking signal (see docs/memory/decisions.md).
type SimilarArtist struct {
	Name  string
	Match float64
}

// Client talks to the Last.fm REST API. BaseURL is a field (not a
// constant) so tests can point it at an httptest.Server, matching
// internal/spotify.Client's own pattern.
type Client struct {
	APIKey     string
	BaseURL    string // default https://ws.audioscrobbler.com/2.0/
	UserAgent  string
	HTTPClient *http.Client
}

// NewClient builds a Client pointed at baseURL (the caller's configured
// Last.fm API root) with apiKey injected into every request.
func NewClient(apiKey, baseURL string) *Client {
	return &Client{
		APIKey:     apiKey,
		BaseURL:    baseURL,
		UserAgent:  defaultUserAgent,
		HTTPClient: http.DefaultClient,
	}
}

// similarArtistsResponse is Last.fm's own artist.getsimilar JSON shape,
// kept private to this package — internal/discovery never sees it
// directly, only the SimilarArtist provider model.
type similarArtistsResponse struct {
	SimilarArtists struct {
		Artist []struct {
			Name  string `json:"name"`
			Match string `json:"match"`
		} `json:"artist"`
	} `json:"similarartists"`
	Error   int    `json:"error"`
	Message string `json:"message"`
}

// SimilarArtists calls artist.getsimilar for artist, requesting up to
// limit similar artists (with autocorrect enabled, so minor misspellings
// still resolve). This is the only Last.fm operation this application
// uses, and it does not recurse — callers must never pass a name this
// method itself returned, to keep discovery to a single hop.
func (c *Client) SimilarArtists(ctx context.Context, artist string, limit int) ([]SimilarArtist, error) {
	if c.APIKey == "" {
		return nil, ErrMissingAPIKey
	}

	q := url.Values{
		"method":      {"artist.getsimilar"},
		"artist":      {artist},
		"api_key":     {c.APIKey},
		"format":      {"json"},
		"autocorrect": {"1"},
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, &APIError{err: ErrTransport, Message: err.Error()}
	}
	defer resp.Body.Close()

	var body similarArtistsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, &APIError{err: ErrDecode, Message: err.Error()}
	}

	if body.Error != 0 {
		if body.Error == rateLimitErrorCode {
			return nil, &APIError{Code: body.Error, Message: body.Message, err: ErrRateLimited}
		}
		return nil, &APIError{Code: body.Error, Message: body.Message, err: ErrAPIFailure}
	}

	artists := make([]SimilarArtist, 0, len(body.SimilarArtists.Artist))
	for _, a := range body.SimilarArtists.Artist {
		match, _ := strconv.ParseFloat(a.Match, 64) // best-effort; 0 on parse failure
		artists = append(artists, SimilarArtist{Name: a.Name, Match: match})
	}
	return artists, nil
}
