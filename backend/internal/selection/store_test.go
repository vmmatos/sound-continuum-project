package selection

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	// modernc.org/sqlite's :memory: database is per-connection; a single
	// connection keeps the whole test on the same in-memory database (same
	// pattern as backend/internal/spotify/store_test.go).
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore returned error: %v", err)
	}
	return store
}

func TestStoreAllSelectedEmpty(t *testing.T) {
	store := newTestStore(t)

	selected, err := store.AllSelected(context.Background())
	if err != nil {
		t.Fatalf("AllSelected returned error: %v", err)
	}
	if len(selected) != 0 {
		t.Fatalf("expected no selected candidates, got %v", selected)
	}
}

func TestStoreKeepMarksSelected(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.Keep(ctx, "cand-1"); err != nil {
		t.Fatalf("Keep returned error: %v", err)
	}

	selected, err := store.AllSelected(ctx)
	if err != nil {
		t.Fatalf("AllSelected returned error: %v", err)
	}
	if _, ok := selected["cand-1"]; !ok {
		t.Fatalf("expected cand-1 to be selected, got %v", selected)
	}
}

func TestStoreKeepIsIdempotent(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.Keep(ctx, "cand-1"); err != nil {
		t.Fatalf("first Keep returned error: %v", err)
	}
	if err := store.Keep(ctx, "cand-1"); err != nil {
		t.Fatalf("second Keep returned error: %v", err)
	}

	selected, err := store.AllSelected(ctx)
	if err != nil {
		t.Fatalf("AllSelected returned error: %v", err)
	}
	if len(selected) != 1 {
		t.Fatalf("expected exactly one selected candidate after repeated Keep, got %v", selected)
	}
	if _, ok := selected["cand-1"]; !ok {
		t.Fatalf("expected cand-1 to still be selected, got %v", selected)
	}
}

func TestStoreKeepDoesNotAffectUnrelatedCandidates(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.Keep(ctx, "cand-1"); err != nil {
		t.Fatalf("Keep returned error: %v", err)
	}

	selected, err := store.AllSelected(ctx)
	if err != nil {
		t.Fatalf("AllSelected returned error: %v", err)
	}
	if _, ok := selected["cand-2"]; ok {
		t.Fatalf("expected cand-2 to remain unselected, got %v", selected)
	}
	if len(selected) != 1 {
		t.Fatalf("expected exactly one selected candidate, got %v", selected)
	}
}

func TestStoreAllUnderReviewEmpty(t *testing.T) {
	store := newTestStore(t)

	underReview, err := store.AllUnderReview(context.Background())
	if err != nil {
		t.Fatalf("AllUnderReview returned error: %v", err)
	}
	if len(underReview) != 0 {
		t.Fatalf("expected no under-review candidates, got %v", underReview)
	}
}

func TestStoreMaybeMarksUnderReview(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.Maybe(ctx, "cand-1"); err != nil {
		t.Fatalf("Maybe returned error: %v", err)
	}

	underReview, err := store.AllUnderReview(ctx)
	if err != nil {
		t.Fatalf("AllUnderReview returned error: %v", err)
	}
	if _, ok := underReview["cand-1"]; !ok {
		t.Fatalf("expected cand-1 to be under review, got %v", underReview)
	}
}

func TestStoreMaybeIsIdempotent(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.Maybe(ctx, "cand-1"); err != nil {
		t.Fatalf("first Maybe returned error: %v", err)
	}
	if err := store.Maybe(ctx, "cand-1"); err != nil {
		t.Fatalf("second Maybe returned error: %v", err)
	}

	underReview, err := store.AllUnderReview(ctx)
	if err != nil {
		t.Fatalf("AllUnderReview returned error: %v", err)
	}
	if len(underReview) != 1 {
		t.Fatalf("expected exactly one under-review candidate after repeated Maybe, got %v", underReview)
	}
}

func TestStoreMaybeDoesNotAffectUnrelatedCandidates(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.Maybe(ctx, "cand-1"); err != nil {
		t.Fatalf("Maybe returned error: %v", err)
	}

	underReview, err := store.AllUnderReview(ctx)
	if err != nil {
		t.Fatalf("AllUnderReview returned error: %v", err)
	}
	if _, ok := underReview["cand-2"]; ok {
		t.Fatalf("expected cand-2 to remain unaffected, got %v", underReview)
	}
}

