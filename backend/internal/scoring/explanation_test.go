package scoring

import (
	"reflect"
	"strings"
	"testing"
)

func scoreWithFactors(f Factors) CandidateScore {
	score, err := Calculate("cand-explanation", f, DefaultWeights())
	if err != nil {
		panic(err)
	}
	return score
}

func TestGenerateExplanationStrongFit(t *testing.T) {
	got := GenerateExplanation(ExplanationInput{Score: scoreWithFactors(Factors{Fit: float64Ptr(0.9)})})
	if !strings.Contains(got.Text, "musical fit") {
		t.Errorf("Text = %q, want it to reference musical fit", got.Text)
	}
	if !reasonsContain(got.Reasons, ReasonStrongFit) {
		t.Errorf("Reasons = %v, want %v", got.Reasons, ReasonStrongFit)
	}
}

func TestGenerateExplanationNeverUsedFreshness(t *testing.T) {
	got := GenerateExplanation(ExplanationInput{Score: scoreWithFactors(Factors{Freshness: float64Ptr(1.0)})})
	if !strings.Contains(got.Text, "playlist") {
		t.Errorf("Text = %q, want it to reference the playlist", got.Text)
	}
	if strings.Contains(got.Text, "fresh new release") || strings.Contains(got.Text, "release") {
		t.Errorf("Text = %q, must not describe release freshness", got.Text)
	}
	if !reasonsContain(got.Reasons, ReasonFreshPlaylistHistory) {
		t.Errorf("Reasons = %v, want %v", got.Reasons, ReasonFreshPlaylistHistory)
	}
}

func TestGenerateExplanationNotRecentlyUsedFreshness(t *testing.T) {
	// A used-but-long-ago candidate: high but not exactly 1.0 Freshness.
	got := GenerateExplanation(ExplanationInput{Score: scoreWithFactors(Factors{Freshness: float64Ptr(0.9)})})
	if !strings.Contains(got.Text, "recently") {
		t.Errorf("Text = %q, want it to describe not appearing recently", got.Text)
	}
	if !reasonsContain(got.Reasons, ReasonFreshPlaylistHistory) {
		t.Errorf("Reasons = %v, want %v", got.Reasons, ReasonFreshPlaylistHistory)
	}
}

func TestGenerateExplanationLowFreshnessOmitted(t *testing.T) {
	got := GenerateExplanation(ExplanationInput{Score: scoreWithFactors(Factors{Freshness: float64Ptr(0.2)})})
	if strings.Contains(got.Text, "playlist") || strings.Contains(got.Text, "recently") {
		t.Errorf("Text = %q, a low Freshness below the mention threshold should be silently omitted", got.Text)
	}
	if reasonsContain(got.Reasons, ReasonFreshPlaylistHistory) {
		t.Errorf("Reasons = %v, a low Freshness must not produce a reason", got.Reasons)
	}
}

func TestGenerateExplanationDiscoveryValue(t *testing.T) {
	got := GenerateExplanation(ExplanationInput{Score: scoreWithFactors(Factors{DiscoveryBonus: float64Ptr(0.8)})})
	if !strings.Contains(got.Text, "discovery") {
		t.Errorf("Text = %q, want it to reference discovery", got.Text)
	}
	if !reasonsContain(got.Reasons, ReasonDiscoveryValue) {
		t.Errorf("Reasons = %v, want %v", got.Reasons, ReasonDiscoveryValue)
	}
}

func TestGenerateExplanationDiversityContribution(t *testing.T) {
	got := GenerateExplanation(ExplanationInput{Score: scoreWithFactors(Factors{Diversity: float64Ptr(0.8)})})
	if !strings.Contains(got.Text, "diversity") {
		t.Errorf("Text = %q, want it to reference diversity", got.Text)
	}
	if !reasonsContain(got.Reasons, ReasonDiversityContribution) {
		t.Errorf("Reasons = %v, want %v", got.Reasons, ReasonDiversityContribution)
	}
}

func TestGenerateExplanationPlaylistFit(t *testing.T) {
	got := GenerateExplanation(ExplanationInput{Score: scoreWithFactors(Factors{PlaylistFit: float64Ptr(0.8)})})
	if !strings.Contains(got.Text, "sequence fit") {
		t.Errorf("Text = %q, want it to reference playlist/sequence fit", got.Text)
	}
	if !reasonsContain(got.Reasons, ReasonPlaylistFit) {
		t.Errorf("Reasons = %v, want %v", got.Reasons, ReasonPlaylistFit)
	}
}

