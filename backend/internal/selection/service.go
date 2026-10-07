package selection

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
)

// Service exposes candidate selection over HTTP. It holds no state of its
// own beyond the Store.
type Service struct {
	store *Store
}

// NewService wires a selection Service to an existing Store.
func NewService(store *Store) *Service {
	return &Service{store: store}
}

// Keep marks candidateID as selected.
func (s *Service) Keep(ctx context.Context, candidateID string) error {
	return s.store.Keep(ctx, candidateID)
}

// AllSelected returns the set of every candidate ID currently selected.
func (s *Service) AllSelected(ctx context.Context) (map[string]struct{}, error) {
	return s.store.AllSelected(ctx)
}

// keepResponse is KeepHandler's response body — a small handler-local type,
// not a mirror of a domain struct, so it gets JSON tags (matching package
// spotify's handler response convention) rather than the untagged-PascalCase
// convention candidate/scoring/discovery structs use.
type keepResponse struct {
	CandidateID string `json:"candidate_id"`
	Status      string `json:"status"`
}

// KeepHandler exposes POST /api/candidates/{id}/keep. Always idempotent:
// repeating the call for the same ID is safe and returns the same response.
// No existence check against a live candidate pool is performed — there is
// no persisted pool to check against (candidate pools are rebuilt per
// request, never stored) — matching this project's existing precedent of
// not inventing validation ahead of a concrete need (see decisions.md's
// Card #27/#28 deferred-parameter entries).
func (s *Service) KeepHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "candidate id is required", http.StatusBadRequest)
		return
	}

	if err := s.Keep(r.Context(), id); err != nil {
		log.Printf("keep candidate %q failed: %v", id, err)
		http.Error(w, "keep failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keepResponse{CandidateID: id, Status: statusSelected})
}