func TestStoreKeepThenMaybeClearsKeep(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.Keep(ctx, "cand-1"); err != nil {
		t.Fatalf("Keep returned error: %v", err)
	}
	if err := store.Maybe(ctx, "cand-1"); err != nil {
		t.Fatalf("Maybe returned error: %v", err)
	}

	selected, err := store.AllSelected(ctx)
	if err != nil {
		t.Fatalf("AllSelected returned error: %v", err)
	}
	if _, ok := selected["cand-1"]; ok {
		t.Fatalf("expected cand-1 to no longer be selected after Maybe, got %v", selected)
	}

	underReview, err := store.AllUnderReview(ctx)
	if err != nil {
		t.Fatalf("AllUnderReview returned error: %v", err)
	}
	if _, ok := underReview["cand-1"]; !ok {
		t.Fatalf("expected cand-1 to be under review, got %v", underReview)
	}
}

func TestStoreMaybeThenKeepClearsMaybe(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.Maybe(ctx, "cand-1"); err != nil {
		t.Fatalf("Maybe returned error: %v", err)
	}
	if err := store.Keep(ctx, "cand-1"); err != nil {
		t.Fatalf("Keep returned error: %v", err)
	}

	underReview, err := store.AllUnderReview(ctx)
	if err != nil {
		t.Fatalf("AllUnderReview returned error: %v", err)
	}
	if _, ok := underReview["cand-1"]; ok {
		t.Fatalf("expected cand-1 to no longer be under review after Keep, got %v", underReview)
	}

	selected, err := store.AllSelected(ctx)
	if err != nil {
		t.Fatalf("AllSelected returned error: %v", err)
	}
	if _, ok := selected["cand-1"]; !ok {
		t.Fatalf("expected cand-1 to be selected, got %v", selected)
	}
}

func TestStoreClearIsIdempotentAndRemovesDecision(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Clearing a candidate with no prior decision is a no-op, not an error.
	if err := store.Clear(ctx, "cand-1"); err != nil {
		t.Fatalf("Clear on undecided candidate returned error: %v", err)
	}

	if err := store.Keep(ctx, "cand-1"); err != nil {
		t.Fatalf("Keep returned error: %v", err)
	}
	if err := store.Clear(ctx, "cand-1"); err != nil {
		t.Fatalf("first Clear returned error: %v", err)
	}
	if err := store.Clear(ctx, "cand-1"); err != nil {
		t.Fatalf("second Clear returned error: %v", err)
	}

	selected, err := store.AllSelected(ctx)
	if err != nil {
		t.Fatalf("AllSelected returned error: %v", err)
	}
	if _, ok := selected["cand-1"]; ok {
		t.Fatalf("expected cand-1 to no longer be selected after Clear, got %v", selected)
	}
	underReview, err := store.AllUnderReview(ctx)
	if err != nil {
		t.Fatalf("AllUnderReview returned error: %v", err)
	}
	if _, ok := underReview["cand-1"]; ok {
		t.Fatalf("expected cand-1 to not be under review after Clear, got %v", underReview)
	}
}

func TestStoreAllRejectedEmpty(t *testing.T) {
	store := newTestStore(t)

	rejected, err := store.AllRejected(context.Background())
	if err != nil {
		t.Fatalf("AllRejected returned error: %v", err)
	}
	if len(rejected) != 0 {
		t.Fatalf("expected no rejected candidates, got %v", rejected)
	}
}

func TestStoreRejectMarksRejected(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.Reject(ctx, "cand-1"); err != nil {
		t.Fatalf("Reject returned error: %v", err)
	}

	rejected, err := store.AllRejected(ctx)
	if err != nil {
		t.Fatalf("AllRejected returned error: %v", err)
	}
	if _, ok := rejected["cand-1"]; !ok {
		t.Fatalf("expected cand-1 to be rejected, got %v", rejected)
	}
}

func TestStoreRejectIsIdempotent(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.Reject(ctx, "cand-1"); err != nil {
		t.Fatalf("first Reject returned error: %v", err)
	}
	if err := store.Reject(ctx, "cand-1"); err != nil {
		t.Fatalf("second Reject returned error: %v", err)
	}

	rejected, err := store.AllRejected(ctx)
	if err != nil {
		t.Fatalf("AllRejected returned error: %v", err)
	}
	if len(rejected) != 1 {
		t.Fatalf("expected exactly one rejected candidate after repeated Reject, got %v", rejected)
	}
}

func TestStoreRejectDoesNotAffectUnrelatedCandidates(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.Reject(ctx, "cand-1"); err != nil {
		t.Fatalf("Reject returned error: %v", err)
	}

	rejected, err := store.AllRejected(ctx)
	if err != nil {
		t.Fatalf("AllRejected returned error: %v", err)
	}
	if _, ok := rejected["cand-2"]; ok {
		t.Fatalf("expected cand-2 to remain unaffected, got %v", rejected)
	}
}

