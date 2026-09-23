const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'

export interface HealthStatus {
  status: string
}

export async function checkHealth(): Promise<HealthStatus | null> {
  try {
    const response = await fetch(`${API_BASE_URL}/health`)
    if (!response.ok) return null
    return (await response.json()) as HealthStatus
  } catch {
    return null
  }
}
