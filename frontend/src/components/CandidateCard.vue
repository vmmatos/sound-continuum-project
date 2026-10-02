<script setup lang="ts">
import { computed } from 'vue'
import type { CandidateReviewEntry, DiscoveryMethod } from '../types/candidateReview'
import CandidateFactors from './CandidateFactors.vue'
import BridgeEvidence from './BridgeEvidence.vue'
import { Card, CardContent, CardFooter, CardHeader } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

const props = defineProps<{
  entry: CandidateReviewEntry
}>()

const provenanceLabel: Record<DiscoveryMethod, string> = {
  classic_reference_artist: 'Discovered via classic reference artists',
  current_reference_artist: 'Discovered via current reference artists',
  lastfm_similar_artist: 'Discovered via Last.fm',
  manual: 'Added manually',
}

const provenance = computed(
  () => props.entry.Ranked.Candidate.Provenance.find((p) => p.Method !== 'manual') ?? null,
)

const provenanceText = computed(() => {
  const p = provenance.value
  if (!p) return null
  let text = provenanceLabel[p.Method]
  if (p.Method === 'lastfm_similar_artist' && p.DiscoveredArtist) {
    text += ` · Related artist: ${p.DiscoveredArtist.Name}`
    if (p.LastFMMatch !== null) text += ` (match ${p.LastFMMatch.toFixed(2)})`
  } else if (p.Seed) {
    text += ` · Seed: ${p.Seed.Name}`
  }
  return text
})
</script>

<template>
  <Card>
    <CardHeader>
      <div class="flex items-start justify-between gap-4">
        <div class="min-w-0">
          <h3 class="truncate text-xl font-semibold tracking-tight">{{ entry.Ranked.Candidate.TrackTitle }}</h3>
          <p class="text-sm font-medium text-muted-foreground">{{ entry.Ranked.Candidate.TrackArtist }}</p>
          <p v-if="entry.Ranked.Candidate.Metadata" class="text-xs text-muted-foreground/70">
            {{ entry.Ranked.Candidate.Metadata.Album.Name }}
          </p>
          <div class="mt-2 flex gap-1">
            <Badge variant="outline" class="text-[0.65rem] uppercase tracking-wide">{{ entry.Ranked.Candidate.Category }}</Badge>
            <Badge variant="outline" class="text-[0.65rem] uppercase tracking-wide">{{ entry.Ranked.Candidate.Type }}</Badge>
          </div>
        </div>
        <div class="shrink-0 text-right">
          <p class="text-xs text-muted-foreground tabular-nums">#{{ entry.Ranked.Rank }}</p>
          <Tooltip v-if="entry.Ranked.Score.FinalScore !== null">
            <TooltipTrigger as-child>
              <p class="text-lg font-semibold text-primary tabular-nums">{{ entry.Ranked.Score.FinalScore.toFixed(2) }}</p>
            </TooltipTrigger>
            <TooltipContent>Internal ranking signal — not a quality rating.</TooltipContent>
          </Tooltip>
          <p v-else class="text-xs text-muted-foreground">Not yet scored</p>
        </div>
      </div>
    </CardHeader>

    <CardContent class="flex flex-col gap-3">
      <p class="text-sm leading-relaxed text-foreground/90">{{ entry.Explanation.Text }}</p>

      <template v-if="entry.Bridge?.PotentialBridge">
        <Separator />
        <BridgeEvidence
          :bridge-track="entry.BridgeTrack"
          :candidate-title="entry.Ranked.Candidate.TrackTitle"
          :bridge="entry.Bridge"
        />
      </template>

      <Separator />
      <CandidateFactors :factors="entry.Ranked.Score.Factors" />

      <p v-if="provenanceText" class="text-xs text-muted-foreground">{{ provenanceText }}</p>
    </CardContent>

    <CardFooter class="justify-end">
      <Button
        type="button"
        variant="outline"
        size="sm"
        @click="() => console.log('Open details', entry.Ranked.Candidate.ID)"
      >
        Open details
      </Button>
    </CardFooter>
  </Card>
</template>
