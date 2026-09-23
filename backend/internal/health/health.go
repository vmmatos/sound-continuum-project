// Package health provides the service health-check HTTP handler.
package health

import (
	"encoding/json"
	"net/http"
)

// Handler responds with a JSON payload indicating the service is healthy.
func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
