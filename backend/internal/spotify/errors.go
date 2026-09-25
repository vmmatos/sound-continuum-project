package spotify

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Sentinels for Spotify Web API request failures. Distinct from
// ErrInvalidGrant, which is specific to the token endpoint. Use errors.Is
// against these, or errors.As(&apiErr) for status code / Retry-After /
// Spotify's own message.
var (
	ErrUnauthorized = errors.New("spotify: unauthorized")
	ErrForbidden    = errors.New("spotify: forbidden")
	ErrNotFound     = errors.New("spotify: not found")
	ErrRateLimited  = errors.New("spotify: rate limited")
	ErrAPIFailure   = errors.New("spotify: api request failed")
	ErrTransport    = errors.New("spotify: request failed")
	ErrDecode       = errors.New("spotify: malformed response")

	// ErrSearchLimitTooHigh is returned by Client.Search without making a
	// request. Spotify's Development Mode caps search limit at 10 — this
	// project rejects an out-of-range limit rather than silently clamping it.
	ErrSearchLimitTooHigh = errors.New("spotify: search limit exceeds Spotify's maximum of 10")

	// ErrEmptyTrackID is returned by Client.Track without making a request —
	// avoids generating a malformed "/v1/tracks/" path.
	ErrEmptyTrackID = errors.New("spotify: track id must not be empty")

	// ErrEmptyArtistID is returned by Client.Artist without making a
	// request — avoids generating a malformed "/v1/artists/" path.
	ErrEmptyArtistID = errors.New("spotify: artist id must not be empty")
)

// APIError carries the HTTP status code, Spotify's own error message (safe
// to surface — never a token/secret), and Retry-After for 429 responses.
// Message is best-effort: an absent or unparsable body leaves it empty.
type APIError struct {
	StatusCode int
	RetryAfter time.Duration // set only when StatusCode == http.StatusTooManyRequests
	Message    string
	err        error // one of the sentinels above
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%v (status %d): %s", e.err, e.StatusCode, e.Message)
}

func (e *APIError) Unwrap() error { return e.err }

// newAPIError builds an APIError from a non-2xx response, consuming its body.
func newAPIError(resp *http.Response) *APIError {
	apiErr := &APIError{StatusCode: resp.StatusCode}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		apiErr.err = ErrUnauthorized
	case http.StatusForbidden:
		apiErr.err = ErrForbidden
	case http.StatusNotFound:
		apiErr.err = ErrNotFound
	case http.StatusTooManyRequests:
		apiErr.err = ErrRateLimited
		if seconds, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil {
			apiErr.RetryAfter = time.Duration(seconds) * time.Second
		}
	default:
		apiErr.err = ErrAPIFailure
	}

	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.NewDecoder(resp.Body).Decode(&body) == nil {
		apiErr.Message = body.Error.Message
	}
	return apiErr
}
