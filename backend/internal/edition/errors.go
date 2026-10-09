package edition

import "errors"

// Sentinels for Edition domain/persistence failures. Use errors.Is against
// these — same convention as candidate/selection's errors.go.
var (
	// ErrEmptySnapshot is returned when confirming with zero tracks — an
	// empty Edition cannot be confirmed (Step 3).
	ErrEmptySnapshot = errors.New("edition: confirmed snapshot must not be empty")

	// ErrInvalidTransition is returned when a lifecycle Status change is
	// not one of Status.CanTransitionTo's allowed edges.
	ErrInvalidTransition = errors.New("edition: invalid lifecycle transition")

	// ErrActiveEditionExists is returned when creating a new Edition while
	// one non-archived Edition already exists (Step 4).
	ErrActiveEditionExists = errors.New("edition: an active edition already exists")

	// ErrEditionNotFound is returned when an operation targets an Edition
	// ID that doesn't exist.
	ErrEditionNotFound = errors.New("edition: not found")

	// ErrSnapshotLocked is returned when confirming an Edition whose
	// Status is already Publishing, Published, or Archived — the confirmed
	// boundary is final once publishing has started (Step 3/6).
	ErrSnapshotLocked = errors.New("edition: confirmed snapshot is locked, publishing has started")
)
