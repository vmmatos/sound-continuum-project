package lastfm

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestSimilarArtistsMissingAPIKeyMakesNoRequest(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	_, err := c.SimilarArtists(context.Background(), "David Bowie", 10)
	if !errors.Is(err, ErrMissingAPIKey) {
		t.Fatalf("expected ErrMissingAPIKey, got %v", err)
	}
	if calls != 0 {
		t.Errorf("expected no request to be made, got %d", calls)
	}
}

func TestSimilarArtistsSendsExpectedRequest(t *testing.T) {
	var gotQuery url.Values
	var gotUserAgent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		gotUserAgent = r.Header.Get("User-Agent")
		w.Write([]byte(`{"similarartists":{"artist":[]}}`))
	}))
	defer srv.Close()

	c := NewClient("test-key", srv.URL)
	c.HTTPClient = srv.Client()
	_, err := c.SimilarArtists(context.Background(), "David Bowie", 10)
	if err != nil {
		t.Fatalf("SimilarArtists returned error: %v", err)
	}

	if gotQuery.Get("method") != "artist.getsimilar" {
		t.Errorf("expected method=artist.getsimilar, got %q", gotQuery.Get("method"))
	}
	if gotQuery.Get("artist") != "David Bowie" {
		t.Errorf("expected artist=David Bowie, got %q", gotQuery.Get("artist"))
	}
	if gotQuery.Get("api_key") != "test-key" {
		t.Errorf("expected api_key=test-key, got %q", gotQuery.Get("api_key"))
	}
	if gotQuery.Get("format") != "json" {
		t.Errorf("expected format=json, got %q", gotQuery.Get("format"))
	}
	if gotQuery.Get("limit") != "10" {
		t.Errorf("expected limit=10, got %q", gotQuery.Get("limit"))
	}
	if gotQuery.Get("autocorrect") != "1" {
		t.Errorf("expected autocorrect=1, got %q", gotQuery.Get("autocorrect"))
	}
	if gotUserAgent == "" || gotUserAgent == "Go-http-client/1.1" {
		t.Errorf("expected an identifiable User-Agent, got %q", gotUserAgent)
	}
}

func TestSimilarArtistsParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"similarartists":{"artist":[
			{"name":"Iggy Pop","match":"0.87"},
			{"name":"Lou Reed","match":"0.5"}
		]}}`))
	}))
	defer srv.Close()

	c := NewClient("test-key", srv.URL)
	c.HTTPClient = srv.Client()
	got, err := c.SimilarArtists(context.Background(), "David Bowie", 10)
	if err != nil {
		t.Fatalf("SimilarArtists returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 similar artists, got %d: %+v", len(got), got)
	}
	if got[0].Name != "Iggy Pop" || got[0].Match != 0.87 {
		t.Errorf("unexpected first artist: %+v", got[0])
	}
	if got[1].Name != "Lou Reed" || got[1].Match != 0.5 {
		t.Errorf("unexpected second artist: %+v", got[1])
	}
}

func TestSimilarArtistsRateLimitError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"error":29,"message":"Rate limit exceeded"}`))
	}))
	defer srv.Close()

	c := NewClient("test-key", srv.URL)
	c.HTTPClient = srv.Client()
	_, err := c.SimilarArtists(context.Background(), "David Bowie", 10)
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Message != "Rate limit exceeded" {
		t.Errorf("expected message surfaced, got %+v", err)
	}
}

func TestSimilarArtistsOtherAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"error":6,"message":"The artist you supplied could not be found"}`))
	}))
	defer srv.Close()

	c := NewClient("test-key", srv.URL)
	c.HTTPClient = srv.Client()
	_, err := c.SimilarArtists(context.Background(), "Nonexistent Artist", 10)
	if !errors.Is(err, ErrAPIFailure) {
		t.Fatalf("expected ErrAPIFailure, got %v", err)
	}
}

func TestSimilarArtistsTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // closed before use -> connection refused

	c := NewClient("test-key", srv.URL)
	_, err := c.SimilarArtists(context.Background(), "David Bowie", 10)
	if !errors.Is(err, ErrTransport) {
		t.Fatalf("expected ErrTransport, got %v", err)
	}
}

func TestSimilarArtistsDecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	c := NewClient("test-key", srv.URL)
	c.HTTPClient = srv.Client()
	_, err := c.SimilarArtists(context.Background(), "David Bowie", 10)
	if !errors.Is(err, ErrDecode) {
		t.Fatalf("expected ErrDecode, got %v", err)
	}
}
