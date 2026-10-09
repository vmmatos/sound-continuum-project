package edition

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/vmmatos/sound-continuum-project/internal/candidate"
)

// newTestStore follows the exact :memory: + SetMaxOpenConns(1) + NewStore
// pattern from spotify/store_test.go and selection/store_test.go.
func newTestStore(t *testing.T) *Store {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore returned error: %v", err)
	}
	return store
}

func sampleTracks() []ConfirmedTrack {
	return []ConfirmedTrack{
		{SpotifyTrackID: "track-a", Metadata: candidate.CandidateMetadata{Title: "A"}},
		{SpotifyTrackID: "track-b", Metadata: candidate.CandidateMetadata{Title: "B"}},
		{SpotifyTrackID: "track-c", Metadata: candidate.CandidateMetadata{Title: "C"}},
	}
}

func trackIDs(tracks []ConfirmedTrack) []string {
	ids := make([]string, len(tracks))
	for i, t := range tracks {
		ids[i] = t.SpotifyTrackID
	}
	return ids
}

// --- Lifecycle ---

func TestCreateDraftStartsInDraft(t *testing.T) {
	store := newTestStore(t)
	e, err := store.CreateDraft(context.Background())
	if err != nil {
		t.Fatalf("CreateDraft returned error: %v", err)
	}
	if e.Status != StatusDraft {
		t.Fatalf("expected StatusDraft, got %q", e.Status)
	}
	if e.ID == "" {
		t.Fatalf("expected a non-empty ID")
	}
}

func TestConfirmRejectsEmptySnapshot(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	draft, _ := store.CreateDraft(ctx)

	_, err := store.Confirm(ctx, draft.ID, nil)
	if err != ErrEmptySnapshot {
		t.Fatalf("expected ErrEmptySnapshot, got %v", err)
	}
}

func TestConfirmFromDraftSucceeds(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	draft, _ := store.CreateDraft(ctx)

	confirmed, err := store.Confirm(ctx, draft.ID, sampleTracks())
	if err != nil {
		t.Fatalf("Confirm returned error: %v", err)
	}
	if confirmed.Status != StatusConfirmed {
		t.Fatalf("expected StatusConfirmed, got %q", confirmed.Status)
	}
	if confirmed.ConfirmedAt == nil {
		t.Fatalf("expected ConfirmedAt to be set")
	}
}

func TestReconfirmWhileConfirmedSucceeds(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	draft, _ := store.CreateDraft(ctx)
	store.Confirm(ctx, draft.ID, sampleTracks())

	reordered := []ConfirmedTrack{sampleTracks()[2], sampleTracks()[0], sampleTracks()[1]}
	confirmed, err := store.Confirm(ctx, draft.ID, reordered)
	if err != nil {
		t.Fatalf("re-Confirm returned error: %v", err)
	}
	if got := trackIDs(confirmed.ConfirmedTracks); fmt.Sprint(got) != fmt.Sprint(trackIDs(reordered)) {
		t.Fatalf("expected re-confirm to overwrite order, got %v", got)
	}
}

func TestConfirmAfterPublishingStartedIsLocked(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	draft, _ := store.CreateDraft(ctx)
	store.Confirm(ctx, draft.ID, sampleTracks())
	store.StartPublishing(ctx, draft.ID)

	if _, err := store.Confirm(ctx, draft.ID, sampleTracks()); err != ErrSnapshotLocked {
		t.Fatalf("expected ErrSnapshotLocked, got %v", err)
	}
}

func TestConfirmOnMissingEditionFails(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Confirm(context.Background(), "nonexistent", sampleTracks()); err != ErrEditionNotFound {
		t.Fatalf("expected ErrEditionNotFound, got %v", err)
	}
}

