import type { CandidateReviewEntry } from '../types/candidateReview'
import { API_BASE_URL } from './apiBase'

// Persists the curator's confirmed, ordered playlist as the backend Edition
// model's confirmed snapshot (Card #139) — the integration point that makes
// Card #61's confirmedPlaylist durable instead of frontend-memory-only.
// Sends each entry's existing SpotifyTrackID/TrackTitle/TrackArtist/Metadata
// directly, exactly as already held on CandidateReviewEntry — the backend
// never re-derives order or metadata from a fresh pool/rank call.
export async function confirmEdition(entries: CandidateReviewEntry[]): Promise<boolean> {
  try {
    const response = await fetch(`${API_BASE_URL}/api/editions/confirm`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        Tracks: entries.map((e) => ({
          SpotifyTrackID: e.Ranked.Candidate.SpotifyTrackID,
          TrackTitle: e.Ranked.Candidate.TrackTitle,
          TrackArtist: e.Ranked.Candidate.TrackArtist,
          Metadata: e.Ranked.Candidate.Metadata,
        })),
      }),
    })
    return response.ok
  } catch {
    return false
  }
}
