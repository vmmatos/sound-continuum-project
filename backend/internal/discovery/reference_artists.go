package discovery

import "strings"

// PastReferenceArtists is Sound Continuum's canonical "Past" category
// reference-artist list (see docs/manifesto.md). Curator-edited data, not
// derived from any Spotify query — the single source of truth for classic
// discovery's starting points.
var PastReferenceArtists = []string{
	"David Bowie", "Prince", "Kate Bush", "Kraftwerk", "Talking Heads",
	"New Order", "Fela Kuti", "Massive Attack", "A Tribe Called Quest",
	"Aphex Twin", "Björk", "Daft Punk", "Radiohead", "Nina Simone",
	"Cocteau Twins",
}

// PresentReferenceArtists is Sound Continuum's canonical "Present" category
// reference-artist list (see docs/manifesto.md). Curator-edited data, not
// derived from any Spotify query — the single source of truth for current
// discovery's starting points.
var PresentReferenceArtists = []string{
	"Fred again.", "James Blake", "FKA twigs", "Jai Paul", "Fontaines D.C.",
	"Little Simz", "Sampha", "Caroline Polachek", "Yaeji", "Peggy Gou",
	"Rosalía", "Arca", "The Smile", "Kelela", "Blood Orange",
}

// EmergingReferenceArtists is Sound Continuum's canonical "Emerging"
// category reference-artist list (see docs/memory/decisions.md, Card #35).
// Curator-edited data, not derived from any Spotify or Last.fm query — the
// single source of truth for emerging discovery's Last.fm seeds.
var EmergingReferenceArtists = []string{
	"The Twins", "Toxe", "Helena Gao", "Florence Road", "XCOMM",
	"Sasha Keable", "Effie", "waterbaby", "Phoenix James", "Pz'",
	"Amil Raja", "Judah Weston", "IsoKeys", "earthsignchels", "girlsweetvoiced",
}

// isCanonicalReferenceArtist reports whether name matches (case-insensitive,
// trimmed) any artist already in Past, Present, or Emerging's own canonical
// reference lists. Used to exclude reference artists themselves from
// emerging discovery's Last.fm results.
func isCanonicalReferenceArtist(name string) bool {
	want := strings.ToLower(strings.TrimSpace(name))
	for _, list := range [][]string{PastReferenceArtists, PresentReferenceArtists, EmergingReferenceArtists} {
		for _, candidate := range list {
			if strings.ToLower(strings.TrimSpace(candidate)) == want {
				return true
			}
		}
	}
	return false
}
