package musicaldna

import "strings"

// WeeklyDirection is the explicit musical direction a human curator has set
// for the current edition. It is distinct from ProjectDNA: ProjectDNA is
// stable across editions, WeeklyDirection changes every week and carries
// greater contextual weight in Fit (see scoring.DefaultFitWeights).
//
// This is a domain-level representation only. Card #41 does not add a UI,
// an HTTP endpoint, or persistence for defining it — per the card brief, a
// curator-facing editor is future work, and the direction must remain an
// explicit human input, never inferred from whichever candidates a
// discovery workflow happens to produce.
type WeeklyDirection struct {
	EditionID string
	Profile   Profile
	Notes     string // free-text editorial context; not used in Fit math
}

// NewWeeklyDirection builds a WeeklyDirection, rejecting an empty or
// whitespace-only EditionID — a direction must be tied to a specific
// edition, not left ambiguous about which chapter it describes.
func NewWeeklyDirection(editionID string, profile Profile, notes string) (WeeklyDirection, error) {
	if strings.TrimSpace(editionID) == "" {
		return WeeklyDirection{}, ErrEmptyEditionID
	}
	return WeeklyDirection{
		EditionID: editionID,
		Profile:   profile,
		Notes:     notes,
	}, nil
}
