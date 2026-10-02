<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { checkHealth } from '../services/health'
import {
  getSpotifyStatus,
  startSpotifyAuth,
  initializeOfficialPlaylist,
  type SpotifyStatus,
  type OfficialPlaylist,
} from '../services/spotify'

const backendStatus = ref<'checking' | 'ok' | 'unreachable'>('checking')
const spotifyStatus = ref<SpotifyStatus | null>(null)
const spotifyRedirectOutcome = ref<string | null>(null)
const officialPlaylist = ref<OfficialPlaylist | null>(null)
const officialPlaylistFailed = ref(false)

async function createOfficialPlaylist() {
  officialPlaylistFailed.value = false
  const result = await initializeOfficialPlaylist()
  if (result) {
    officialPlaylist.value = result
  } else {
    officialPlaylistFailed.value = true
  }
}

onMounted(async () => {
  const health = await checkHealth()
  backendStatus.value = health?.status === 'ok' ? 'ok' : 'unreachable'

  spotifyRedirectOutcome.value = new URLSearchParams(window.location.search).get('spotify')
  spotifyStatus.value = await getSpotifyStatus()
})
</script>

<template>
  <header class="border-b border-border px-4 py-3">
    <div class="mx-auto flex max-w-3xl flex-wrap items-center justify-between gap-x-6 gap-y-1">
      <div>
        <h1 class="text-base font-semibold tracking-tight">Sound Continuum</h1>
        <p class="text-xs text-muted-foreground">Weekly music curation</p>
      </div>
      <p class="text-xs text-muted-foreground">Backend: {{ backendStatus }}</p>
    </div>

    <div class="mx-auto mt-2 max-w-3xl text-xs text-muted-foreground">
      <template v-if="spotifyRedirectOutcome === 'denied'">
        <p>Spotify authorization was cancelled.</p>
      </template>
      <template v-else-if="spotifyRedirectOutcome === 'error'">
        <p>Spotify authorization failed. Please try again.</p>
      </template>

      <template v-if="spotifyStatus?.status === 'connected'">
        <p>Spotify connected{{ spotifyStatus.display_name ? ` as ${spotifyStatus.display_name}` : '' }}.</p>
        <button
          class="mt-1 rounded border border-border px-2 py-1 text-xs text-muted-foreground"
          @click="createOfficialPlaylist"
        >
          Initialize official playlist
        </button>
        <p v-if="officialPlaylist">
          Official playlist: <a :href="officialPlaylist.url" target="_blank">{{ officialPlaylist.name }}</a>
        </p>
        <p v-if="officialPlaylistFailed">Failed to initialize the official playlist.</p>
      </template>
      <template v-else-if="spotifyStatus?.status === 'authorization_required'">
        <p>Spotify authorization expired.</p>
        <button
          class="mt-1 rounded border border-border px-2 py-1 text-xs text-muted-foreground"
          @click="startSpotifyAuth"
        >
          Reconnect Spotify
        </button>
      </template>
      <template v-else-if="spotifyStatus?.status === 'disconnected'">
        <button
          class="mt-1 rounded border border-border px-2 py-1 text-xs text-muted-foreground"
          @click="startSpotifyAuth"
        >
          Connect Spotify
        </button>
      </template>
    </div>
  </header>
</template>