func TestGenerateExplanationBridgeFromDimensionEvidence(t *testing.T) {
	bridge := BridgeResult{
		PotentialBridge: true,
		EvidenceCount:   2,
		Dimensions: []BridgeDimensionResult{
			{Dimension: BridgeDimensionEnergy, Available: true, Relationship: BridgeRelationshipProgression, Evidence: true},
		},
		Signals: []BridgeSignalResult{
			{Signal: BridgeSignalSharedArtist, Available: true, Present: true},
		},
	}
	got := GenerateExplanation(ExplanationInput{
		Score:       scoreWithFactors(Factors{}),
		Bridge:      &bridge,
		BridgeTrack: "Other Track",
	})
	if !strings.Contains(got.Text, "bridge") || !strings.Contains(got.Text, "Other Track") || !strings.Contains(got.Text, "energy") {
		t.Errorf("Text = %q, want a bridge mention naming the partner track and the energy evidence", got.Text)
	}
	if !reasonsContain(got.Reasons, ReasonPotentialBridge) {
		t.Errorf("Reasons = %v, want %v", got.Reasons, ReasonPotentialBridge)
	}
}

func TestGenerateExplanationBridgeFromSignalEvidenceOnly(t *testing.T) {
	bridge := BridgeResult{
		PotentialBridge: true,
		EvidenceCount:   2,
		Dimensions: []BridgeDimensionResult{
			{Dimension: BridgeDimensionMood, Available: false},
		},
		Signals: []BridgeSignalResult{
			{Signal: BridgeSignalSharedArtist, Available: true, Present: true},
			{Signal: BridgeSignalReleaseEra, Available: true, Present: true},
		},
	}
	got := GenerateExplanation(ExplanationInput{Score: scoreWithFactors(Factors{}), Bridge: &bridge})
	if !strings.Contains(got.Text, "shared artist") {
		t.Errorf("Text = %q, want it to fall back to the shared-artist signal when no dimension has evidence", got.Text)
	}
}

func TestGenerateExplanationNoBridgeWhenNotDetected(t *testing.T) {
	bridge := BridgeResult{PotentialBridge: false, EvidenceCount: 1}
	got := GenerateExplanation(ExplanationInput{Score: scoreWithFactors(Factors{Fit: float64Ptr(0.9)}), Bridge: &bridge})
	if strings.Contains(got.Text, "bridge") {
		t.Errorf("Text = %q, must not mention a bridge when PotentialBridge is false", got.Text)
	}
	if reasonsContain(got.Reasons, ReasonPotentialBridge) {
		t.Errorf("Reasons = %v, must not include %v", got.Reasons, ReasonPotentialBridge)
	}
}

func TestGenerateExplanationRepetitionPenalty(t *testing.T) {
	got := GenerateExplanation(ExplanationInput{Score: scoreWithFactors(Factors{
		Fit:               float64Ptr(0.9),
		RepetitionPenalty: float64Ptr(0.6),
	})})
	if !strings.Contains(got.Text, "repetition") {
		t.Errorf("Text = %q, want it to mention the repetition penalty", got.Text)
	}
	if strings.Contains(got.Text, "reject") || strings.Contains(got.Text, "should not be selected") {
		t.Errorf("Text = %q, must never describe repetition as an automatic rejection", got.Text)
	}
	if !reasonsContain(got.Reasons, ReasonRepetitionPenalty) {
		t.Errorf("Reasons = %v, want %v", got.Reasons, ReasonRepetitionPenalty)
	}
}

func TestGenerateExplanationSmallRepetitionPenaltyOmitted(t *testing.T) {
	got := GenerateExplanation(ExplanationInput{Score: scoreWithFactors(Factors{
		Fit:               float64Ptr(0.9),
		RepetitionPenalty: float64Ptr(0.05),
	})})
	if strings.Contains(got.Text, "repetition") {
		t.Errorf("Text = %q, a negligible repetition penalty should be silently omitted", got.Text)
	}
}

