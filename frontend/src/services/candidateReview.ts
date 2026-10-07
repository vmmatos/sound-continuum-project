import type { CandidateReviewPool } from '../types/candidateReview'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'

export async function getCandidateReviewPool(): Promise<CandidateReviewPool | null> {
  try {
    const response = await fetch(`${API_BASE_URL}/api/candidates/review`)
    if (!response.ok) return null
    return (await response.json()) as CandidateReviewPool
  } catch {
    return null
  }
}

export async function keepCandidate(id: string): Promise<boolean> {
  try {
    const response = await fetch(`${API_BASE_URL}/api/candidates/${id}/keep`, { method: 'POST' })
    return response.ok
  } catch {
    return false
  }
}