func TestStoreKeepThenRejectClearsKeep(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.Keep(ctx, "cand-1"); err != nil {
		t.Fatalf("Keep returned error: %v", err)
	}
	if err := store.Reject(ctx, "cand-1"); err != nil {
		t.Fatalf("Reject returned error: %v", err)
	}

	selected, err := store.AllSelected(ctx)
	if err != nil {
		t.Fatalf("AllSelected returned error: %v", err)
	}
	if _, ok := selected["cand-1"]; ok {
		t.Fatalf("expected cand-1 to no longer be selected after Reject, got %v", selected)
	}

	rejected, err := store.AllRejected(ctx)
	if err != nil {
		t.Fatalf("AllRejected returned error: %v", err)
	}
	if _, ok := rejected["cand-1"]; !ok {
		t.Fatalf("expected cand-1 to be rejected, got %v", rejected)
	}
}

func TestStoreMaybeThenRejectClearsMaybe(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.Maybe(ctx, "cand-1"); err != nil {
		t.Fatalf("Maybe returned error: %v", err)
	}
	if err := store.Reject(ctx, "cand-1"); err != nil {
		t.Fatalf("Reject returned error: %v", err)
	}

	underReview, err := store.AllUnderReview(ctx)
	if err != nil {
		t.Fatalf("AllUnderReview returned error: %v", err)
	}
	if _, ok := underReview["cand-1"]; ok {
		t.Fatalf("expected cand-1 to no longer be under review after Reject, got %v", underReview)
	}

	rejected, err := store.AllRejected(ctx)
	if err != nil {
		t.Fatalf("AllRejected returned error: %v", err)
	}
	if _, ok := rejected["cand-1"]; !ok {
		t.Fatalf("expected cand-1 to be rejected, got %v", rejected)
	}
}

func TestStoreRejectThenKeepClearsReject(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.Reject(ctx, "cand-1"); err != nil {
		t.Fatalf("Reject returned error: %v", err)
	}
	if err := store.Keep(ctx, "cand-1"); err != nil {
		t.Fatalf("Keep returned error: %v", err)
	}

	rejected, err := store.AllRejected(ctx)
	if err != nil {
		t.Fatalf("AllRejected returned error: %v", err)
	}
	if _, ok := rejected["cand-1"]; ok {
		t.Fatalf("expected cand-1 to no longer be rejected after Keep, got %v", rejected)
	}

	selected, err := store.AllSelected(ctx)
	if err != nil {
		t.Fatalf("AllSelected returned error: %v", err)
	}
	if _, ok := selected["cand-1"]; !ok {
		t.Fatalf("expected cand-1 to be selected, got %v", selected)
	}
}

func TestStoreRejectThenMaybeClearsReject(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.Reject(ctx, "cand-1"); err != nil {
		t.Fatalf("Reject returned error: %v", err)
	}
	if err := store.Maybe(ctx, "cand-1"); err != nil {
		t.Fatalf("Maybe returned error: %v", err)
	}

	rejected, err := store.AllRejected(ctx)
	if err != nil {
		t.Fatalf("AllRejected returned error: %v", err)
	}
	if _, ok := rejected["cand-1"]; ok {
		t.Fatalf("expected cand-1 to no longer be rejected after Maybe, got %v", rejected)
	}

	underReview, err := store.AllUnderReview(ctx)
	if err != nil {
		t.Fatalf("AllUnderReview returned error: %v", err)
	}
	if _, ok := underReview["cand-1"]; !ok {
		t.Fatalf("expected cand-1 to be under review, got %v", underReview)
	}
}

func TestStoreClearIsIdempotentAndRemovesRejectDecision(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.Reject(ctx, "cand-1"); err != nil {
		t.Fatalf("Reject returned error: %v", err)
	}
	if err := store.Clear(ctx, "cand-1"); err != nil {
		t.Fatalf("first Clear returned error: %v", err)
	}
	if err := store.Clear(ctx, "cand-1"); err != nil {
		t.Fatalf("second Clear returned error: %v", err)
	}

	rejected, err := store.AllRejected(ctx)
	if err != nil {
		t.Fatalf("AllRejected returned error: %v", err)
	}
	if _, ok := rejected["cand-1"]; ok {
		t.Fatalf("expected cand-1 to no longer be rejected after Clear, got %v", rejected)
	}
}
