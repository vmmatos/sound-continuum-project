// Package musicaldna defines Sound Continuum's shared vocabulary of
// musical-character dimensions ("Musical DNA") and the two editorial
// contexts it describes: the project's stable identity (ProjectDNA) and the
// current edition's direction (WeeklyDirection). See docs/scoring-model.md
// and docs/memory/decisions.md (Card #41) for the full reasoning.
//
// Every dimension here is optional and comes only from explicit editorial
// input — this package never derives a value from Spotify metadata, genre
// strings, popularity, or audio analysis. Audio Features/Audio Analysis are
// unavailable through the current Spotify API (see docs/memory/decisions.md),
// and even if they were available, translating them into these dimensions
// would be a different, unimplemented capability. A nil field means "not
// supplied," never "poor fit."
package musicaldna

// Profile is one set of musical-character dimension values. The same type
// describes three different things depending on context: a candidate's
// editorially-tagged characteristics, ProjectDNA's stable baseline, and
// WeeklyDirection's target for the current edition — one vocabulary, reused,
// rather than three competing representations.
//
// The dimension set is intentionally small. It is drawn from the qualities
// docs/scoring-model.md already names for the Fit factor (mood, energy,
// texture, cultural context) rather than invented independently, and it
// excludes qualities that would require unavailable audio-level analysis to
// observe reliably (e.g. tempo, detailed rhythmic/melodic structure) — those
// remain future work, not fabricated placeholders.
type Profile struct {
	Mood              *string
	Energy            *string
	Texture           *string
	CulturalInfluence *string
}
