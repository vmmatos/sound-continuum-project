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

export async function keepCandidate(id: string): Promise<boolean> {
  try {
    const response = await fetch(`${API_BASE_URL}/api/candidates/${id}/keep`, { method: 'POST' })
    return response.ok
  } catch {
    return false
  }
}

export async function maybeCandidate(id: string): Promise<boolean> {
  try {
    const response = await fetch(`${API_BASE_URL}/api/candidates/${id}/maybe`, { method: 'POST' })
    return response.ok
  } catch {
    return false
  }
}

export async function clearCandidateDecision(id: string): Promise<boolean> {
  try {
    const response = await fetch(`${API_BASE_URL}/api/candidates/${id}/clear`, { method: 'POST' })
    return response.ok
  } catch {
    return false
  }
}
