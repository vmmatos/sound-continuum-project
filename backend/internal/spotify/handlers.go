package spotify

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"
)

// expiryLeeway treats a token as due for refresh slightly before it
// actually expires, so a request doesn't race Spotify's clock.
const expiryLeeway = 30 * time.Second

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
	http.Redirect(w, r, s.client.AuthURL(s.cfg.RedirectURI, state), http.StatusFound)
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
	conn, err := s.store.Get(ctx)
	if err != nil {
		return "disconnected", "", err
	}
	if conn == nil {
		return "disconnected", "", nil
	}
	if conn.NeedsReauth {
		return "authorization_required", "", nil
	}

	if time.Now().Add(expiryLeeway).Before(conn.ExpiresAt) {
		return "connected", conn.DisplayName, nil
	}

	log.Print("Spotify token refresh required")
	token, err := s.client.Refresh(ctx, conn.RefreshToken)
	if errors.Is(err, ErrInvalidGrant) {
		log.Print("Spotify reauthorization required")
		if markErr := s.store.MarkNeedsReauth(ctx); markErr != nil {
			return "disconnected", "", markErr
		}
		return "authorization_required", "", nil
	}
	if err != nil {
		return "disconnected", "", err
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
		return "disconnected", "", err
	}

	log.Print("Spotify token refresh successful")
	return "connected", updated.DisplayName, nil
}
