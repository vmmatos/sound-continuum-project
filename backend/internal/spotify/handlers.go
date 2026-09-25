package spotify

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"
)

// expiryLeeway treats a token as due for refresh slightly before it
// actually expires, so a request doesn't race Spotify's clock.
const expiryLeeway = 30 * time.Second

// oauthScope is requested so the Spotify API client (Card #26) can read
// the curator's own playlists, and (Card #30) create/manage the official
// Sound Continuum playlist. GET /v1/me itself needs no scope.
const oauthScope = "user-read-private playlist-read-private playlist-modify-public"

// officialPlaylistName and officialPlaylistDescription are the fixed
// identity of the one official Sound Continuum Spotify playlist (Card
// #30) — see docs/manifesto.md and decisions.md.
const (
	officialPlaylistName        = "Sound Continuum — Weekly Journey"
	officialPlaylistDescription = "A weekly musical journey connecting timeless classics, current sounds and emerging artists through intentional musical bridges."
)

// ErrNotConnected indicates the curator has never connected Spotify.
var ErrNotConnected = errors.New("spotify: not connected")

// Service wires together Spotify OAuth configuration, the Spotify HTTP
// client, token storage, and OAuth state — and exposes the three
// application-level endpoints the frontend calls.
type Service struct {
	cfg            Config
	client         *Client
	store          *Store
	state          *stateGuard
	frontendOrigin string
}

// NewService creates the spotify_connection table (if needed) and returns a
// ready-to-use Service. cfg may be invalid (empty credentials) — that's
// checked per-request, not here, since dev/.secrets.env ships empty.
func NewService(db *sql.DB, cfg Config, frontendOrigin string) (*Service, error) {
	store, err := NewStore(db)
	if err != nil {
		return nil, err
	}
	return &Service{
		cfg:            cfg,
		client:         NewClient(cfg.ClientID, cfg.ClientSecret),
		store:          store,
		state:          &stateGuard{},
		frontendOrigin: frontendOrigin,
	}, nil
}

// AuthHandler starts the OAuth flow: generates a state, builds the Spotify
// authorization URL, and redirects the browser to it.
func (s *Service) AuthHandler(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.valid() {
		log.Print("Spotify authorization requested but Spotify is not configured")
		http.Error(w, "Spotify is not configured", http.StatusServiceUnavailable)
		return
	}

	state, err := s.state.generate()
	if err != nil {
		log.Printf("Spotify state generation failed: %v", err)
		http.Error(w, "failed to start Spotify authorization", http.StatusInternalServerError)
		return
	}

	log.Print("Spotify authorization started")
	http.Redirect(w, r, s.client.AuthURL(s.cfg.RedirectURI, state, oauthScope), http.StatusFound)
}

// CallbackHandler handles Spotify's redirect back after the curator
// approves or denies authorization.
func (s *Service) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("Spotify callback received")
	q := r.URL.Query()

	if q.Get("error") == "access_denied" {
		log.Print("Spotify authorization denied")
		s.redirectFrontend(w, r, "denied")
		return
	}

	if !s.state.consume(q.Get("state")) {
		log.Print("Spotify callback rejected: invalid or missing state")
		s.redirectFrontend(w, r, "error")
		return
	}

	code := q.Get("code")
	if code == "" {
		log.Print("Spotify callback rejected: missing authorization code")
		s.redirectFrontend(w, r, "error")
		return
	}

	ctx := r.Context()
	token, err := s.client.Exchange(ctx, code, s.cfg.RedirectURI)
	if err != nil {
		log.Printf("Spotify token exchange failed: %v", err)
		s.redirectFrontend(w, r, "error")
		return
	}

	var userID, displayName string
	if profile, err := s.client.Me(ctx, token.AccessToken); err != nil {
		log.Printf("Spotify profile lookup failed (connection still saved): %v", err)
	} else {
		userID = profile.UserID()
		displayName = profile.DisplayName
	}

	err = s.store.Upsert(ctx, Connection{
		AccessToken:   token.AccessToken,
		RefreshToken:  token.RefreshToken,
		TokenType:     token.TokenType,
		ExpiresAt:     time.Now().Add(token.ExpiresIn),
		SpotifyUserID: userID,
		DisplayName:   displayName,
	})
	if err != nil {
		log.Printf("Spotify connection save failed: %v", err)
		s.redirectFrontend(w, r, "error")
		return
	}

	log.Print("Spotify authorization successful")
	s.redirectFrontend(w, r, "connected")
}

func (s *Service) redirectFrontend(w http.ResponseWriter, r *http.Request, outcome string) {
	http.Redirect(w, r, s.frontendOrigin+"/?spotify="+outcome, http.StatusFound)
}

