package musicaldna

import (
	"errors"
	"testing"
)

func strPtr(s string) *string { return &s }

func TestNewWeeklyDirectionRejectsEmptyEditionID(t *testing.T) {
	cases := []string{"", "   ", "\t"}
	for _, id := range cases {
		_, err := NewWeeklyDirection(id, Profile{}, "")
		if !errors.Is(err, ErrEmptyEditionID) {
			t.Errorf("EditionID=%q: err = %v, want ErrEmptyEditionID", id, err)
		}
	}
}

func TestNewWeeklyDirectionSetsFields(t *testing.T) {
	profile := Profile{Mood: strPtr("melancholic"), Energy: strPtr("low")}
	got, err := NewWeeklyDirection("2026-w40", profile, "leaning into a soulful, late-night arc")
	if err != nil {
		t.Fatalf("NewWeeklyDirection() err = %v, want nil", err)
	}
	if got.EditionID != "2026-w40" {
		t.Errorf("EditionID = %v, want 2026-w40", got.EditionID)
	}
	if got.Profile != profile {
		t.Errorf("Profile = %+v, want %+v", got.Profile, profile)
	}
	if got.Notes != "leaning into a soulful, late-night arc" {
		t.Errorf("Notes = %q, want the given notes", got.Notes)
	}
}

func TestDefaultProjectDNAIsEmpty(t *testing.T) {
	dna := DefaultProjectDNA()
	if dna.Profile != (Profile{}) {
		t.Errorf("DefaultProjectDNA().Profile = %+v, want zero value", dna.Profile)
	}
}
