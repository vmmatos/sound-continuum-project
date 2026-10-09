package edition

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	return NewService(newTestStore(t))
}

func TestConfirmFromReviewRejectsEmptySnapshot(t *testing.T) {
	svc := newTestService(t)
	if _, err := svc.ConfirmFromReview(context.Background(), nil); err != ErrEmptySnapshot {
		t.Fatalf("expected ErrEmptySnapshot, got %v", err)
	}
}

func TestConfirmFromReviewCreatesDraftWhenNoneActive(t *testing.T) {
	svc := newTestService(t)
	e, err := svc.ConfirmFromReview(context.Background(), sampleTracks())
	if err != nil {
		t.Fatalf("ConfirmFromReview returned error: %v", err)
	}
	if e.Status != StatusConfirmed {
		t.Fatalf("expected StatusConfirmed, got %q", e.Status)
	}
	if len(e.ConfirmedTracks) != 3 {
		t.Fatalf("expected 3 confirmed tracks, got %d", len(e.ConfirmedTracks))
	}
}

func TestConfirmFromReviewReconfirmsExistingActiveEdition(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	first, _ := svc.ConfirmFromReview(ctx, sampleTracks())

	reordered := []ConfirmedTrack{sampleTracks()[1], sampleTracks()[0]}
	second, err := svc.ConfirmFromReview(ctx, reordered)
	if err != nil {
		t.Fatalf("second ConfirmFromReview returned error: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected the same Edition to be reconfirmed, got a different ID")
	}
	if len(second.ConfirmedTracks) != 2 {
		t.Fatalf("expected the snapshot to be overwritten, got %d tracks", len(second.ConfirmedTracks))
	}
}

func TestConfirmFromReviewLockedOncePublishing(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	e, _ := svc.ConfirmFromReview(ctx, sampleTracks())
	svc.store.StartPublishing(ctx, e.ID)

	if _, err := svc.ConfirmFromReview(ctx, sampleTracks()); err != ErrSnapshotLocked {
		t.Fatalf("expected ErrSnapshotLocked, got %v", err)
	}
}

func TestConfirmHandlerRoundTrip(t *testing.T) {
	svc := newTestService(t)

	body := confirmRequest{
		Tracks: []confirmTrackRequest{
			{
				SpotifyTrackID: "track-a",
				TrackTitle:     "A",
				TrackArtist:    "Artist A",
				Metadata:       &candidate.CandidateMetadata{Title: "A", SpotifyURL: "https://open.spotify.com/track/track-a"},
			},
			{
				SpotifyTrackID: "track-b",
				TrackTitle:     "B",
				TrackArtist:    "Artist B",
				Metadata:       nil, // enrichment failed upstream — handler must fall back, not reject
			},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/editions/confirm", bytes.NewReader(payload))
	rec := httptest.NewRecorder()
	svc.ConfirmHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp confirmResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.TrackCount != 2 {
		t.Fatalf("expected track_count 2, got %d", resp.TrackCount)
	}
	if resp.Status != string(StatusConfirmed) {
		t.Fatalf("expected status %q, got %q", StatusConfirmed, resp.Status)
	}

	fetched, err := svc.store.GetByID(context.Background(), resp.EditionID)
	if err != nil || fetched == nil {
		t.Fatalf("expected persisted edition, err=%v fetched=%v", err, fetched)
	}
	if fetched.ConfirmedTracks[1].Metadata.Title != "B" {
		t.Fatalf("expected fallback metadata built from TrackTitle for the unenriched track, got %+v", fetched.ConfirmedTracks[1].Metadata)
	}
}

func TestConfirmHandlerRejectsEmptyTracks(t *testing.T) {
	svc := newTestService(t)

	payload, _ := json.Marshal(confirmRequest{Tracks: nil})
	req := httptest.NewRequest(http.MethodPost, "/api/editions/confirm", bytes.NewReader(payload))
	rec := httptest.NewRecorder()
	svc.ConfirmHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestConfirmHandlerReturnsConflictWhenLocked(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	e, _ := svc.ConfirmFromReview(ctx, sampleTracks())
	svc.store.StartPublishing(ctx, e.ID)

	payload, _ := json.Marshal(confirmRequest{
		Tracks: []confirmTrackRequest{{SpotifyTrackID: "track-a", TrackTitle: "A", TrackArtist: "Artist A"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/editions/confirm", bytes.NewReader(payload))
	rec := httptest.NewRecorder()
	svc.ConfirmHandler(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestConfirmHandlerRejectsMalformedBody(t *testing.T) {
	svc := newTestService(t)

	req := httptest.NewRequest(http.MethodPost, "/api/editions/confirm", bytes.NewReader([]byte("not json")))
	rec := httptest.NewRecorder()
	svc.ConfirmHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
