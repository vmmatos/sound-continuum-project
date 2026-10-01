# Musical Similarity — Research (Card #47)

Investigation only. No similarity engine, ranking logic, API endpoint, or
persistence is implemented by this card. This document asks a narrower
question than Card #46 (Playlist Fit): can *external metadata* (Spotify,
Last.fm) give additional evidence about a relationship between two tracks,
on top of what a human curator already supplies through `musicaldna.Profile`?
It does not touch, and does not propose changing, Card #46's model.

All claims about Spotify/Last.fm availability below are inherited from
[`docs/spotify-api.md`](../spotify-api.md) (Card 23, researched September
2026) — that research is treated as current and authoritative; it is not
repeated independently here. This document's own contribution is mapping
those already-established constraints onto the specific question of
track-to-track similarity, and onto what this codebase has already built
(`backend/internal/candidate`, `backend/internal/lastfm`,
`backend/internal/musicaldna`, `backend/internal/scoring`).

---

## 1. Spotify audio attributes

Explicitly unavailable, confirmed against `docs/spotify-api.md` §2.5 and
the M3 decision in `docs/memory/decisions.md`:

| Attribute | Available? |
|---|---|
| danceability | ❌ No |
| energy | ❌ No |
| valence | ❌ No |
| tempo | ❌ No |
| acousticness | ❌ No |
| instrumentalness | ❌ No |
| loudness | ❌ No |
| speechiness | ❌ No |
| key | ❌ No |
| mode | ❌ No |