// statusResponse is the safe, application-level shape returned by
// StatusHandler — never a token, secret, or authorization code.
type statusResponse struct {
	Status      string `json:"status"`
	DisplayName string `json:"display_name,omitempty"`
}

// StatusHandler reports whether Spotify is connected, refreshing the access
// token first if it's expired.
func (s *Service) StatusHandler(w http.ResponseWriter, r *http.Request) {
	status, displayName, err := s.EnsureValidToken(r.Context())
	if err != nil {
		log.Printf("Spotify status check failed: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statusResponse{Status: status, DisplayName: displayName})
}

// EnsureValidToken returns the curator's current connection status,
// transparently refreshing the access token if it's expired or close to it.
// Refresh happens lazily, on read — there is no background worker, matching
// this project's "no queue, no background workers" rate-limit strategy.
func (s *Service) EnsureValidToken(ctx context.Context) (status string, displayName string, err error) {
	conn, err := s.connection(ctx)
	switch {
	case errors.Is(err, ErrNotConnected):
		return "disconnected", "", nil
	case errors.Is(err, ErrInvalidGrant):
		return "authorization_required", "", nil
	case err != nil:
		return "disconnected", "", err
	}
	return "connected", conn.DisplayName, nil
}

// connection returns a valid, live Connection, proactively refreshing the
// access token if it's within expiryLeeway of expiry. Returns
// ErrNotConnected if the curator never connected, or ErrInvalidGrant if
// the connection needs reauthorization (already flagged, or just flagged
// by a failed refresh here).
func (s *Service) connection(ctx context.Context) (Connection, error) {
	conn, err := s.store.Get(ctx)
	if err != nil {
		return Connection{}, err
	}
	if conn == nil {
		return Connection{}, ErrNotConnected
	}
	if conn.NeedsReauth {
		return Connection{}, ErrInvalidGrant
	}
	if time.Now().Add(expiryLeeway).Before(conn.ExpiresAt) {
		return *conn, nil
	}

	log.Print("Spotify token refresh required")
	updated, err := s.refresh(ctx, *conn)
	if err != nil {
		return Connection{}, err
	}
	log.Print("Spotify token refresh successful")
	return updated, nil
}

// refresh unconditionally calls Spotify's token endpoint with conn's
// refresh token and persists the result. On ErrInvalidGrant it marks the
// connection as needing reauthorization and returns ErrInvalidGrant —
// callers must not retry.
func (s *Service) refresh(ctx context.Context, conn Connection) (Connection, error) {
	token, err := s.client.Refresh(ctx, conn.RefreshToken)
	if errors.Is(err, ErrInvalidGrant) {
		log.Print("Spotify reauthorization required")
		if markErr := s.store.MarkNeedsReauth(ctx); markErr != nil {
			return Connection{}, markErr
		}
		return Connection{}, ErrInvalidGrant
	}
	if err != nil {
		return Connection{}, err
	}

	// Spotify may omit refresh_token on refresh, meaning the original one
	// stays valid — keep it in that case.
	newRefreshToken := token.RefreshToken
	if newRefreshToken == "" {
		newRefreshToken = conn.RefreshToken
	}

	updated := Connection{
		AccessToken:   token.AccessToken,
		RefreshToken:  newRefreshToken,
		TokenType:     token.TokenType,
		ExpiresAt:     time.Now().Add(token.ExpiresIn),
		SpotifyUserID: conn.SpotifyUserID,
		DisplayName:   conn.DisplayName,
	}
	if err := s.store.Upsert(ctx, updated); err != nil {
		return Connection{}, err
	}
	return updated, nil
}

// withToken calls fn with a live access token, obtained via connection()
// (which proactively refreshes near-expiry tokens). If fn fails with
// ErrUnauthorized — Spotify rejected a token that looked unexpired, e.g.
// clock skew or revocation — withToken forces exactly one refresh and
// retries fn exactly once. It never retries a second time.
func (s *Service) withToken(ctx context.Context, fn func(accessToken string) error) error {
	conn, err := s.connection(ctx)
	if err != nil {
		return err
	}

	err = fn(conn.AccessToken)
	if !errors.Is(err, ErrUnauthorized) {
		return err
	}

	log.Print("Spotify request unauthorized, forcing token refresh")
	conn, err = s.refresh(ctx, conn)
	if err != nil {
		return err
	}
	return fn(conn.AccessToken)
}

// withTokenResult calls fn with a live access token (via withToken) and
// returns its result, collapsing the "declare zero value, run withToken,
// assign inside the closure" boilerplate every Service passthrough below
// would otherwise repeat.
func withTokenResult[T any](ctx context.Context, s *Service, fn func(accessToken string) (T, error)) (T, error) {
	var result T
	err := s.withToken(ctx, func(accessToken string) error {
		var err error
		result, err = fn(accessToken)
		return err
	})
	return result, err
}

// Me returns the curator's Spotify profile.
func (s *Service) Me(ctx context.Context) (Profile, error) {
	return withTokenResult(ctx, s, func(accessToken string) (Profile, error) {
		return s.client.Me(ctx, accessToken)
	})
}

// Playlists returns a page of the curator's own Spotify playlists.
func (s *Service) Playlists(ctx context.Context, limit, offset int) (Paging[Playlist], error) {
	return withTokenResult(ctx, s, func(accessToken string) (Paging[Playlist], error) {
		return s.client.Playlists(ctx, accessToken, limit, offset)
	})
}

// Playlist returns metadata for a single playlist the curator can access.
func (s *Service) Playlist(ctx context.Context, playlistID string) (Playlist, error) {
	return withTokenResult(ctx, s, func(accessToken string) (Playlist, error) {
		return s.client.Playlist(ctx, accessToken, playlistID)
	})
}

// PlaylistItems returns a page of items from one of the curator's playlists.
func (s *Service) PlaylistItems(ctx context.Context, playlistID string, limit, offset int) (Paging[PlaylistItem], error) {
	return withTokenResult(ctx, s, func(accessToken string) (Paging[PlaylistItem], error) {
		return s.client.PlaylistItems(ctx, accessToken, playlistID, limit, offset)
	})
}

// Search queries the Spotify catalog on the curator's behalf.
func (s *Service) Search(ctx context.Context, query, types string, limit, offset int) (SearchResult, error) {
	return withTokenResult(ctx, s, func(accessToken string) (SearchResult, error) {
		return s.client.Search(ctx, accessToken, query, types, limit, offset)
	})
}

// Track returns metadata for a single track.
func (s *Service) Track(ctx context.Context, trackID string) (Track, error) {
	return withTokenResult(ctx, s, func(accessToken string) (Track, error) {
		return s.client.Track(ctx, accessToken, trackID)
	})
}

// Artist returns metadata for a single artist.
func (s *Service) Artist(ctx context.Context, artistID string) (Artist, error) {
	return withTokenResult(ctx, s, func(accessToken string) (Artist, error) {
		return s.client.Artist(ctx, accessToken, artistID)
	})
}

// InitializeOfficialPlaylist ensures the one official Sound Continuum
// Spotify playlist exists, creating it on Spotify at most once ever. If a
// playlist is already persisted locally, it's returned immediately with
// no Spotify call at all — no existence re-check, no name search. This is
// deliberate (see decisions.md): it's what guarantees a second call can
// never create a duplicate, and it means there is nothing to reconcile if
// Spotify were ever unable to confirm the playlist still exists.
func (s *Service) InitializeOfficialPlaylist(ctx context.Context) (OfficialPlaylist, error) {
	existing, err := s.store.GetOfficialPlaylist(ctx)
	if err != nil {
		return OfficialPlaylist{}, err
	}
	if existing != nil {
		return *existing, nil
	}

	var created Playlist
	err = s.withToken(ctx, func(accessToken string) error {
		var err error
		created, err = s.client.CreatePlaylist(ctx, accessToken, PlaylistCreateRequest{
			Name:        officialPlaylistName,
			Description: officialPlaylistDescription,
			Public:      true,
		})
		return err
	})
	if err != nil {
		return OfficialPlaylist{}, err
	}

	official := OfficialPlaylist{
		SpotifyPlaylistID: created.ID,
		Name:              created.Name,
		URL:               created.ExternalURLs.Spotify,
	}
	if err := s.store.SaveOfficialPlaylist(ctx, official); err != nil {
		// The playlist now exists on Spotify (id logged here — otherwise
		// unrecoverable) but local persistence failed. Do NOT retry
		// creation: retrying would risk a second Spotify playlist, since
		// the local row is what dedupes. Manual reconciliation (insert the
		// row by hand, or delete the stray Spotify playlist) is required;
		// see decisions.md — no distributed transaction.
		log.Printf("Spotify playlist %s (%s) created but not persisted locally: %v",
			created.ID, created.ExternalURLs.Spotify, err)
		return OfficialPlaylist{}, err
	}
	return official, nil
}

// officialPlaylistResponse is the safe, application-level shape returned by
// InitializePlaylistHandler.
type officialPlaylistResponse struct {
	SpotifyPlaylistID string `json:"spotify_playlist_id"`
	Name              string `json:"name"`
	URL               string `json:"url"`
}

// InitializePlaylistHandler exposes POST /api/spotify/playlist. Idempotent
// — repeated calls return the same persisted playlist, never creating a
// second one on Spotify.
func (s *Service) InitializePlaylistHandler(w http.ResponseWriter, r *http.Request) {
	official, err := s.InitializeOfficialPlaylist(r.Context())
	if err != nil {
		log.Printf("Spotify official playlist initialization failed: %v", err)
		writeSpotifyError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(officialPlaylistResponse{
		SpotifyPlaylistID: official.SpotifyPlaylistID,
		Name:              official.Name,
		URL:               official.URL,
	})
}

// MeHandler exposes GET /api/spotify/me.
func (s *Service) MeHandler(w http.ResponseWriter, r *http.Request) {
	profile, err := s.Me(r.Context())
	if err != nil {
		log.Printf("Spotify /me request failed: %v", err)
		writeSpotifyError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// PlaylistsHandler exposes GET /api/spotify/playlists?limit=&offset=.
func (s *Service) PlaylistsHandler(w http.ResponseWriter, r *http.Request) {
	page, err := s.Playlists(r.Context(), queryInt(r, "limit"), queryInt(r, "offset"))
	if err != nil {
		log.Printf("Spotify /playlists request failed: %v", err)
		writeSpotifyError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(page)
}

// PlaylistHandler exposes GET /api/spotify/playlists/{id}.
func (s *Service) PlaylistHandler(w http.ResponseWriter, r *http.Request) {
	playlist, err := s.Playlist(r.Context(), r.PathValue("id"))
	if err != nil {
		log.Printf("Spotify /playlists/{id} request failed: %v", err)
		writeSpotifyError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(playlist)
}

// PlaylistItemsHandler exposes
// GET /api/spotify/playlists/{id}/items?limit=&offset=.
func (s *Service) PlaylistItemsHandler(w http.ResponseWriter, r *http.Request) {
	page, err := s.PlaylistItems(r.Context(), r.PathValue("id"), queryInt(r, "limit"), queryInt(r, "offset"))
	if err != nil {
		log.Printf("Spotify /playlists/{id}/items request failed: %v", err)
		writeSpotifyError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(page)
}

// SearchHandler exposes GET /api/spotify/search?q=&type=&limit=&offset=.
func (s *Service) SearchHandler(w http.ResponseWriter, r *http.Request) {
	result, err := s.Search(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("type"),
		queryInt(r, "limit"), queryInt(r, "offset"))
	if errors.Is(err, ErrSearchLimitTooHigh) {
		http.Error(w, "limit must not exceed 10", http.StatusBadRequest)
		return
	}
	if err != nil {
		log.Printf("Spotify /search request failed: %v", err)
		writeSpotifyError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// TrackHandler exposes GET /api/spotify/tracks/{id}.
func (s *Service) TrackHandler(w http.ResponseWriter, r *http.Request) {
	track, err := s.Track(r.Context(), r.PathValue("id"))
	if errors.Is(err, ErrEmptyTrackID) {
		http.Error(w, "track id must not be empty", http.StatusBadRequest)
		return
	}
	if err != nil {
		log.Printf("Spotify /tracks/{id} request failed: %v", err)
		writeSpotifyError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(track)
}

// ArtistHandler exposes GET /api/spotify/artists/{id}.
func (s *Service) ArtistHandler(w http.ResponseWriter, r *http.Request) {
	artist, err := s.Artist(r.Context(), r.PathValue("id"))
	if errors.Is(err, ErrEmptyArtistID) {
		http.Error(w, "artist id must not be empty", http.StatusBadRequest)
		return
	}
	if err != nil {
		log.Printf("Spotify /artists/{id} request failed: %v", err)
		writeSpotifyError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(artist)
}

// queryInt parses a query parameter as an int, returning 0 if it's absent
// or invalid — the Client then applies its own default.
func queryInt(r *http.Request, name string) int {
	n, _ := strconv.Atoi(r.URL.Query().Get(name))
	return n
}

// writeSpotifyError maps a Service/Client error to an HTTP status. It never
// echoes tokens, secrets, or Authorization headers to the caller.
func writeSpotifyError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotConnected):
		http.Error(w, "Spotify is not connected", http.StatusServiceUnavailable)
	case errors.Is(err, ErrInvalidGrant):
		http.Error(w, "Spotify authorization required", http.StatusUnauthorized)
	case errors.Is(err, ErrRateLimited):
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.RetryAfter > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(int(apiErr.RetryAfter.Seconds())))
		}
		http.Error(w, "Spotify rate limit exceeded", http.StatusTooManyRequests)
	default:
		http.Error(w, "Spotify request failed", http.StatusBadGateway)
	}
}
