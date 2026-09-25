// Command server runs the Sound Continuum HTTP backend.
package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "modernc.org/sqlite"

	"github.com/vmmatos/sound-continuum-project/internal/health"
	"github.com/vmmatos/sound-continuum-project/internal/spotify"
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
