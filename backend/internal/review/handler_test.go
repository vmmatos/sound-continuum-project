package review

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
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
