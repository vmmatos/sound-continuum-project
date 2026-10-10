import type { CandidateReviewEntry } from '../types/candidateReview'

// Minimal CandidateReviewEntry fixture shared by component/view specs.
export function makeEntry(id: string, status: CandidateReviewEntry['Ranked']['Candidate']['Status'], rank: number): CandidateReviewEntry {
  return {
    Ranked: {
      Rank: rank,
      Candidate: {
        ID: id,
        SpotifyTrackID: id,
        Source: 'Spotify',
        Category: 'Past',
        Type: 'Classic',
        Status: status,
        TrackTitle: `Track ${id}`,
        TrackArtist: `Artist ${id}`,
        CreatedAt: '',
        UpdatedAt: '',
        Metadata: null,
        Provenance: [],
      },
      Score: {
        CandidateID: id,
        ModelVersion: 'v1',
        Weights: { Fit: 0, Freshness: 0, DiscoveryBonus: 0, Diversity: 0, PlaylistFit: 0, RepetitionWeight: 0 },
        Factors: { Fit: null, Freshness: null, DiscoveryBonus: null, Diversity: null, PlaylistFit: null, RepetitionPenalty: null },
        FinalScore: null,
        AvailableWeight: 0,
      },
    },
    Explanation: { Text: '', Reasons: [] },
    Bridge: null,
    BridgeTrack: null,
  }
}
