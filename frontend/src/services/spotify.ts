const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'

export type SpotifyConnectionStatus = 'connected' | 'disconnected' | 'authorization_required'

export interface SpotifyStatus {
  status: SpotifyConnectionStatus
  display_name?: string
}

export async function getSpotifyStatus(): Promise<SpotifyStatus | null> {
  try {
    const response = await fetch(`${API_BASE_URL}/api/spotify/status`)
    if (!response.ok) return null
    return (await response.json()) as SpotifyStatus
  } catch {
    return null
  }
}

export function startSpotifyAuth(): void {
  window.location.href = `${API_BASE_URL}/api/spotify/auth`
}
