package candidate

// CandidateMetadata is the editorial/display metadata attached to a
// CandidateTrack for later ranking, UI display, and curation (Card #38).
// It is Sound Continuum's own, intentionally small metadata model — not a
// copy of any provider's full response. A candidate's Metadata is nil until
// enrichment runs, and is never populated with fabricated/placeholder
// values: nil always means "not enriched," never "enriched with nothing."
type CandidateMetadata struct {
	Title      string
	Artists    []CandidateArtist
	Album      CandidateAlbum
	DurationMS int
	Explicit   bool
	SpotifyURL string
	SpotifyURI string
}

// CandidateArtist is one performing artist on a candidate, kept as
// structured identity rather than folded into a single display string —
// later editorial logic may need artist identity, not just display text.
type CandidateArtist struct {
	SpotifyArtistID string
	Name            string
	SpotifyURL      string
}

// CandidateAlbum is the album a candidate track appears on. ReleaseDate and
// ReleaseDatePrecision preserve exactly what the provider reported (e.g. a
// year-only release date stays year-only) — precision is never upgraded or
// invented.
type CandidateAlbum struct {
	SpotifyAlbumID       string
	Name                 string
	AlbumType            string
	ReleaseDate          string
	ReleaseDatePrecision string
	Artwork              []CandidateImage
	SpotifyURL           string
}

// CandidateImage is one piece of artwork metadata (e.g. one size variant of
// an album's cover art). Sound Continuum never downloads, transforms, or
// stores the underlying image — only its provider-supplied URL/dimensions.
type CandidateImage struct {
	URL    string
	Width  int
	Height int
}
