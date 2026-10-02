package scoring

import (
	"strings"

	"github.com/vmmatos/sound-continuum-project/internal/musicaldna"
)

// PlaylistFitDimension identifies one transition dimension considered when
// evaluating Playlist Fit. Drawn from musicaldna.Profile — the same four
// dimensions Fit (Card #41) and Diversity (Card #44) already use.
type PlaylistFitDimension string

const (
	PlaylistFitDimensionMood              PlaylistFitDimension = "mood"
	PlaylistFitDimensionEnergy            PlaylistFitDimension = "energy"
	PlaylistFitDimensionTexture           PlaylistFitDimension = "texture"
	PlaylistFitDimensionCulturalInfluence PlaylistFitDimension = "cultural_influence"
)

// PlaylistFitDimensionResult explains one dimension's contribution to the
// previous-track -> candidate transition. Available is false when either
// side lacks this dimension; Value is only meaningful when Available is
// true, and is this dimension's own [0,1] transition score (not yet
// weighted).
type PlaylistFitDimensionResult struct {
	Dimension PlaylistFitDimension
	Available bool
	Value     *float64
}

// PlaylistFitResult is the outcome of CalculatePlaylistFit: a normalized
// [0,1] Value (nil when no transition could be evaluated — never a
// fabricated 0.0 or 1.0) plus the structural facts needed to explain it.
//
// ContextProvided is false only when ctx itself was nil ("no edition
// context was wired up"). PreviousTrackIndex is nil whenever there is no
// transition anchor — both when ContextProvided is false and when
// ContextProvided is true but ctx.Tracks is empty ("the edition
// legitimately has nothing selected yet") — the two remain distinguishable
// via ContextProvided alone, without a redundant third bool: unlike
// Diversity's EditionEmpty (which exists because 1.0, "maximally diverse,"
// is a real competing value Diversity must not be confused with), Playlist
// Fit has no equivalent ambiguity — "no previous track" only ever means
// Value == nil. When non-nil, PreviousTrackIndex is always
// len(ctx.Tracks)-1 — the single immediately-preceding track, never any
// earlier one.
type PlaylistFitResult struct {
	Value              *float64
	ContextProvided    bool
	PreviousTrackIndex *int
	Dimensions         []PlaylistFitDimensionResult
}

// playlistFitDimensionWeight is each of the four transition dimensions'
// fixed, equal contribution (25%), renormalized over whichever dimensions
// are actually available. There is no PlaylistFitWeights config struct —
// v1 has no real configurable knob here, mirroring the Diversity (Card
// #44) precedent of fixed weights with no caller-supplied/validatable
// value.
const playlistFitDimensionWeight = 0.25

// playlistFitEnergyLevels is a small, fixed 5-level ordinal vocabulary used
// only to score Energy transitions — it does not touch musicaldna.Profile,
// which stays *string everywhere else in the codebase. Case-insensitive,
// trimmed lookup; a value outside this vocabulary falls back to
// playlistFitMatchOrBaseline, exactly like Mood/Texture/CulturalInfluence.
var playlistFitEnergyLevels = map[string]int{
	"very low":  1,
	"low":       2,
	"medium":    3,
	"high":      4,
	"very high": 5,
}

// playlistFitMatchOrBaseline is Mood/Texture/CulturalInfluence's transition
// scorer: a case-insensitive, trimmed exact match scores 1.0; any mismatch
// (both sides present, different values) scores a flat 0.5 baseline, not
// 0.0. This is deliberately not compareProfiles' (fit.go) exact-match-or-
// nothing rule: a baseline lets an "intentional contrast" on one dimension
// still contribute meaningfully to Playlist Fit, rather than being
// punished as if it were simply wrong — more-similar is never assumed to
// always be better.
func playlistFitMatchOrBaseline(a, b string) float64 {
	if strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b)) {
		return 1.0
	}
	return 0.5
}

