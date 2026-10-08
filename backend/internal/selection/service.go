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
	s.statusHandler(w, r, statusSelected, s.store.Keep)
}

// MaybeHandler exposes POST /api/candidates/{id}/maybe (Card #57) — marks a
// candidate under review. Same idempotency/no-existence-check contract as
// KeepHandler.
func (s *Service) MaybeHandler(w http.ResponseWriter, r *http.Request) {
	s.statusHandler(w, r, statusUnderReview, s.store.Maybe)
}

// RejectHandler exposes POST /api/candidates/{id}/skip (Card #58) — marks a
// candidate rejected. Named "skip" (the curator-facing action) rather than
// "reject" (the domain status), matching the existing /keep and /maybe
// endpoint naming — both are already UI-action names, not status names.
// Same idempotency/no-existence-check contract as KeepHandler/MaybeHandler.
func (s *Service) RejectHandler(w http.ResponseWriter, r *http.Request) {
	s.statusHandler(w, r, statusRejected, s.store.Reject)
}

// statusHandler is the shared body for KeepHandler/MaybeHandler: both set a
// fixed status via a single store call and report it back unchanged.
func (s *Service) statusHandler(w http.ResponseWriter, r *http.Request, status string, set func(context.Context, string) error) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "candidate id is required", http.StatusBadRequest)
		return
	}

	if err := set(r.Context(), id); err != nil {
		log.Printf("set candidate %q status %q failed: %v", id, status, err)
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keepResponse{CandidateID: id, Status: status})
}

// ClearHandler exposes POST /api/candidates/{id}/clear (Card #57) — removes
// any persisted Keep/Maybe/Skip decision, returning the candidate to the
// neutral "discovered" state. The curator's undo path for all three
// actions: the frontend decides when to call this based on which button is
// already active, so each endpoint stays a plain, idempotent "set" or
// "clear" operation.
func (s *Service) ClearHandler(w http.ResponseWriter, r *http.Request) {
	s.statusHandler(w, r, "discovered", s.store.Clear)
}