func TestGenerateExplanationMultipleFactorsStaysConcise(t *testing.T) {
	got := GenerateExplanation(ExplanationInput{Score: scoreWithFactors(Factors{
		Fit:            float64Ptr(0.95),
		PlaylistFit:    float64Ptr(0.9),
		Freshness:      float64Ptr(1.0),
		DiscoveryBonus: float64Ptr(0.85),
		Diversity:      float64Ptr(0.8),
	})})
	if len(got.Reasons) > explanationMaxFragments {
		t.Errorf("Reasons = %v (len %d), want at most %d fragments for conciseness", got.Reasons, len(got.Reasons), explanationMaxFragments)
	}
	// The three strongest factors (Fit, PlaylistFit, Freshness) must win over
	// the two weaker ones.
	if !reasonsContain(got.Reasons, ReasonStrongFit) || !reasonsContain(got.Reasons, ReasonPlaylistFit) || !reasonsContain(got.Reasons, ReasonFreshPlaylistHistory) {
		t.Errorf("Reasons = %v, want the three strongest factors to be included", got.Reasons)
	}
	if reasonsContain(got.Reasons, ReasonDiversityContribution) {
		t.Errorf("Reasons = %v, want the weakest factor (Diversity) dropped to stay concise", got.Reasons)
	}
}

func TestGenerateExplanationMissingFactorsNotDescribedNegatively(t *testing.T) {
	got := GenerateExplanation(ExplanationInput{Score: scoreWithFactors(Factors{Fit: float64Ptr(0.9)})})
	for _, bad := range []string{"low diversity", "weak", "poor", "0.00", "nil"} {
		if strings.Contains(strings.ToLower(got.Text), bad) {
			t.Errorf("Text = %q, must not describe a missing factor as %q", got.Text, bad)
		}
	}
	if reasonsContain(got.Reasons, ReasonDiversityContribution) {
		t.Errorf("Reasons = %v, a nil Diversity must not produce a reason", got.Reasons)
	}
}

func TestGenerateExplanationExplicitZeroOmittedNotNegative(t *testing.T) {
	// An explicit, assessed 0.0 (e.g. Discovery Bonus with no meaningful
	// value) must stay silently omitted, never phrased as a negative claim.
	got := GenerateExplanation(ExplanationInput{Score: scoreWithFactors(Factors{
		Fit:            float64Ptr(0.9),
		DiscoveryBonus: float64Ptr(0.0),
	})})
	if strings.Contains(got.Text, "discovery") {
		t.Errorf("Text = %q, an explicit zero Discovery Bonus must stay omitted, not mentioned negatively", got.Text)
	}
}

func TestGenerateExplanationNoSignalFallback(t *testing.T) {
	got := GenerateExplanation(ExplanationInput{Score: scoreWithFactors(Factors{})})
	if got.Text != "No strong scoring signal available." {
		t.Errorf("Text = %q, want the neutral no-signal fallback", got.Text)
	}
	if !reflect.DeepEqual(got.Reasons, []ExplanationReason{ReasonNoSignal}) {
		t.Errorf("Reasons = %v, want [%v]", got.Reasons, ReasonNoSignal)
	}
}

func TestGenerateExplanationRepetitionOnlyFallback(t *testing.T) {
	// No positive factor available/strong, but a meaningful penalty exists:
	// must still produce a grounded sentence, not the generic no-signal one.
	got := GenerateExplanation(ExplanationInput{Score: scoreWithFactors(Factors{RepetitionPenalty: float64Ptr(0.9)})})
	if !strings.Contains(got.Text, "repetition") {
		t.Errorf("Text = %q, want a repetition-only explanation", got.Text)
	}
	if got.Text == "No strong scoring signal available." {
		t.Error("Text = no-signal fallback, want a repetition-specific sentence since a real signal is present")
	}
}

func TestGenerateExplanationDeterministic(t *testing.T) {
	in := ExplanationInput{Score: scoreWithFactors(Factors{
		Fit:               float64Ptr(0.8),
		Freshness:         float64Ptr(1.0),
		RepetitionPenalty: float64Ptr(0.5),
	})}
	first := GenerateExplanation(in)
	second := GenerateExplanation(in)
	if first.Text != second.Text {
		t.Errorf("Text differs across identical calls: %q vs %q", first.Text, second.Text)
	}
	if !reflect.DeepEqual(first.Reasons, second.Reasons) {
		t.Errorf("Reasons differ across identical calls: %v vs %v", first.Reasons, second.Reasons)
	}
}

func TestGenerateExplanationDoesNotMutateScore(t *testing.T) {
	score := scoreWithFactors(Factors{Fit: float64Ptr(0.8)})
	before := score
	_ = GenerateExplanation(ExplanationInput{Score: score})
	if !reflect.DeepEqual(before, score) {
		t.Errorf("CandidateScore mutated by GenerateExplanation: before = %+v, after = %+v", before, score)
	}
}

func reasonsContain(reasons []ExplanationReason, want ExplanationReason) bool {
	for _, r := range reasons {
		if r == want {
			return true
		}
	}
	return false
}
