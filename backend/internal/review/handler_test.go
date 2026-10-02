package review

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
	"github.com/vmmatos/sound-continuum-project/internal/discovery"
	"github.com/vmmatos/sound-continuum-project/internal/spotify"
)

// TestHandlerReturnsRealContract verifies GET /api/candidates/review returns
// candidate metadata, artwork, category, score, AvailableWeight,
// explanation, a null Bridge, and a deterministic Rank.
func TestHandlerReturnsRealContract(t *testing.T) {
	c := testCandidate(t, "c1", "track-1", "artist-1")
	f := fakeWithEligible([]candidate.CandidateTrack{c})
	svc := newTestService(f)

	rec := httptest.NewRecorder()
	svc.Handler(rec, httptest.NewRequest(http.MethodGet, "/api/candidates/review", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var pool ReviewPool
	if err := json.Unmarshal(rec.Body.Bytes(), &pool); err != nil {
		t.Fatalf("response body did not decode as ReviewPool: %v", err)
	}

	if len(pool.Entries) != 1 {
		t.Fatalf("len(Entries) = %d, want 1", len(pool.Entries))
	}
	entry := pool.Entries[0]

	if entry.Ranked.Rank != 1 {
		t.Errorf("Rank = %d, want 1", entry.Ranked.Rank)
	}
	if entry.Ranked.Candidate.TrackTitle != c.TrackTitle {
		t.Errorf("TrackTitle = %q, want %q", entry.Ranked.Candidate.TrackTitle, c.TrackTitle)
	}
	if entry.Ranked.Candidate.Category != candidate.CategoryEmerging {
		t.Errorf("Category = %q, want %q", entry.Ranked.Candidate.Category, candidate.CategoryEmerging)
	}
	if entry.Ranked.Candidate.Metadata == nil {
		t.Fatal("Metadata is nil, want enriched metadata")
	}
	if len(entry.Ranked.Candidate.Metadata.Album.Artwork) == 0 {
		t.Error("Album.Artwork is empty, want at least one image")
	}
	if entry.Ranked.Score.FinalScore == nil {
		t.Error("FinalScore is nil, want a computed value")
	}
	if entry.Ranked.Score.AvailableWeight == 0 {
		t.Error("AvailableWeight = 0, want Freshness's weight")
	}
	if entry.Explanation.Text == "" {
		t.Error("Explanation.Text is empty")
	}
	if entry.Bridge != nil {
		t.Errorf("Bridge = %v, want null", entry.Bridge)
	}
}

// TestHandlerJSONContractOmitsFailuresWhenClean confirms a clean run's
// response body has no WorkflowErrors/Failures noise — the JSON contract
// for the common case.
func TestHandlerJSONContractOmitsFailuresWhenClean(t *testing.T) {
	c := testCandidate(t, "c1", "track-1", "artist-1")
	f := fakeWithEligible([]candidate.CandidateTrack{c})
	svc := newTestService(f)

	rec := httptest.NewRecorder()
	svc.Handler(rec, httptest.NewRequest(http.MethodGet, "/api/candidates/review", nil))

	var pool ReviewPool
	if err := json.Unmarshal(rec.Body.Bytes(), &pool); err != nil {
		t.Fatalf("response body did not decode as ReviewPool: %v", err)
	}
	if len(pool.WorkflowErrors) != 0 || len(pool.Failures) != 0 {
		t.Errorf("WorkflowErrors=%v Failures=%v, want both empty", pool.WorkflowErrors, pool.Failures)
	}
}

// TestHandlerJSONContractSurfacesFailures confirms WorkflowErrors/Failures
// round-trip through the real HTTP handler's JSON encoding, with the exact
// field names the frontend contract (frontend/src/types/candidateReview.ts)
// depends on.
func TestHandlerJSONContractSurfacesFailures(t *testing.T) {
	f := &fakeDiscovery{
		pool: discovery.CandidatePool{
			WorkflowErrors: []discovery.WorkflowError{{Workflow: "emerging", Err: "lastfm: missing API key"}},
			ClassicResult: discovery.Result{
				Failures: []discovery.Failure{{Artist: "Some Artist", Stage: "albums", Err: "429 Too Many Requests"}},
			},
		},
	}
	svc := newTestService(f)

	rec := httptest.NewRecorder()
	svc.Handler(rec, httptest.NewRequest(http.MethodGet, "/api/candidates/review", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	for _, want := range []string{`"WorkflowErrors"`, `"Workflow":"emerging"`, `"Failures"`, `"Stage":"albums"`} {
		if !strings.Contains(body, want) {
			t.Errorf("response body missing %s: %s", want, body)
		}
	}

	var pool ReviewPool
	if err := json.Unmarshal(rec.Body.Bytes(), &pool); err != nil {
		t.Fatalf("response body did not decode as ReviewPool: %v", err)
	}
	if len(pool.Entries) != 0 {
		t.Errorf("len(Entries) = %d, want 0", len(pool.Entries))
	}
	if len(pool.WorkflowErrors) != 1 || pool.WorkflowErrors[0].Workflow != "emerging" {
		t.Errorf("WorkflowErrors = %v, want one emerging entry", pool.WorkflowErrors)
	}
	if len(pool.Failures) != 1 || pool.Failures[0].Stage != "albums" {
		t.Errorf("Failures = %v, want one albums-stage entry", pool.Failures)
	}
}

func TestHandlerReturnsServiceUnavailableWhenSpotifyNotConnected(t *testing.T) {
	f := &fakeDiscovery{filterErr: spotify.ErrNotConnected}
	svc := newTestService(f)

	rec := httptest.NewRecorder()
	svc.Handler(rec, httptest.NewRequest(http.MethodGet, "/api/candidates/review", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestHandlerReturnsBadGatewayOnGenericFailure(t *testing.T) {
	f := &fakeDiscovery{filterErr: errUnexpected}
	svc := newTestService(f)

	rec := httptest.NewRecorder()
	svc.Handler(rec, httptest.NewRequest(http.MethodGet, "/api/candidates/review", nil))

	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
}
