package edition

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
)

// Service exposes Edition confirmation over HTTP. It holds no state beyond
// the Store.
type Service struct {
	store *Store
}

// NewService wires an edition Service to an existing Store.
func NewService(store *Store) *Service {
	return &Service{store: store}
}

// ConfirmFromReview is the Card #61 integration boundary: it takes the
// curator's confirmed, ordered Kept tracks — exactly as the frontend
// already holds them, metadata included — and persists them as the active
// Edition's confirmed snapshot. It never re-derives order or metadata from
// a fresh candidate pool/rank/discovery call (Step 5/6).
//
//   - No active Edition yet: a Draft is created, then immediately confirmed
//     (there is no separate "start a new Edition" UI anywhere in this app
//     yet — see decisions.md).
//   - Active Edition in Draft or Confirmed: the snapshot is (re)written —
//     a deliberate curator action (Edit → reorder → reconfirm), never a
//     silent mutation by candidate review state, since Keep/Maybe/Skip
//     changes never call this path.
//   - Active Edition already Publishing/Published: ErrSnapshotLocked,
//     surfaced to the curator rather than silently ignored.
func (s *Service) ConfirmFromReview(ctx context.Context, tracks []ConfirmedTrack) (Edition, error) {
	if len(tracks) == 0 {
		return Edition{}, ErrEmptySnapshot
	}

	active, err := s.store.GetActive(ctx)
	if err != nil {
		return Edition{}, err
	}
	if active == nil {
		draft, err := s.store.CreateDraft(ctx)
		if err != nil {
			return Edition{}, err
		}
		active = &draft
	}

	return s.store.Confirm(ctx, active.ID, tracks)
}

// confirmTrackRequest mirrors ConfirmedTrack's/candidate.CandidateTrack's
// field names exactly (no JSON tags, PascalCase) rather than inventing a
// snake_case wire format — the frontend already holds and sends this exact
// shape (CandidateReviewEntry.Ranked.Candidate), so decoding it directly
// into the same field names needs no translation on either side. Metadata
// may be nil if a candidate's enrichment failed (a pre-existing Card #38
// possibility) — see metadataOrFallback.
type confirmTrackRequest struct {
	SpotifyTrackID string
	TrackTitle     string
	TrackArtist    string
	Metadata       *candidate.CandidateMetadata
}

type confirmRequest struct {
	Tracks []confirmTrackRequest
}

// confirmResponse is ConfirmHandler's response body — a small handler-local
// ack type with explicit snake_case JSON tags, matching selection's
// keepResponse convention (handler-local acks get tags; the request body
// above mirrors domain field names directly instead, since it's a direct
// copy of data the frontend already holds in that shape).
type confirmResponse struct {
	EditionID   string `json:"edition_id"`
	Status      string `json:"status"`
	TrackCount  int    `json:"track_count"`
	ConfirmedAt string `json:"confirmed_at"`
}

// ConfirmHandler exposes POST /api/editions/confirm — the only new HTTP
// route this card adds (no publish/archive endpoint; those stay Go-level
// Store methods with no caller until Cards #68/#69).
func (s *Service) ConfirmHandler(w http.ResponseWriter, r *http.Request) {
	var req confirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	tracks := make([]ConfirmedTrack, len(req.Tracks))
	for i, t := range req.Tracks {
		tracks[i] = ConfirmedTrack{
			SpotifyTrackID: t.SpotifyTrackID,
			Metadata:       metadataOrFallback(t),
		}
	}

	e, err := s.ConfirmFromReview(r.Context(), tracks)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmptySnapshot):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrSnapshotLocked):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			log.Printf("confirm edition failed: %v", err)
			http.Error(w, "confirm failed", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(confirmResponse{
		EditionID:   e.ID,
		Status:      string(e.Status),
		TrackCount:  len(e.ConfirmedTracks),
		ConfirmedAt: e.ConfirmedAt.Format(time.RFC3339),
	})
}

// metadataOrFallback returns t.Metadata dereferenced when present, or a
// CandidateMetadata built from the candidate's own always-present
// TrackTitle/TrackArtist when enrichment never ran for it — existing data
// carried over, never fabricated, so a confirmed snapshot still names every
// track even if one candidate's enrichment failed upstream (Card #38).
func metadataOrFallback(t confirmTrackRequest) candidate.CandidateMetadata {
	if t.Metadata != nil {
		return *t.Metadata
	}
	return candidate.CandidateMetadata{
		Title:   t.TrackTitle,
		Artists: []candidate.CandidateArtist{{Name: t.TrackArtist}},
	}
}