Both `GET /audio-features/{id}` and `GET /audio-analysis/{id}` are gated
behind extended-quota access predating 2024-11-27. Sound Continuum is a new
app with no such access and no path to acquire it. This isn't new
information — M3's research already closed this door, and the
`musicaldna`/`scoring` packages were deliberately designed around its
absence (`docs/scoring-model.md`'s Fit section: "Qualities that would
require unavailable audio-level analysis... are left out"). Card #47 adds
nothing new here; it confirms the constraint still holds and that no
replacement has appeared since Card 23.

## 2. Artist overlap

What exists in code today: `candidate.CandidateArtist.SpotifyArtistID`
(Card #38) is already the artist-identity key `scoring.CalculateDiversity`
(Card #44, Artist Diversity) and `scoring.CalculateRepetitionPenalty` (Card
#45) both use.

**As an identity signal**: strong and reliable — a shared `SpotifyArtistID`
unambiguously means "the same artist." Useful for "same album/artist
context" questions (e.g. two tracks from the same release).

**As a similarity signal**: weak on its own. Two tracks by the same artist
can differ enormously in mood, energy, era, and style — an artist's
catalogue is not musically uniform, and nothing about shared artist ID says
*which* of those tracks are alike. Conversely, two tracks by different
artists can be a far stronger bridge than two tracks by the same artist.
Artist overlap is evidence of *context* (same creative source, same album),
not evidence of *musical resemblance between two specific tracks*.

This mirrors what Diversity and Repetition Penalty already do with the same
data: they use shared artist ID to measure *concentration/reuse*, never to
infer that two tracks sound alike. Card #47 finds no reason to build a
second, similarity-flavored use of the same identifier.

## 3. Genre metadata

- **Availability**: `spotify.Artist.Genres` (`[]string`) is already decoded
  (Card 29) and exposed wherever `Artist` appears.
- **Limitation — artist-level, not track-level**: Spotify genres describe
  an *artist's* overall classification, not an individual track. An
  artist tagged `"jazz"` can still release a track with no jazz character,
  and the field says nothing about which specific track is being evaluated.
- **Limitation — optional/deprecated**: Card 29's own decision already
  flags `genres` as "optional, deprecated metadata only... no genre
  normalization or mapping into Sound Continuum's musical DNA is built on
  top of it." Values are frequently `null` (confirmed live for at least one
  real artist, Card 29's verification). There is no taxonomy, no
  consistency guarantee, and no indication Spotify maintains this field
  with care going forward.
- **Reliability for automated similarity**: low. Genre strings are
  inconsistent in granularity (e.g. `"pop"` vs. `"portuguese fado"`), not
  hierarchical, and not guaranteed present. Treating genre-string overlap
  as a similarity score would fabricate precision the field doesn't
  support — the same reasoning that kept genre out of `musicaldna.Profile`
  in the first place (Card #41: "No fuzzy or embedding-based similarity is
  used... any partial-credit heuristic between arbitrary strings would be
  fabricated precision").

No genre taxonomy is proposed or needed here, per the card's own
constraint.

## 4. Last.fm similarity

What's integrated today: `backend/internal/lastfm/lastfm.go` wraps exactly
one method, `artist.getsimilar` (`Client.SimilarArtists`), returning
`[]SimilarArtist{Name, Match}` where `Match` is Last.fm's own `[0,1]`
similarity value. It's used by `discovery.DiscoverEmerging` as a
single-hop discovery signal and carried onward as
`candidate.DiscoveryProvenance.LastFMMatch` — documented since Cards #39
and #43 as "discovery metadata only, never a ranking signal."

**What this tells us about two tracks**: `artist.getsimilar`'s `Match` is
artist-to-artist, derived from Last.fm scrobble co-occurrence ("people who
played X also played Y"), not from either track's audio content
(`docs/spotify-api.md` §3.2/§3.3). It can support a claim like "these two
tracks' artists are perceived as related by Last.fm's listener base" — a
cultural/contextual signal. It cannot support "these two specific tracks
sound alike," and it says nothing when the two tracks share an artist
(`Match` is undefined for an artist against itself).

**What's available but unintegrated**: `track.getSimilar` (track-to-track
match percentages) and `tag.*` (folksonomy tags, also available
unauthenticated per Card 23's research) would be directly relevant to a
track-pair question — `track.getSimilar` is the closest thing either API
offers to "how related are these two specific tracks." Neither is called by
any code in this repository today. Building either would be new Last.fm
integration surface (a new `lastfm.Client` method, new error handling, new
tests) — not a trivial addition, and not justified by this investigation
alone (see §8).

Even if integrated, `track.getSimilar`'s match percentage would carry the
same caveat as `artist.getSimilar`'s: it's a listening-co-occurrence proxy,
not an acoustic-similarity measurement, and per this project's existing,
firm convention (`LastFMMatch` "never a ranking signal"), it should not
flow into `scoring.Factors` as a score input without a new, explicitly
justified decision.

## 5. Release / era relationship

`candidate.CandidateAlbum.ReleaseDate`/`ReleaseDatePrecision` (Card #38)
already preserve exactly what Spotify reports, including partial precision
(year-only stays year-only — never upgraded or invented). `scoring.
CalculateDiversity`'s Era dimension (Card #44) already extracts a coarse
decade from this same field.

Temporal proximity between two tracks (same decade, close release dates)
is **contextual information only** — it can support "these feel like they
belong to the same moment" as a narrative/editorial framing device, but it
says nothing about rhythm, mood, instrumentation, or any musical property.
Two tracks released a year apart can be musically unrelated; two tracks
decades apart can be a strong bridge (the manifesto's entire "music is a
continuum" principle, §1, depends on exactly this). Card #44 already
reached this same conclusion for Diversity's purposes — this investigation
finds no reason to treat it differently for a similarity question.

## 6. Simple deterministic heuristics

Combining the signals above (shared artist, Last.fm artist `Match`, genre
overlap, era proximity) into one heuristic was considered. Per the card's
explicit instruction, **no numerical weights are proposed** — nothing in
this investigation demonstrates that any particular weighting would be
justified over another; doing so would be fabricating precision the way
Card #41 already explicitly refused to do for musical dimensions.

What *can* be said without inventing weights: each signal above is
independently inspectable and boolean/near-boolean in character (same
artist: yes/no; genre overlap: yes/no per shared string; era: same
decade or not). If a future card ever needs to combine them, the existing
scoring model's missing-data idiom — treat an unavailable signal as
excluded, never substituted with a worst-case value, renormalize over what
is present (`scoring.Calculate`, `CalculateFit`, `CalculateDiversity`,
`CalculatePlaylistFit` all already do this) — is the right pattern to
reuse, not reinvent. This is a note on *mechanism*, not a recommendation to
build the combination now (see §8).

## 7. Editorial similarity

The manifesto (§4) names musical bridges "the most important editorial
craft in the project" — not a byproduct of matching metadata. Per
`docs/spotify-api.md` §6, the only automatable bridge *inputs* available
from any API are: Last.fm similarity scores, Last.fm/Spotify tags and
genres, and Spotify catalog facts (era, album grouping) — none of them
acoustic-property bridges. The manifesto itself anticipates exactly the
kind of bridge no metadata signal above can detect:

- Similar rhythm, different genre — genre overlap would score this as
  *unrelated*; it is often a strong bridge.
- Similar energy, different instrumentation — no API signal observes
  energy or instrumentation at all (§1).
- Similar vocal character — not observable through any available signal.
- Shared cultural influence without shared genre or artist — partially
  reachable only through `musicaldna.Profile.CulturalInfluence`, which is
  explicit human input, never derived.
- Deliberate contrast followed by release — the opposite of "similar" by
  any metric above, yet explicitly a valid, strong editorial bridge
  (consistent with `scoring.CalculatePlaylistFit`'s own 0.5 "intentional
  contrast" baseline, Card #46).

None of §§2–5's signals can detect any of these. This is not a gap to be
closed by better metadata — it's the expected shape of the problem: the
manifesto's bridge dimensions were never claimed to be derivable from an
API, and M3's research already reached the same conclusion
(`docs/spotify-api.md` §6: "M5 cannot be 'compute a bridge score from API
data'... editorial-first, optionally informed by similarity/tag signals as
supporting context"). Card #47 confirms this holds specifically for the
track-pair similarity question, not just the general bridge-scoring
question M3 already answered.

## 8. Recommendation

**What signals are currently available?**
Shared Spotify artist identity (`SpotifyArtistID`), Spotify artist genres
(artist-level, optional/deprecated), release date + precision, Last.fm
artist similarity (`artist.getsimilar`, already integrated). Last.fm
`track.getSimilar`/`tag.*` are available via the API but not integrated
into this codebase.

**Which signals are useful?**
Shared artist identity (as identity/context — "same artist" or "same
album"), Last.fm artist similarity (as contextual evidence — "artists
perceived as related by listeners"), release era (as contextual framing).
All three are useful *as displayed context for a human curator*, not as
inputs to a computed score.

**Which signals are unreliable or too weak?**
Spotify genres for anything resembling precision (artist-level, sparse,
no taxonomy, frequently null). Shared artist alone as a similarity claim
(same artist ≠ similar tracks). Any listening-popularity figure
(`listeners`/`playcount`) as an editorial signal — confirmed already by
M3's research, reconfirmed here as equally inapplicable to similarity.

**Which signals should be avoided?**
Treating Last.fm `Match` (artist- or hypothetically track-level) as a
ranking/scoring input — this would cross a line this codebase has already
drawn twice (Cards #39, #43) and should not be crossed quietly here.
Fabricating a numeric track-similarity score from genre-string overlap or
any combination in §6 without a demonstrated justification for the
weighting.

**What could realistically be used in a future MVP similarity heuristic?**
If a future curator-facing UI (M6+) wants to show *supporting context*
next to a candidate — "same artist as [previous track]," "Last.fm artist
similarity: 0.82," "released in the same decade" — all three are available
today with no new integration work beyond UI/display logic. This is
presentation, not scoring: none of it should feed `scoring.Factors`,
`musicaldna.Profile`, or any ranking mechanism.

**What should remain editorial/human judgment?**
Everything the manifesto actually calls a bridge: rhythm, tempo, melody,
instrumentation, texture, vocal character, energy, production, mood,
cross-genre or contrast-based transitions. No available signal, singly or
combined, approximates any of these — this is unchanged from M3's finding
and §7 above confirms it holds for the narrower similarity question too.

**Is a similarity engine justified now, or should it remain deferred?**
**Deferred.** No signal investigated here is strong or precise enough to
justify automated scoring without materially misrepresenting its
reliability, and the manifesto's own bridge philosophy explicitly includes
relationships no available signal can detect. This matches the posture
this project already took for Last.fm-in-M3 (deferred to M4, Card 23) and
for audio-features-based bridging (ruled out entirely, Card 23/M3
decision) — consistent, not a new exception.
