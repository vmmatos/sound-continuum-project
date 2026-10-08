// Command server runs the Sound Continuum HTTP backend.
package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"

	_ "modernc.org/sqlite"

	"github.com/vmmatos/sound-continuum-project/internal/discovery"
	"github.com/vmmatos/sound-continuum-project/internal/health"
	"github.com/vmmatos/sound-continuum-project/internal/lastfm"
	"github.com/vmmatos/sound-continuum-project/internal/review"
	"github.com/vmmatos/sound-continuum-project/internal/selection"
	"github.com/vmmatos/sound-continuum-project/internal/spotify"
	"github.com/vmmatos/sound-continuum-project/internal/spotifymock"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	sqlitePath := os.Getenv("SQLITE_PATH")
	if sqlitePath == "" {
		sqlitePath = "sound-continuum.db"
	}
	db, err := sql.Open("sqlite", sqlitePath)
	if err != nil {
		log.Fatalf("failed to open SQLite database at %s: %v", sqlitePath, err)
	}
	defer db.Close()

	spotifyService, err := spotify.NewService(db, spotify.Config{
		ClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		ClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
		RedirectURI:  os.Getenv("SPOTIFY_REDIRECT_URI"),
	}, devFrontendOrigin)
	if err != nil {
		log.Fatalf("failed to initialize Spotify service: %v", err)
	}

	lastfmClient := lastfm.NewClient(os.Getenv("LASTFM_API_KEY"), os.Getenv("LASTFM_API_URL"))

	recentTrackLookbackDays := discovery.DefaultRecentTrackLookbackDays
	if v := os.Getenv("RECENT_TRACK_LOOKBACK_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			recentTrackLookbackDays = n
		} else {
			log.Printf("ignoring invalid RECENT_TRACK_LOOKBACK_DAYS=%q, using default of %d days",
				v, recentTrackLookbackDays)
		}
	}

	// SPOTIFY_MOCK_MODE (Card #56) swaps discovery's Spotify dependency for a
	// deterministic, zero-network mock at the one seam every Spotify
	// touchpoint in the discovery pipeline already funnels through
	// (discovery.NewService's first parameter) — see
	// internal/spotifymock's doc comment and docs/memory/decisions.md.
	// Default false preserves real Spotify behavior exactly as before this
	// card.
	classicCfg, currentCfg, emergingCfg := discovery.DefaultConfig(), discovery.DefaultCurrentConfig(), discovery.DefaultEmergingConfig()
	var discoveryService *discovery.Service
	if os.Getenv("SPOTIFY_MOCK_MODE") == "true" {
		log.Println("SPOTIFY_MOCK_MODE enabled — using deterministic mock Spotify data, zero real Spotify API calls")
		discoveryService = discovery.NewService(spotifymock.NewCatalogue(), lastfmClient, classicCfg, currentCfg, emergingCfg, recentTrackLookbackDays)
	} else {
		discoveryService = discovery.NewService(spotifyService, lastfmClient, classicCfg, currentCfg, emergingCfg, recentTrackLookbackDays)
	}

	selectionStore, err := selection.NewStore(db)
	if err != nil {
		log.Fatalf("failed to initialize selection store: %v", err)
	}
	selectionService := selection.NewService(selectionStore)

	reviewService := review.NewService(discoveryService, selectionStore)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health.Handler)
	mux.HandleFunc("GET /api/spotify/auth", spotifyService.AuthHandler)
	mux.HandleFunc("GET /api/spotify/callback", spotifyService.CallbackHandler)
	mux.HandleFunc("GET /api/spotify/status", spotifyService.StatusHandler)
	mux.HandleFunc("GET /api/spotify/me", spotifyService.MeHandler)
	mux.HandleFunc("POST /api/spotify/playlist", spotifyService.InitializePlaylistHandler)
	mux.HandleFunc("GET /api/spotify/playlists", spotifyService.PlaylistsHandler)
	mux.HandleFunc("GET /api/spotify/playlists/{id}", spotifyService.PlaylistHandler)
	mux.HandleFunc("GET /api/spotify/playlists/{id}/items", spotifyService.PlaylistItemsHandler)
	mux.HandleFunc("GET /api/spotify/search", spotifyService.SearchHandler)
	mux.HandleFunc("GET /api/spotify/tracks/{id}", spotifyService.TrackHandler)
	mux.HandleFunc("GET /api/spotify/artists/{id}", spotifyService.ArtistHandler)
	mux.HandleFunc("POST /api/discovery/classic", discoveryService.ClassicHandler)
	mux.HandleFunc("POST /api/discovery/current", discoveryService.CurrentHandler)
	mux.HandleFunc("POST /api/discovery/emerging", discoveryService.EmergingHandler)
	mux.HandleFunc("POST /api/candidates/pool", discoveryService.PoolHandler)
	mux.HandleFunc("GET /api/candidates/review", reviewService.Handler)
	mux.HandleFunc("POST /api/candidates/{id}/keep", selectionService.KeepHandler)
	mux.HandleFunc("POST /api/candidates/{id}/maybe", selectionService.MaybeHandler)
	mux.HandleFunc("POST /api/candidates/{id}/skip", selectionService.RejectHandler)
	mux.HandleFunc("POST /api/candidates/{id}/clear", selectionService.ClearHandler)

	addr := ":" + port
	log.Printf("sound-continuum server listening on %s", addr)
	if err := http.ListenAndServe(addr, withDevCORS(mux)); err != nil {
		log.Fatal(err)
	}
}

// devFrontendOrigin is the Vite dev server origin, both when run directly
// on the host and via Docker Compose (the frontend container publishes
// the same port to the host).
const devFrontendOrigin = "http://localhost:5173"

// withDevCORS allows the local frontend dev server to call this API across
// origins. There is exactly one frontend origin in dev, so it's static.
// Every route was GET until Card #30 added a JSON POST — browsers preflight
// a non-simple request with OPTIONS, so that method is now answered here
// directly rather than reaching mux (which has no OPTIONS route).
func withDevCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", devFrontendOrigin)
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}