func TestValidTransitionsSucceed(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	draft, _ := store.CreateDraft(ctx)
	store.Confirm(ctx, draft.ID, sampleTracks())

	if _, err := store.StartPublishing(ctx, draft.ID); err != nil {
		t.Fatalf("StartPublishing returned error: %v", err)
	}
	published, err := store.RecordPublishSuccess(ctx, draft.ID, "sp-playlist-1", "https://open.spotify.com/playlist/sp-playlist-1")
	if err != nil {
		t.Fatalf("RecordPublishSuccess returned error: %v", err)
	}
	if published.Status != StatusPublished {
		t.Fatalf("expected StatusPublished, got %q", published.Status)
	}
	if published.PublishedAt == nil {
		t.Fatalf("expected PublishedAt to be set")
	}

	archived, err := store.Archive(ctx, draft.ID)
	if err != nil {
		t.Fatalf("Archive returned error: %v", err)
	}
	if archived.Status != StatusArchived {
		t.Fatalf("expected StatusArchived, got %q", archived.Status)
	}
	if archived.ArchivedAt == nil {
		t.Fatalf("expected ArchivedAt to be set")
	}
}

func TestInvalidTransitionsAreRejected(t *testing.T) {
	ctx := context.Background()

	t.Run("draft to publishing", func(t *testing.T) {
		store := newTestStore(t)
		draft, _ := store.CreateDraft(ctx)
		if _, err := store.StartPublishing(ctx, draft.ID); err != ErrInvalidTransition {
			t.Fatalf("expected ErrInvalidTransition, got %v", err)
		}
	})

	// draft/confirmed/publishing → archived are covered by
	// TestArchiveOnlyFromPublished.

	t.Run("published back to publishing", func(t *testing.T) {
		store := newTestStore(t)
		draft, _ := store.CreateDraft(ctx)
		store.Confirm(ctx, draft.ID, sampleTracks())
		store.StartPublishing(ctx, draft.ID)
		store.RecordPublishSuccess(ctx, draft.ID, "sp-1", "https://example.com/sp-1")
		if _, err := store.StartPublishing(ctx, draft.ID); err != ErrInvalidTransition {
			t.Fatalf("expected ErrInvalidTransition, got %v", err)
		}
	})

	t.Run("archived to anything", func(t *testing.T) {
		store := newTestStore(t)
		draft, _ := store.CreateDraft(ctx)
		store.Confirm(ctx, draft.ID, sampleTracks())
		store.StartPublishing(ctx, draft.ID)
		store.RecordPublishSuccess(ctx, draft.ID, "sp-1", "https://example.com/sp-1")
		store.Archive(ctx, draft.ID)
		if _, err := store.Archive(ctx, draft.ID); err != ErrInvalidTransition {
			t.Fatalf("expected ErrInvalidTransition re-archiving, got %v", err)
		}
	})
}

func TestArchiveOnlyFromPublished(t *testing.T) {
	ctx := context.Background()

	t.Run("draft cannot be archived", func(t *testing.T) {
		store := newTestStore(t)
		draft, _ := store.CreateDraft(ctx)
		if _, err := store.Archive(ctx, draft.ID); err != ErrInvalidTransition {
			t.Fatalf("expected ErrInvalidTransition, got %v", err)
		}
	})

	t.Run("confirmed cannot be archived", func(t *testing.T) {
		store := newTestStore(t)
		draft, _ := store.CreateDraft(ctx)
		store.Confirm(ctx, draft.ID, sampleTracks())
		if _, err := store.Archive(ctx, draft.ID); err != ErrInvalidTransition {
			t.Fatalf("expected ErrInvalidTransition, got %v", err)
		}
	})

	t.Run("publishing cannot be archived", func(t *testing.T) {
		store := newTestStore(t)
		draft, _ := store.CreateDraft(ctx)
		store.Confirm(ctx, draft.ID, sampleTracks())
		store.StartPublishing(ctx, draft.ID)
		if _, err := store.Archive(ctx, draft.ID); err != ErrInvalidTransition {
			t.Fatalf("expected ErrInvalidTransition, got %v", err)
		}
	})
}

// --- Snapshot and ordering ---

func TestConfirmPreservesExactInputOrder(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	draft, _ := store.CreateDraft(ctx)
	tracks := sampleTracks()

	confirmed, err := store.Confirm(ctx, draft.ID, tracks)
	if err != nil {
		t.Fatalf("Confirm returned error: %v", err)
	}
	if got, want := trackIDs(confirmed.ConfirmedTracks), trackIDs(tracks); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("order not preserved: got %v, want %v", got, want)
	}
}

func TestPersistedOrderSurvivesRoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	draft, _ := store.CreateDraft(ctx)
	tracks := sampleTracks()
	store.Confirm(ctx, draft.ID, tracks)

	fetched, err := store.GetByID(ctx, draft.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if fetched == nil {
		t.Fatalf("expected edition to exist")
	}
	if got, want := trackIDs(fetched.ConfirmedTracks), trackIDs(tracks); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("order not preserved across round trip: got %v, want %v", got, want)
	}
	for i, tr := range fetched.ConfirmedTracks {
		if tr.Metadata.Title != tracks[i].Metadata.Title {
			t.Fatalf("metadata not preserved for track %d: got %q, want %q", i, tr.Metadata.Title, tracks[i].Metadata.Title)
		}
	}
}

func TestPublicationMetadataDoesNotChangeConfirmedOrder(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	draft, _ := store.CreateDraft(ctx)
	tracks := sampleTracks()
	confirmed, _ := store.Confirm(ctx, draft.ID, tracks)
	wantOrder := trackIDs(confirmed.ConfirmedTracks)

	store.StartPublishing(ctx, draft.ID)
	published, err := store.RecordPublishSuccess(ctx, draft.ID, "sp-1", "https://example.com/sp-1")
	if err != nil {
		t.Fatalf("RecordPublishSuccess returned error: %v", err)
	}
	if got := trackIDs(published.ConfirmedTracks); fmt.Sprint(got) != fmt.Sprint(wantOrder) {
		t.Fatalf("publication metadata changed confirmed order: got %v, want %v", got, wantOrder)
	}
}

// --- Single-active-edition invariant ---

func TestFirstActiveEditionCanBeCreated(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.CreateDraft(context.Background()); err != nil {
		t.Fatalf("CreateDraft returned error: %v", err)
	}
}

func TestSecondActiveEditionIsRejected(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	if _, err := store.CreateDraft(ctx); err != nil {
		t.Fatalf("first CreateDraft returned error: %v", err)
	}
	if _, err := store.CreateDraft(ctx); err != ErrActiveEditionExists {
		t.Fatalf("expected ErrActiveEditionExists, got %v", err)
	}
}

func TestArchivingPermitsTheNextEditionToBeCreated(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	first, _ := store.CreateDraft(ctx)
	store.Confirm(ctx, first.ID, sampleTracks())
	store.StartPublishing(ctx, first.ID)
	store.RecordPublishSuccess(ctx, first.ID, "sp-1", "https://example.com/sp-1")
	if _, err := store.Archive(ctx, first.ID); err != nil {
		t.Fatalf("Archive returned error: %v", err)
	}

	second, err := store.CreateDraft(ctx)
	if err != nil {
		t.Fatalf("expected a new Edition to be creatable after archiving, got error: %v", err)
	}
	if second.ID == first.ID {
		t.Fatalf("expected a distinct Edition ID")
	}
}

