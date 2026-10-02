<script setup lang="ts">
import { computed, ref } from 'vue'
import type { CandidateAlbum } from '../types/candidateReview'

const props = defineProps<{
  title: string
  artist: string
  album: CandidateAlbum | null
}>()

const artworkUrl = computed(() => props.album?.Artwork[0]?.URL ?? null)
const artworkFailed = ref(false)
</script>

<template>
  <div class="flex items-center gap-3">
    <img
      v-if="artworkUrl && !artworkFailed"
      :src="artworkUrl"
      :alt="`${title} album artwork`"
      class="h-14 w-14 shrink-0 rounded-md object-cover"
      @error="artworkFailed = true"
    />
    <div v-else class="h-14 w-14 shrink-0 rounded-md border border-border bg-muted" aria-hidden="true" />

    <div class="min-w-0">
      <h3 class="truncate text-xl font-semibold tracking-tight">{{ title }}</h3>
      <p class="text-sm font-medium text-muted-foreground">{{ artist }}</p>
      <p v-if="album" class="text-xs text-muted-foreground/70">{{ album.Name }}</p>
    </div>
  </div>
</template>
