package scoring

import (
	"sort"
	"strings"
)

// ExplanationReason tags one piece of evidence a CandidateExplanation's Text
// was built from — structured, machine-readable detail retained alongside
// the prose for possible future UI use (e.g. icons per reason), without
// requiring a reader to parse Text.
type ExplanationReason string

const (
	ReasonStrongFit             ExplanationReason = "strong_fit"
	ReasonFreshPlaylistHistory  ExplanationReason = "fresh_playlist_history"
	ReasonDiscoveryValue        ExplanationReason = "discovery_value"
	ReasonDiversityContribution ExplanationReason = "diversity_contribution"
	ReasonPlaylistFit           ExplanationReason = "playlist_fit"
	ReasonPotentialBridge       ExplanationReason = "potential_bridge"
	ReasonRepetitionPenalty     ExplanationReason = "repetition_penalty"
	ReasonNoSignal              ExplanationReason = "no_signal"
)

// CandidateExplanation is a short, deterministic, human-readable account of
// why a CandidateScore looks the way it does. It describes the existing
// scoring result; it is never an independent editorial judgement, never a
// new scoring factor, and generating it never changes FinalScore or
// ranking — see docs/scoring-model.md.
type CandidateExplanation struct {
	Text    string
	Reasons []ExplanationReason
}

// ExplanationInput bundles a candidate's already-computed CandidateScore
// with an optional potential-bridge result for the same candidate pair
// (Card #48's DetectPotentialBridge output, kept outside Factors/Calculate
// by design — see bridge.go). BridgeTrack is an optional display name for
// the other track in the pair (e.g. "Title — Artist"), used only when
// Bridge is non-nil and Bridge.PotentialBridge is true.
type ExplanationInput struct {
	Score       CandidateScore
	Bridge      *BridgeResult
	BridgeTrack string
}

// explanationMentionThreshold is the factor value at or above which a
// positive factor is considered meaningful enough to mention. Below it, the
// factor is silently omitted — never described as weak, and a nil factor is
// never described as zero (see Factors' own missing-data convention).
const explanationMentionThreshold = 0.6

// explanationRepetitionThreshold is the RepetitionPenalty value at or above
// which the penalty is worth mentioning as a caveat.
const explanationRepetitionThreshold = 0.3

// explanationMaxFragments bounds how many positive-factor fragments appear
// in the main clause: prefer explaining the factors that materially
// contribute over mechanically listing every one.
const explanationMaxFragments = 3

type explanationFragment struct {
	reason ExplanationReason
	phrase string
	value  float64
	order  int
}

// GenerateExplanation builds a short, deterministic explanation for
// in.Score, optionally informed by a potential-bridge result for the same
// candidate. It never recomputes or mutates in.Score, never selects or
// rejects the candidate, and for identical inputs always returns identical
// output.
func GenerateExplanation(in ExplanationInput) CandidateExplanation {
	f := in.Score.Factors

	var fragments []explanationFragment
	if f.Fit != nil && *f.Fit >= explanationMentionThreshold {
		fragments = append(fragments, explanationFragment{ReasonStrongFit, "strong musical fit", *f.Fit, 0})
	}
	if f.PlaylistFit != nil && *f.PlaylistFit >= explanationMentionThreshold {
		fragments = append(fragments, explanationFragment{ReasonPlaylistFit, "a promising sequence fit", *f.PlaylistFit, 1})
	}
	if f.Freshness != nil {
		switch {
		case *f.Freshness == 1.0:
			fragments = append(fragments, explanationFragment{ReasonFreshPlaylistHistory, "new to the Sound Continuum playlist", *f.Freshness, 2})
		case *f.Freshness >= explanationMentionThreshold:
			fragments = append(fragments, explanationFragment{ReasonFreshPlaylistHistory, "has not appeared recently in Sound Continuum", *f.Freshness, 2})
		}
	}
	if f.DiscoveryBonus != nil && *f.DiscoveryBonus >= explanationMentionThreshold {
		fragments = append(fragments, explanationFragment{ReasonDiscoveryValue, "emerging discovery with strong editorial value", *f.DiscoveryBonus, 3})
	}
	if f.Diversity != nil && *f.Diversity >= explanationMentionThreshold {
		fragments = append(fragments, explanationFragment{ReasonDiversityContribution, "a strong diversity contribution", *f.Diversity, 4})
	}

	sortFragments(fragments)
	if len(fragments) > explanationMaxFragments {
		fragments = fragments[:explanationMaxFragments]
	}

	var reasons []ExplanationReason
	var phrases []string
	for _, fr := range fragments {
		phrases = append(phrases, fr.phrase)
		reasons = append(reasons, fr.reason)
	}

	if in.Bridge != nil && in.Bridge.PotentialBridge {
		phrases = append(phrases, bridgePhrase(*in.Bridge, in.BridgeTrack))
		reasons = append(reasons, ReasonPotentialBridge)
	}

	repetitionPhrase := ""
	if f.RepetitionPenalty != nil && *f.RepetitionPenalty >= explanationRepetitionThreshold {
		repetitionPhrase = "a repetition penalty from recent playlist history"
		reasons = append(reasons, ReasonRepetitionPenalty)
	}

	text := joinWithAnd(phrases)
	switch {
	case text == "" && repetitionPhrase == "":
		return CandidateExplanation{Text: "No strong scoring signal available.", Reasons: []ExplanationReason{ReasonNoSignal}}
	case text == "":
		text = repetitionPhrase + " reduces the score"
	case repetitionPhrase != "":
		text += ", with " + repetitionPhrase
	}

	return CandidateExplanation{Text: capitalize(text) + ".", Reasons: reasons}
}

// sortFragments orders fragments by factor value descending, breaking ties
// on the fixed editorial priority order (order field) so the result is
// deterministic regardless of input order.
func sortFragments(fragments []explanationFragment) {
	sort.SliceStable(fragments, func(i, j int) bool {
		if fragments[i].value != fragments[j].value {
			return fragments[i].value > fragments[j].value
		}
		return fragments[i].order < fragments[j].order
	})
}

// bridgePhrase describes a detected potential bridge from the first piece of
// evidence BridgeResult actually reports — a dimension with Evidence true,
// falling back to a contextual signal with Present true. PotentialBridge
// true guarantees at least DefaultMinimumBridgeEvidence such entries exist.
func bridgePhrase(b BridgeResult, partner string) string {
	desc := bridgeEvidenceDescription(b)
	if partner != "" {
		return "a potential bridge to " + partner + " through " + desc
	}
	return "a potential bridge through " + desc
}

func bridgeEvidenceDescription(b BridgeResult) string {
	for _, d := range b.Dimensions {
		if !d.Evidence {
			continue
		}
		switch d.Dimension {
		case BridgeDimensionMood:
			return "a shared mood"
		case BridgeDimensionEnergy:
			return "a shared energy profile"
		case BridgeDimensionTexture:
			return "a shared texture"
		case BridgeDimensionCulturalInfluence:
			return "a shared cultural influence"
		}
	}
	for _, s := range b.Signals {
		if !s.Present {
			continue
		}
		switch s.Signal {
		case BridgeSignalSharedArtist:
			return "a shared artist"
		case BridgeSignalReleaseEra:
			return "a shared release era"
		case BridgeSignalLastFMArtistSimilarity:
			return "related artist similarity"
		}
	}
	return "shared musical evidence"
}

func joinWithAnd(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	default:
		return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
