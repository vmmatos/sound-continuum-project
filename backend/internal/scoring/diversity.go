package scoring

import (
	"slices"
	"strconv"
	"strings"

	"github.com/vmmatos/sound-continuum-project/internal/musicaldna"
)

// EditionTrack is the minimal representation of one track already part of
// the current edition being assembled, used only to evaluate Diversity
// against. It is deliberately not candidate.CandidateTrack and not the
// official historical Spotify playlist — it carries only what Diversity
// needs to answer "what's already represented in this edition?"
type EditionTrack struct {
	// ArtistSpotifyIDs reuses the existing Spotify artist identity
	// (candidate.CandidateArtist.SpotifyArtistID) — no second artist
	// identity model.
	ArtistSpotifyIDs []string
	// Era is a coarse decade string (e.g. "1990s"), produced by
	// DiversityEra; nil if unknown.
	Era *string
	// Sound reuses musicaldna.Profile — the same explicit, editorially
	// supplied sound vocabulary Fit (Card #41) already uses — as the
	// smallest sound-oriented representation available, without coupling
	// Diversity to Fit's calculation.
	Sound musicaldna.Profile
}

// CurrentEditionContext describes what is already represented in the
// edition currently being assembled/evaluated. It is a transient,
// function-level input — not persisted, not derived from the candidate
// pool, and not the historical Spotify playlist (that context belongs to
// Freshness/Repetition Penalty, not Diversity).
type CurrentEditionContext struct {
	Tracks []EditionTrack
}

// DiversityEra extracts a coarse decade (e.g. "1990s") from a release date
// string in whatever precision the provider reported (year, year-month, or
// full date — all start with a 4-digit year). Returns nil for an empty or
// unparseable value; no precision beyond "decade" is invented.
func DiversityEra(releaseDate string) *string {
	if len(releaseDate) < 4 {
		return nil
	}
	year, err := strconv.Atoi(releaseDate[:4])
	if err != nil || year <= 0 {
		return nil
	}
	decade := (year / 10) * 10
	era := strconv.Itoa(decade) + "s"
	return &era
}

// DiversitySoundDimension identifies one musicaldna.Profile dimension
// considered when evaluating Sound Diversity.
type DiversitySoundDimension string

const (
	DiversitySoundMood              DiversitySoundDimension = "mood"
	DiversitySoundEnergy            DiversitySoundDimension = "energy"
	DiversitySoundTexture           DiversitySoundDimension = "texture"
	DiversitySoundCulturalInfluence DiversitySoundDimension = "cultural_influence"
)

// DiversityAspectResult explains one of Diversity's three aspects (Artist,
// Era). Occurrences is the number of current-edition tracks already
// contributing to this candidate's concentration; Value is nil when
// Available is false.
type DiversityAspectResult struct {
	Available   bool
	Occurrences int
	Value       *float64
}

// DiversitySoundDimensionResult explains one Sound dimension's
// contribution.
type DiversitySoundDimensionResult struct {
	Dimension   DiversitySoundDimension
	Available   bool
	Occurrences int
	Value       *float64
}

// DiversitySoundResult is Sound Diversity's outcome: an overall Value (nil
// if no dimension was available on the candidate) plus every dimension's
// own contribution, for explainability.
type DiversitySoundResult struct {
	Available  bool
	Value      *float64
	Dimensions []DiversitySoundDimensionResult
}

// DiversityResult is the outcome of CalculateDiversity. Value is nil when
// no current-edition context was supplied, when the edition is empty (no
// concentration yet is not the same as maximal diversity — see
// CalculateDiversity's doc comment), or when none of the three dimensions
// were available. ContextProvided/EditionEmpty make these cases
// independently inspectable rather than collapsing them into one opaque
// nil.
type DiversityResult struct {
	Value           *float64
	ContextProvided bool
	EditionSize     int
	EditionEmpty    bool
	Artist          DiversityAspectResult
	Era             DiversityAspectResult
	Sound           DiversitySoundResult
}

// diversityContribution is the shared diminishing-concentration formula:
// 0 existing occurrences contributes the strongest signal (1.0),
// progressively less as concentration increases. Deterministic, bounded
// (0,1], with no hardcoded per-occurrence thresholds.
func diversityContribution(occurrences int) float64 {
	return 1 / float64(1+occurrences)
}

