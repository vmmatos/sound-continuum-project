package candidate

// DiscoveryMethod identifies how a candidate entered Sound Continuum's
// discovery pipeline — distinct from Source, which identifies which
// provider supplied the candidate track itself. Add a new constant here
// only once a workflow actually constructs provenance with it.
type DiscoveryMethod string

const (
	DiscoveryMethodClassicReferenceArtist DiscoveryMethod = "classic_reference_artist"
	DiscoveryMethodCurrentReferenceArtist DiscoveryMethod = "current_reference_artist"
	DiscoveryMethodLastFMSimilarArtist    DiscoveryMethod = "lastfm_similar_artist"
	DiscoveryMethodManual                 DiscoveryMethod = "manual"
)

// Valid reports whether m is one of the supported discovery methods.
func (m DiscoveryMethod) Valid() bool {
	switch m {
	case DiscoveryMethodClassicReferenceArtist, DiscoveryMethodCurrentReferenceArtist,
		DiscoveryMethodLastFMSimilarArtist, DiscoveryMethodManual:
		return true
	default:
		return false
	}
}

// ProvenanceProvider identifies which external provider participated in a
// discovery step — distinct from Source, which identifies the provider of
// the candidate track itself. Empty is valid: it means no provider was
// involved (e.g. DiscoveryMethodManual) or no stable provider ID is
// available for a SeedArtist.
type ProvenanceProvider string

const (
	ProvenanceProviderSpotify ProvenanceProvider = "Spotify"
	ProvenanceProviderLastFM  ProvenanceProvider = "Last.fm"
)

// SeedArtist identifies an artist or entity involved in discovery, with a
// stable provider identifier where the discovery workflow already has one —
// never fabricated. Provider/ProviderArtistID stay empty when no such ID
// was resolved (e.g. an Emerging seed artist's name, never itself resolved
// through Spotify).
type SeedArtist struct {
	Provider         ProvenanceProvider
	ProviderArtistID string
	Name             string
}

// DiscoveryProvenance records one path by which a candidate entered the
// discovery pipeline. A candidate may carry more than one — see
// CandidateTrack.Provenance and MergeProvenance.
type DiscoveryProvenance struct {
	Method DiscoveryMethod
	// Provider is the provider queried for this discovery step (e.g.
	// Last.fm for a similarity lookup) — empty for DiscoveryMethodManual.
	Provider ProvenanceProvider
	// Seed is the artist/entity that triggered discovery; nil for
	// DiscoveryMethodManual. For classic_reference_artist/
	// current_reference_artist this is the reference artist itself. For
	// lastfm_similar_artist this is the original Emerging reference-artist
	// seed (see DiscoveredArtist for the artist Last.fm returned).
	Seed *SeedArtist
	// DiscoveredArtist is set only for lastfm_similar_artist: the artist
	// Last.fm's similarity lookup returned, resolved through Spotify.
	DiscoveredArtist *SeedArtist
	// LastFMMatch is Last.fm's own similarity score, carried as discovery
	// metadata only — never a ranking signal. Set only for
	// lastfm_similar_artist.
	LastFMMatch *float64
}

// seedKey returns a's identity for MergeProvenance's dedup — the provider
// artist ID when available (stable), falling back to the name.
func (a *SeedArtist) seedKey() string {
	if a == nil {
		return ""
	}
	if a.ProviderArtistID != "" {
		return string(a.Provider) + ":" + a.ProviderArtistID
	}
	return "name:" + a.Name
}

// key returns a dedup identity for one DiscoveryProvenance entry: same
// method plus same seed/discovered-artist identity means the same
// discovery path, even if it was recorded by two different workflow runs.
func (p DiscoveryProvenance) key() string {
	return string(p.Method) + "|" + p.Seed.seedKey() + "|" + p.DiscoveredArtist.seedKey()
}

// MergeProvenance combines two candidates' provenance when Candidate Pool
// merges the same Spotify track found by more than one discovery workflow.
// Deduplication removes duplicate candidates, not useful provenance: both
// inputs' entries are kept, in order, with exact duplicates (by key)
// collapsed to their first occurrence.
func MergeProvenance(a, b []DiscoveryProvenance) []DiscoveryProvenance {
	seen := make(map[string]bool, len(a)+len(b))
	merged := make([]DiscoveryProvenance, 0, len(a)+len(b))
	for _, p := range a {
		if k := p.key(); !seen[k] {
			seen[k] = true
			merged = append(merged, p)
		}
	}
	for _, p := range b {
		if k := p.key(); !seen[k] {
			seen[k] = true
			merged = append(merged, p)
		}
	}
	return merged
}
