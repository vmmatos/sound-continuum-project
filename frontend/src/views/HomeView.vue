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
  <main>
    <h1>Sound Continuum</h1>
    <p>Foundation for the weekly music curation workflow.</p>
    <p class="backend-status">Backend: {{ backendStatus }}</p>

    <section class="spotify">
      <template v-if="spotifyRedirectOutcome === 'denied'">
        <p>Spotify authorization was cancelled.</p>
      </template>
      <template v-else-if="spotifyRedirectOutcome === 'error'">
        <p>Spotify authorization failed. Please try again.</p>
      </template>

      <template v-if="spotifyStatus?.status === 'connected'">
        <p>Spotify connected{{ spotifyStatus.display_name ? ` as ${spotifyStatus.display_name}` : '' }}.</p>
        <button @click="createOfficialPlaylist">Initialize official playlist</button>
        <p v-if="officialPlaylist">
          Official playlist: <a :href="officialPlaylist.url" target="_blank">{{ officialPlaylist.name }}</a>
        </p>
        <p v-if="officialPlaylistFailed">Failed to initialize the official playlist.</p>
      </template>
      <template v-else-if="spotifyStatus?.status === 'authorization_required'">
        <p>Spotify authorization expired.</p>
        <button @click="startSpotifyAuth">Reconnect Spotify</button>
      </template>
      <template v-else-if="spotifyStatus?.status === 'disconnected'">
        <button @click="startSpotifyAuth">Connect Spotify</button>
      </template>
    </section>
  </main>
</template>

<style scoped>
main {
  text-align: center;
  padding: 4rem 1rem;
}

.backend-status {
  color: var(--text-h, #666);
  font-size: 0.9rem;
}

.spotify {
  margin-top: 2rem;
}
</style>
