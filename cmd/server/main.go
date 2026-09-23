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
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