// TestConcurrentCreateDraftOnlyOneSucceeds exercises the single-active-
// edition UNIQUE index under real concurrency: several goroutines call
// CreateDraft at the same time against one shared Store. CreateDraft is a
// single INSERT with no separate check-then-write step, so the UNIQUE
// index — not any application-level ordering — is what must reject every
// attempt but the first, regardless of how goroutines happen to interleave.
func TestConcurrentCreateDraftOnlyOneSucceeds(t *testing.T) {
	store := newTestStore(t)

	const attempts = 8
	var wg sync.WaitGroup
	results := make(chan error, attempts)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.CreateDraft(context.Background())
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	successes, rejections := 0, 0
	for err := range results {
		switch err {
		case nil:
			successes++
		case ErrActiveEditionExists:
			rejections++
		default:
			t.Fatalf("unexpected error from concurrent CreateDraft: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("expected exactly 1 successful CreateDraft under concurrency, got %d (rejections: %d)", successes, rejections)
	}
	if rejections != attempts-1 {
		t.Fatalf("expected %d rejections, got %d", attempts-1, rejections)
	}
}

// --- Retry safety ---

func TestFailedPublishPreservesEditionIdentityAndSnapshot(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	draft, _ := store.CreateDraft(ctx)
	confirmed, _ := store.Confirm(ctx, draft.ID, sampleTracks())
	wantOrder := trackIDs(confirmed.ConfirmedTracks)

	store.StartPublishing(ctx, draft.ID)
	failed, err := store.RecordPublishFailure(ctx, draft.ID, "spotify: transient 502")
	if err != nil {
		t.Fatalf("RecordPublishFailure returned error: %v", err)
	}
	if failed.ID != draft.ID {
		t.Fatalf("expected Edition identity preserved, got ID %q", failed.ID)
	}
	if failed.Status != StatusConfirmed {
		t.Fatalf("expected failure to return to StatusConfirmed, got %q", failed.Status)
	}
	if got := trackIDs(failed.ConfirmedTracks); fmt.Sprint(got) != fmt.Sprint(wantOrder) {
		t.Fatalf("confirmed snapshot changed after failure: got %v, want %v", got, wantOrder)
	}
	if failed.LastPublishError == nil || *failed.LastPublishError != "spotify: transient 502" {
		t.Fatalf("expected LastPublishError to be recorded, got %v", failed.LastPublishError)
	}
}

func TestRetryAfterFailureNeedsNoReconfirm(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	draft, _ := store.CreateDraft(ctx)
	store.Confirm(ctx, draft.ID, sampleTracks())
	store.StartPublishing(ctx, draft.ID)
	store.RecordPublishFailure(ctx, draft.ID, "spotify: transient 502")

	// Retry: StartPublishing again directly from the Confirmed state the
	// failure returned to, with no intervening Confirm call.
	retried, err := store.StartPublishing(ctx, draft.ID)
	if err != nil {
		t.Fatalf("retry StartPublishing returned error: %v", err)
	}
	if retried.Status != StatusPublishing {
		t.Fatalf("expected StatusPublishing on retry, got %q", retried.Status)
	}

	published, err := store.RecordPublishSuccess(ctx, draft.ID, "sp-1", "https://example.com/sp-1")
	if err != nil {
		t.Fatalf("RecordPublishSuccess returned error: %v", err)
	}
	if published.Status != StatusPublished {
		t.Fatalf("expected StatusPublished, got %q", published.Status)
	}
}

func TestPublicationMetadataRecordedWithoutChangingTrackOrder(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	draft, _ := store.CreateDraft(ctx)
	confirmed, _ := store.Confirm(ctx, draft.ID, sampleTracks())
	wantOrder := trackIDs(confirmed.ConfirmedTracks)
	store.StartPublishing(ctx, draft.ID)

	published, err := store.RecordPublishSuccess(ctx, draft.ID, "sp-99", "https://example.com/sp-99")
	if err != nil {
		t.Fatalf("RecordPublishSuccess returned error: %v", err)
	}
	if published.SpotifyPlaylistID != "sp-99" {
		t.Fatalf("expected SpotifyPlaylistID to be recorded, got %q", published.SpotifyPlaylistID)
	}
	if got := trackIDs(published.ConfirmedTracks); fmt.Sprint(got) != fmt.Sprint(wantOrder) {
		t.Fatalf("publication metadata changed track order: got %v, want %v", got, wantOrder)
	}
}

// --- Persistence ---

func TestGetActiveReturnsNilWhenNoneExists(t *testing.T) {
	store := newTestStore(t)
	active, err := store.GetActive(context.Background())
	if err != nil {
		t.Fatalf("GetActive returned error: %v", err)
	}
	if active != nil {
		t.Fatalf("expected nil, got %+v", active)
	}
}

func TestGetActiveExcludesArchived(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	draft, _ := store.CreateDraft(ctx)
	store.Confirm(ctx, draft.ID, sampleTracks())
	store.StartPublishing(ctx, draft.ID)
	store.RecordPublishSuccess(ctx, draft.ID, "sp-1", "https://example.com/sp-1")
	store.Archive(ctx, draft.ID)

	active, err := store.GetActive(ctx)
	if err != nil {
		t.Fatalf("GetActive returned error: %v", err)
	}
	if active != nil {
		t.Fatalf("expected no active edition after archiving, got %+v", active)
	}
}

func TestGetByIDReturnsNilForUnknownID(t *testing.T) {
	store := newTestStore(t)
	e, err := store.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if e != nil {
		t.Fatalf("expected nil, got %+v", e)
	}
}

func TestStoreMethodsReturnErrorOnClosedDB(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore returned error: %v", err)
	}
	db.Close()

	if _, err := store.CreateDraft(context.Background()); err == nil {
		t.Fatalf("expected an error from CreateDraft against a closed database")
	}
}
