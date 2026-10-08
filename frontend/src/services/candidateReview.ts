import type { CandidateReviewPool } from '../types/candidateReview'
import { API_BASE_URL } from './apiBase'

export async function getCandidateReviewPool(): Promise<CandidateReviewPool | null> {
  try {
    const response = await fetch(`${API_BASE_URL}/api/candidates/review`)
    if (!response.ok) return null
    return (await response.json()) as CandidateReviewPool
  } catch {
    return null
  }
}

async function postAction(path: string): Promise<boolean> {
  try {
    const response = await fetch(`${API_BASE_URL}${path}`, { method: 'POST' })
    return response.ok
  } catch {
    return false
  }
}

export const keepCandidate = (id: string) => postAction(`/api/candidates/${id}/keep`)
export const maybeCandidate = (id: string) => postAction(`/api/candidates/${id}/maybe`)
export const skipCandidate = (id: string) => postAction(`/api/candidates/${id}/skip`)
export const clearCandidateDecision = (id: string) => postAction(`/api/candidates/${id}/clear`)
