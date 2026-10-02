<script setup lang="ts">
import { computed } from 'vue'
import type { BridgeDimension, BridgeResult, BridgeSignal } from '../types/candidateReview'

const props = defineProps<{
  bridgeTrack: string | null
  candidateTitle: string
  bridge: BridgeResult
}>()

const dimensionLabel: Record<BridgeDimension, string> = {
  mood: 'Mood',
  energy: 'Energy',
  texture: 'Texture',
  cultural_influence: 'Cultural influence',
}

const signalLabel: Record<BridgeSignal, string> = {
  shared_artist: 'Shared artist',
  lastfm_artist_similarity: 'Context',
  release_era: 'Release era',
}

const signalValue: Record<BridgeSignal, string> = {
  shared_artist: 'Shared artist between tracks',
  lastfm_artist_similarity: 'Related artist (Last.fm)',
  release_era: 'Similar release era',
}

const rows = computed(() => [
  ...props.bridge.Dimensions.filter((d) => d.Evidence).map((d) => ({
    label: dimensionLabel[d.Dimension],
    value: d.Relationship,
  })),
  ...props.bridge.Signals.filter((s) => s.Present).map((s) => ({
    label: signalLabel[s.Signal],
    value: signalValue[s.Signal],
  })),
])
</script>

<template>
  <section>
    <p class="text-[0.65rem] font-semibold uppercase tracking-widest text-muted-foreground">Musical bridge</p>
    <p v-if="bridgeTrack" class="mt-1 text-sm">
      {{ bridgeTrack }} <span class="text-muted-foreground">→</span> {{ candidateTitle }}
    </p>
    <div class="mt-2 flex flex-col gap-1">
      <div v-for="row in rows" :key="row.label" class="flex justify-between text-xs">
        <span class="text-muted-foreground">{{ row.label }}</span>
        <span>{{ row.value }}</span>
      </div>
    </div>
  </section>
</template>