// CalculateDiversity computes the Diversity factor: does this candidate
// contribute meaningful variation to the current edition context, across
// Artist, Era, and Sound concentration? It is contextual by construction —
// a candidate cannot be evaluated for diversity in isolation, only against
// ctx.
//
// candidateArtistSpotifyIDs/candidateEra/candidateSound are plain values
// the caller derives from a real candidate (e.g. its enriched
// CandidateMetadata) — CalculateDiversity itself takes no
// candidate.CandidateTrack, CandidateType, Category, Source,
// DiscoveryProvenance, or any other scoring factor's value, so it cannot
// be affected by them by construction. In particular it is independent of
// Repetition Penalty (which uses historical playlist data, not edition
// context) and of Fit (Sound Diversity measures concentration against the
// edition's existing profiles, never a target-profile match).
//
// A nil ctx means no current-edition context was supplied: Value is nil.
// A non-nil ctx with zero Tracks means the edition has no existing
// concentration to diversify against — this is deliberately not treated as
// "maximally diverse": Value is nil, not 1.0, per Card #44's explicit
// convention that "no concentration yet" is not the same claim as "this
// candidate was evaluated as maximally diverse."
//
// Each of Artist/Era/Sound is computed independently and is Available only
// when the candidate supplies that dimension; Artist/Era/Sound are
// combined with equal 1/3 weight each, renormalized over whichever
// dimensions are Available (the same missing-data idiom Calculate and
// CalculateFit already use) — a missing dimension is excluded, never
// treated as 0.
func CalculateDiversity(candidateArtistSpotifyIDs []string, candidateEra *string, candidateSound musicaldna.Profile, ctx *CurrentEditionContext) DiversityResult {
	if ctx == nil {
		return DiversityResult{ContextProvided: false}
	}

	result := DiversityResult{
		ContextProvided: true,
		EditionSize:     len(ctx.Tracks),
	}

	if len(ctx.Tracks) == 0 {
		result.EditionEmpty = true
		return result
	}

	result.Artist = artistDiversity(candidateArtistSpotifyIDs, ctx.Tracks)
	result.Era = eraDiversity(candidateEra, ctx.Tracks)
	result.Sound = soundDiversity(candidateSound, ctx.Tracks)

	type weighted struct {
		value  *float64
		weight float64
	}
	aspects := []weighted{
		{result.Artist.Value, 1.0 / 3},
		{result.Era.Value, 1.0 / 3},
		{result.Sound.Value, 1.0 / 3},
	}

	var weightedSum, availableWeight float64
	for _, a := range aspects {
		if a.value != nil {
			weightedSum += *a.value * a.weight
			availableWeight += a.weight
		}
	}
	if availableWeight > 0 {
		value := weightedSum / availableWeight
		result.Value = &value
	}

	return result
}

func artistDiversity(candidateArtistSpotifyIDs []string, tracks []EditionTrack) DiversityAspectResult {
	if len(candidateArtistSpotifyIDs) == 0 {
		return DiversityAspectResult{Available: false}
	}

	occurrences := 0
	for _, t := range tracks {
		for _, id := range t.ArtistSpotifyIDs {
			if slices.Contains(candidateArtistSpotifyIDs, id) {
				occurrences++
				break
			}
		}
	}

	value := diversityContribution(occurrences)
	return DiversityAspectResult{Available: true, Occurrences: occurrences, Value: &value}
}

func eraDiversity(candidateEra *string, tracks []EditionTrack) DiversityAspectResult {
	if candidateEra == nil {
		return DiversityAspectResult{Available: false}
	}

	occurrences := 0
	for _, t := range tracks {
		if t.Era != nil && *t.Era == *candidateEra {
			occurrences++
		}
	}

	value := diversityContribution(occurrences)
	return DiversityAspectResult{Available: true, Occurrences: occurrences, Value: &value}
}

func soundDiversity(candidateSound musicaldna.Profile, tracks []EditionTrack) DiversitySoundResult {
	dims := []struct {
		name  DiversitySoundDimension
		value *string
		pick  func(musicaldna.Profile) *string
	}{
		{DiversitySoundMood, candidateSound.Mood, func(p musicaldna.Profile) *string { return p.Mood }},
		{DiversitySoundEnergy, candidateSound.Energy, func(p musicaldna.Profile) *string { return p.Energy }},
		{DiversitySoundTexture, candidateSound.Texture, func(p musicaldna.Profile) *string { return p.Texture }},
		{DiversitySoundCulturalInfluence, candidateSound.CulturalInfluence, func(p musicaldna.Profile) *string { return p.CulturalInfluence }},
	}

	dimResults := make([]DiversitySoundDimensionResult, 0, len(dims))
	var sum float64
	var count int
	for _, d := range dims {
		if d.value == nil {
			dimResults = append(dimResults, DiversitySoundDimensionResult{Dimension: d.name, Available: false})
			continue
		}

		occurrences := 0
		candidateVal := strings.ToLower(strings.TrimSpace(*d.value))
		for _, t := range tracks {
			other := d.pick(t.Sound)
			if other == nil {
				continue
			}
			if strings.ToLower(strings.TrimSpace(*other)) == candidateVal {
				occurrences++
			}
		}

		value := diversityContribution(occurrences)
		dimResults = append(dimResults, DiversitySoundDimensionResult{
			Dimension:   d.name,
			Available:   true,
			Occurrences: occurrences,
			Value:       &value,
		})
		sum += value
		count++
	}

	if count == 0 {
		return DiversitySoundResult{Available: false, Dimensions: dimResults}
	}
	value := sum / float64(count)
	return DiversitySoundResult{Available: true, Value: &value, Dimensions: dimResults}
}
