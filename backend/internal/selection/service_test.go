package selection

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestMux wires the three selection routes the same way
// backend/cmd/server/main.go does, so r.PathValue("id") resolves correctly
// (net/http's ServeMux, not the handler, extracts path wildcards).
func newTestMux(t *testing.T) (*http.ServeMux, *Store) {
	t.Helper()
	store := newTestStore(t)
	svc := NewService(store)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/candidates/{id}/keep", svc.KeepHandler)
	mux.HandleFunc("POST /api/candidates/{id}/maybe", svc.MaybeHandler)
	mux.HandleFunc("POST /api/candidates/{id}/clear", svc.ClearHandler)
	return mux, store
}

func doAction(mux *http.ServeMux, action, id string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/candidates/"+id+"/"+action, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func decodeKeepResponse(t *testing.T, rec *httptest.ResponseRecorder) keepResponse {
	t.Helper()
	var resp keepResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func TestKeepHandlerReturnsSelected(t *testing.T) {
	mux, _ := newTestMux(t)

	rec := doAction(mux, "keep", "cand-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	resp := decodeKeepResponse(t, rec)
	if resp.CandidateID != "cand-1" || resp.Status != "selected" {
		t.Fatalf("response = %+v, want {cand-1 selected}", resp)
	}
}

func TestMaybeHandlerReturnsUnderReview(t *testing.T) {
	mux, _ := newTestMux(t)

	rec := doAction(mux, "maybe", "cand-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	resp := decodeKeepResponse(t, rec)
	if resp.CandidateID != "cand-1" || resp.Status != "under review" {
		t.Fatalf("response = %+v, want {cand-1 under review}", resp)
	}
}

func TestClearHandlerReturnsDiscovered(t *testing.T) {
	mux, _ := newTestMux(t)

	doAction(mux, "keep", "cand-1")
	rec := doAction(mux, "clear", "cand-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	resp := decodeKeepResponse(t, rec)
	if resp.CandidateID != "cand-1" || resp.Status != "discovered" {
		t.Fatalf("response = %+v, want {cand-1 discovered}", resp)
	}
}

func TestKeepThenMaybeAreMutuallyExclusive(t *testing.T) {
	mux, store := newTestMux(t)

	doAction(mux, "keep", "cand-1")
	doAction(mux, "maybe", "cand-1")

	selected, err := store.AllSelected(t.Context())
	if err != nil {
		t.Fatalf("AllSelected: %v", err)
	}
	if _, ok := selected["cand-1"]; ok {
		t.Fatalf("expected cand-1 to no longer be selected after Maybe, got %v", selected)
	}
	underReview, err := store.AllUnderReview(t.Context())
	if err != nil {
		t.Fatalf("AllUnderReview: %v", err)
	}
	if _, ok := underReview["cand-1"]; !ok {
		t.Fatalf("expected cand-1 to be under review, got %v", underReview)
	}
}

func TestMaybeThenKeepAreMutuallyExclusive(t *testing.T) {
	mux, store := newTestMux(t)

	doAction(mux, "maybe", "cand-1")
	doAction(mux, "keep", "cand-1")

	underReview, err := store.AllUnderReview(t.Context())
	if err != nil {
		t.Fatalf("AllUnderReview: %v", err)
	}
	if _, ok := underReview["cand-1"]; ok {
		t.Fatalf("expected cand-1 to no longer be under review after Keep, got %v", underReview)
	}
	selected, err := store.AllSelected(t.Context())
	if err != nil {
		t.Fatalf("AllSelected: %v", err)
	}
	if _, ok := selected["cand-1"]; !ok {
		t.Fatalf("expected cand-1 to be selected, got %v", selected)
	}
}

func TestRepeatingKeepIsIdempotentOverHTTP(t *testing.T) {
	mux, store := newTestMux(t)

	doAction(mux, "keep", "cand-1")
	doAction(mux, "keep", "cand-1")

	selected, err := store.AllSelected(t.Context())
	if err != nil {
		t.Fatalf("AllSelected: %v", err)
	}
	if len(selected) != 1 {
		t.Fatalf("expected exactly one selected candidate, got %v", selected)
	}
}

func TestRepeatingClearIsIdempotentOverHTTP(t *testing.T) {
	mux, _ := newTestMux(t)

	doAction(mux, "keep", "cand-1")
	rec1 := doAction(mux, "clear", "cand-1")
	rec2 := doAction(mux, "clear", "cand-1")

	if rec1.Code != http.StatusOK || rec2.Code != http.StatusOK {
		t.Fatalf("expected both Clear calls to succeed, got %d and %d", rec1.Code, rec2.Code)
	}
}

func TestActionsDoNotAffectUnrelatedCandidatesOverHTTP(t *testing.T) {
	mux, store := newTestMux(t)

	doAction(mux, "keep", "cand-1")
	doAction(mux, "maybe", "cand-2")

	selected, err := store.AllSelected(t.Context())
	if err != nil {
		t.Fatalf("AllSelected: %v", err)
	}
	if _, ok := selected["cand-2"]; ok {
		t.Fatalf("expected cand-2 to remain unselected, got %v", selected)
	}
	underReview, err := store.AllUnderReview(t.Context())
	if err != nil {
		t.Fatalf("AllUnderReview: %v", err)
	}
	if _, ok := underReview["cand-1"]; ok {
		t.Fatalf("expected cand-1 to remain not-under-review, got %v", underReview)
	}
}
