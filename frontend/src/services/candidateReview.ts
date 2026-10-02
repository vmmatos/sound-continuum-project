import type { CandidateReviewPool } from '../types/candidateReview'

// TODO(#51 follow-up): no backend endpoint combines Rank + GenerateExplanation
// + DetectPotentialBridge today (only POST /api/candidates/pool exists, and
// it returns raw, unscored candidates). When that endpoint exists, replace
// the body below with:
//   const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'
//   const response = await fetch(`${API_BASE_URL}/api/candidates/review`)
//   if (!response.ok) return null
//   return (await response.json()) as CandidateReviewPool
// Nothing else in this file or CandidateReviewView.vue needs to change —
// the function signature and return type already match.
export async function getCandidateReviewPool(): Promise<CandidateReviewPool | null> {
  return mockCandidateReviewPool
}

const mockCandidateReviewPool: CandidateReviewPool = {
  EditionContext: 'Edition #42 — in progress',
  Entries: [
    {
      Ranked: {
        Rank: 1,
        Candidate: {
          ID: 'spotify:track:001',
          SpotifyTrackID: 'spotify:track:001',
          Source: 'Spotify',
          Category: 'Present',
          Type: 'Current',
          Status: 'discovered',
          TrackTitle: 'Night Signals',
          TrackArtist: 'Vela Monroe',
          CreatedAt: '2026-09-28T10:00:00Z',
          UpdatedAt: '2026-09-28T10:00:00Z',
          Metadata: {
            Title: 'Night Signals',
            Artists: [{ SpotifyArtistID: 'art-001', Name: 'Vela Monroe', SpotifyURL: 'https://open.spotify.com/artist/art-001' }],
            Album: {
              SpotifyAlbumID: 'alb-001',
              Name: 'Night Signals EP',
              AlbumType: 'single',
              ReleaseDate: '2026-09-01',
              ReleaseDatePrecision: 'day',
              Artwork: [{ URL: 'https://i.scdn.co/image/placeholder-001', Width: 300, Height: 300 }],
              SpotifyURL: 'https://open.spotify.com/album/alb-001',
            },
            DurationMS: 214000,
            Explicit: false,
            SpotifyURL: 'https://open.spotify.com/track/001',
            SpotifyURI: 'spotify:track:001',
          },
          Provenance: [
            {
              Method: 'current_reference_artist',
              Provider: 'Spotify',
              Seed: { Provider: 'Spotify', ProviderArtistID: 'art-ref-01', Name: 'Reference Artist A' },
              DiscoveredArtist: null,
              LastFMMatch: null,
            },
          ],
        },
        Score: {
          CandidateID: 'spotify:track:001',
          ModelVersion: 'v1',
          Weights: { Fit: 0.35, Freshness: 0.1, DiscoveryBonus: 0.15, Diversity: 0.15, PlaylistFit: 0.25, RepetitionWeight: 0.3 },
          Factors: { Fit: 0.91, Freshness: 1.0, DiscoveryBonus: null, Diversity: 0.8, PlaylistFit: 0.84, RepetitionPenalty: 0.1 },
          FinalScore: 0.87,
          AvailableWeight: 0.85,
        },
      },
      Explanation: {
        Text: 'Strong musical fit and freshness, with a small repetition penalty from recent artist history.',
        Reasons: ['strong_fit', 'fresh_playlist_history', 'repetition_penalty'],
      },
      Bridge: {
        PotentialBridge: true,
        EvidenceCount: 2,
        Dimensions: [
          { Dimension: 'energy', Available: true, Relationship: 'progression', Score: 0.7, Evidence: true },
          { Dimension: 'mood', Available: true, Relationship: 'matching', Score: 0.9, Evidence: true },
          { Dimension: 'texture', Available: true, Relationship: 'contrasting', Score: 0.4, Evidence: false },
          { Dimension: 'cultural_influence', Available: false, Relationship: 'unavailable', Score: null, Evidence: false },
        ],
        Signals: [
          { Signal: 'shared_artist', Available: true, Present: false },
          { Signal: 'lastfm_artist_similarity', Available: true, Present: true },
          { Signal: 'release_era', Available: true, Present: false },
        ],
      },
      BridgeTrack: 'Midnight Static',
    },
    {
      Ranked: {
        Rank: 2,
        Candidate: {
          ID: 'spotify:track:002',
          SpotifyTrackID: 'spotify:track:002',
          Source: 'Spotify',
          Category: 'Past',
          Type: 'Classic',
          Status: 'discovered',
          TrackTitle: 'Harbor Light',
          TrackArtist: 'The Coastline Four',
          CreatedAt: '2026-09-27T10:00:00Z',
          UpdatedAt: '2026-09-27T10:00:00Z',
          Metadata: {
            Title: 'Harbor Light',
            Artists: [{ SpotifyArtistID: 'art-002', Name: 'The Coastline Four', SpotifyURL: 'https://open.spotify.com/artist/art-002' }],
            Album: {
              SpotifyAlbumID: 'alb-002',
              Name: 'Harbor Light',
              AlbumType: 'album',
              ReleaseDate: '1978',
              ReleaseDatePrecision: 'year',
              Artwork: [{ URL: 'https://i.scdn.co/image/placeholder-002', Width: 300, Height: 300 }],
              SpotifyURL: 'https://open.spotify.com/album/alb-002',
            },
            DurationMS: 198000,
            Explicit: false,
            SpotifyURL: 'https://open.spotify.com/track/002',
            SpotifyURI: 'spotify:track:002',
          },
          Provenance: [
            {
              Method: 'classic_reference_artist',
              Provider: 'Spotify',
              Seed: { Provider: 'Spotify', ProviderArtistID: 'art-ref-02', Name: 'Reference Artist B' },
              DiscoveredArtist: null,
              LastFMMatch: null,
            },
          ],
        },
        Score: {
          CandidateID: 'spotify:track:002',
          ModelVersion: 'v1',
          Weights: { Fit: 0.35, Freshness: 0.1, DiscoveryBonus: 0.15, Diversity: 0.15, PlaylistFit: 0.25, RepetitionWeight: 0.3 },
          Factors: { Fit: 0.75, Freshness: 1.0, DiscoveryBonus: null, Diversity: 0.6, PlaylistFit: null, RepetitionPenalty: 0.0 },
          FinalScore: 0.78,
          AvailableWeight: 0.6,
        },
      },
      Explanation: {
        Text: 'Good diversity contribution and no recent repetition concerns.',
        Reasons: ['diversity_contribution', 'fresh_playlist_history'],
      },
      Bridge: null,
      BridgeTrack: null,
    },
    {
      Ranked: {
        Rank: 3,
        Candidate: {
          ID: 'spotify:track:003',
          SpotifyTrackID: 'spotify:track:003',
          Source: 'Spotify',
          Category: 'Emerging',
          Type: 'Discovery',
          Status: 'discovered',
          TrackTitle: 'Glass Orbit',
          TrackArtist: 'Paper Moons',
          CreatedAt: '2026-09-29T10:00:00Z',
          UpdatedAt: '2026-09-29T10:00:00Z',
          Metadata: {
            Title: 'Glass Orbit',
            Artists: [{ SpotifyArtistID: 'art-003', Name: 'Paper Moons', SpotifyURL: 'https://open.spotify.com/artist/art-003' }],
            Album: {
              SpotifyAlbumID: 'alb-003',
              Name: 'Glass Orbit',
              AlbumType: 'single',
              ReleaseDate: '2026-08-15',
              ReleaseDatePrecision: 'day',
              Artwork: [{ URL: 'https://i.scdn.co/image/placeholder-003', Width: 300, Height: 300 }],
              SpotifyURL: 'https://open.spotify.com/album/alb-003',
            },
            DurationMS: 231000,
            Explicit: false,
            SpotifyURL: 'https://open.spotify.com/track/003',
            SpotifyURI: 'spotify:track:003',
          },
          Provenance: [
            {
              Method: 'lastfm_similar_artist',
              Provider: 'Last.fm',
              Seed: { Provider: 'Spotify', ProviderArtistID: 'art-ref-03', Name: 'Reference Artist C' },
              DiscoveredArtist: { Provider: 'Spotify', ProviderArtistID: 'art-003', Name: 'Paper Moons' },
              LastFMMatch: 0.81,
            },
          ],
        },
        Score: {
          CandidateID: 'spotify:track:003',
          ModelVersion: 'v1',
          Weights: { Fit: 0.35, Freshness: 0.1, DiscoveryBonus: 0.15, Diversity: 0.15, PlaylistFit: 0.25, RepetitionWeight: 0.3 },
          Factors: { Fit: 0.62, Freshness: 1.0, DiscoveryBonus: 0.7, Diversity: 0.85, PlaylistFit: 0.55, RepetitionPenalty: 0.0 },
          FinalScore: 0.69,
          AvailableWeight: 1.0,
        },
      },
      Explanation: {
        Text: 'Meaningful discovery value and strong diversity contribution, with moderate musical fit.',
        Reasons: ['discovery_value', 'diversity_contribution'],
      },
      Bridge: {
        PotentialBridge: false,
        EvidenceCount: 1,
        Dimensions: [
          { Dimension: 'energy', Available: true, Relationship: 'contrasting', Score: 0.3, Evidence: false },
          { Dimension: 'mood', Available: true, Relationship: 'progression', Score: 0.6, Evidence: true },
          { Dimension: 'texture', Available: false, Relationship: 'unavailable', Score: null, Evidence: false },
          { Dimension: 'cultural_influence', Available: false, Relationship: 'unavailable', Score: null, Evidence: false },
        ],
        Signals: [
          { Signal: 'shared_artist', Available: true, Present: false },
          { Signal: 'lastfm_artist_similarity', Available: true, Present: false },
          { Signal: 'release_era', Available: true, Present: false },
        ],
      },
      BridgeTrack: 'Harbor Light',
    },
    {
      Ranked: {
        Rank: 4,
        Candidate: {
          ID: 'spotify:track:004',
          SpotifyTrackID: 'spotify:track:004',
          Source: 'Spotify',
          Category: 'Present',
          Type: 'Current',
          Status: 'discovered',
          TrackTitle: 'Quiet Traffic',
          TrackArtist: 'Ines Okafor',
          CreatedAt: '2026-09-26T10:00:00Z',
          UpdatedAt: '2026-09-26T10:00:00Z',
          Metadata: null,
          Provenance: [
            {
              Method: 'current_reference_artist',
              Provider: 'Spotify',
              Seed: { Provider: 'Spotify', ProviderArtistID: 'art-ref-04', Name: 'Reference Artist D' },
              DiscoveredArtist: null,
              LastFMMatch: null,
            },
          ],
        },
        Score: {
          CandidateID: 'spotify:track:004',
          ModelVersion: 'v1',
          Weights: { Fit: 0.35, Freshness: 0.1, DiscoveryBonus: 0.15, Diversity: 0.15, PlaylistFit: 0.25, RepetitionWeight: 0.3 },
          Factors: { Fit: 0.4, Freshness: 0.3, DiscoveryBonus: null, Diversity: 0.3, PlaylistFit: 0.35, RepetitionPenalty: 0.6 },
          FinalScore: 0.31,
          AvailableWeight: 0.85,
        },
      },
      Explanation: {
        Text: 'Weak fit and a meaningful repetition penalty from recent playlist history outweigh its modest freshness.',
        Reasons: ['repetition_penalty'],
      },
      Bridge: null,
      BridgeTrack: null,
    },
    {
      Ranked: {
        Rank: 5,
        Candidate: {
          ID: 'spotify:track:005',
          SpotifyTrackID: 'spotify:track:005',
          Source: 'Spotify',
          Category: 'Past',
          Type: 'Classic',
          Status: 'discovered',
          TrackTitle: 'Low Tide Avenue',
          TrackArtist: 'Marguerite Hall',
          CreatedAt: '2026-09-25T10:00:00Z',
          UpdatedAt: '2026-09-25T10:00:00Z',
          Metadata: {
            Title: 'Low Tide Avenue',
            Artists: [{ SpotifyArtistID: 'art-005', Name: 'Marguerite Hall', SpotifyURL: 'https://open.spotify.com/artist/art-005' }],
            Album: {
              SpotifyAlbumID: 'alb-005',
              Name: 'Low Tide Avenue',
              AlbumType: 'album',
              ReleaseDate: '1991-03',
              ReleaseDatePrecision: 'month',
              Artwork: [],
              SpotifyURL: 'https://open.spotify.com/album/alb-005',
            },
            DurationMS: 256000,
            Explicit: false,
            SpotifyURL: 'https://open.spotify.com/track/005',
            SpotifyURI: 'spotify:track:005',
          },
          Provenance: [
            {
              Method: 'classic_reference_artist',
              Provider: 'Spotify',
              Seed: { Provider: 'Spotify', ProviderArtistID: 'art-ref-05', Name: 'Reference Artist E' },
              DiscoveredArtist: null,
              LastFMMatch: null,
            },
          ],
        },
        Score: {
          CandidateID: 'spotify:track:005',
          ModelVersion: 'v1',
          Weights: { Fit: 0.35, Freshness: 0.1, DiscoveryBonus: 0.15, Diversity: 0.15, PlaylistFit: 0.25, RepetitionWeight: 0.3 },
          Factors: { Fit: null, Freshness: null, DiscoveryBonus: null, Diversity: null, PlaylistFit: null, RepetitionPenalty: null },
          FinalScore: null,
          AvailableWeight: 0,
        },
      },
      Explanation: {
        Text: 'Not enough signal is available yet to assess this candidate.',
        Reasons: ['no_signal'],
      },
      Bridge: null,
      BridgeTrack: null,
    },
  ],
}
