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
