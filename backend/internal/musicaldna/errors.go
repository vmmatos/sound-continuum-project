package musicaldna

import "errors"

// Sentinels for WeeklyDirection validation failures. Use errors.Is against
// these.
var (
	// ErrEmptyEditionID is returned when a WeeklyDirection has no edition
	// identity.
	ErrEmptyEditionID = errors.New("musicaldna: edition id must not be empty")
)