// playlistFitEnergyScore is Energy's transition scorer. When both values
// parse into playlistFitEnergyLevels, the score is 1 - |levelDiff|/4 —
// symmetric and direction-agnostic (low->medium and medium->low score
// identically), so a transition toward higher energy is never
// automatically treated as better than one toward lower energy. Same
// level -> 1.0, adjacent -> 0.75, two apart -> 0.5, three apart -> 0.25,
// opposite ends -> 0.0. When either value falls outside the vocabulary,
// falls back to playlistFitMatchOrBaseline.
func playlistFitEnergyScore(a, b string) float64 {
	la, okA := playlistFitEnergyLevels[strings.ToLower(strings.TrimSpace(a))]
	lb, okB := playlistFitEnergyLevels[strings.ToLower(strings.TrimSpace(b))]
	if !okA || !okB {
		return playlistFitMatchOrBaseline(a, b)
	}
	diff := la - lb
	if diff < 0 {
		diff = -diff
	}
	return 1 - float64(diff)/4
}

// CalculatePlaylistFit computes the Playlist Fit factor: if candidateSound
// were added next, would it make the musical journey flow naturally from
// the track immediately preceding it in the edition currently being
// assembled? The sole transition anchor is the single most recently
// selected track (ctx.Tracks[len(ctx.Tracks)-1]) -> candidate — never
// every track in the edition, never an arbitrary insertion position.
// CalculatePlaylistFit never reorders, selects, or mutates ctx itself.
//
// ctx reuses scoring.CurrentEditionContext/EditionTrack (Card #44) as-is —
// not a second "current edition" representation, and EditionTrack gains no
// ID/title field for this card. Since Tracks is an ordinary Go slice,
// insertion order is already preserved, so the previous track is simply
// its last element.
//
// candidateSound/ctx are plain values — CalculatePlaylistFit takes no
// candidate.CandidateTrack and no other factor's calculated value, so it
// cannot be affected by CandidateType, Category, Fit, Freshness,
// DiscoveryBonus, Diversity, RepetitionPenalty, popularity, or historical
// (non-edition) playlist usage by construction.
//
// A nil ctx, or a non-nil ctx with zero Tracks, both mean no transition
// anchor exists: Value is nil in both cases ("no transition exists" is not
// "bad transition") — see PlaylistFitResult's doc comment for how the two
// remain distinguishable. A missing profile dimension on either side
// excludes that dimension rather than scoring it 0; weights are
// renormalized over whichever dimensions are available (the same
// missing-data idiom CalculateFit/CalculateDiversity already use). Value
// is nil only when no dimension was comparable at all.
func CalculatePlaylistFit(candidateSound musicaldna.Profile, ctx *CurrentEditionContext) PlaylistFitResult {
	if ctx == nil {
		return PlaylistFitResult{ContextProvided: false}
	}

	result := PlaylistFitResult{ContextProvided: true}
	if len(ctx.Tracks) == 0 {
		return result
	}

	idx := len(ctx.Tracks) - 1
	result.PreviousTrackIndex = &idx
	previous := ctx.Tracks[idx].Sound

	dims := []struct {
		name  PlaylistFitDimension
		a, b  *string
		score func(a, b string) float64
	}{
		{PlaylistFitDimensionMood, candidateSound.Mood, previous.Mood, playlistFitMatchOrBaseline},
		{PlaylistFitDimensionEnergy, candidateSound.Energy, previous.Energy, playlistFitEnergyScore},
		{PlaylistFitDimensionTexture, candidateSound.Texture, previous.Texture, playlistFitMatchOrBaseline},
		{PlaylistFitDimensionCulturalInfluence, candidateSound.CulturalInfluence, previous.CulturalInfluence, playlistFitMatchOrBaseline},
	}

	dimResults := make([]PlaylistFitDimensionResult, 0, len(dims))
	items := make([]weightedValue, 0, len(dims))
	for _, d := range dims {
		if d.a == nil || d.b == nil {
			dimResults = append(dimResults, PlaylistFitDimensionResult{Dimension: d.name, Available: false})
			continue
		}
		v := d.score(*d.a, *d.b)
		dimResults = append(dimResults, PlaylistFitDimensionResult{Dimension: d.name, Available: true, Value: &v})
		items = append(items, weightedValue{&v, playlistFitDimensionWeight})
	}
	result.Dimensions = dimResults
	result.Value, _ = weightedAverage(items)
	return result
}
