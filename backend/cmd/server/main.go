// Command server runs the Sound Continuum HTTP backend.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/vmmatos/sound-continuum-project/internal/health"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health.Handler)

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
func withDevCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", devFrontendOrigin)
		h.ServeHTTP(w, r)
	})
}
