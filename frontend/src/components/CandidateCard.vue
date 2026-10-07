<script setup lang="ts">
import { computed, ref } from 'vue'
import type { CandidateReviewEntry, DiscoveryMethod } from '../types/candidateReview'
import { keepCandidate } from '../services/candidateReview'
import CandidateFactors from './CandidateFactors.vue'
import BridgeEvidence from './BridgeEvidence.vue'
import CandidateTrackMetadata from './CandidateTrackMetadata.vue'
import { Card, CardContent, CardFooter, CardHeader } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

const props = defineProps<{
  entry: CandidateReviewEntry
}>()

// isSelected mirrors the candidate's persisted Status (the backend overlays
// StatusSelected onto a freshly-discovered candidate once Keep has been
// called — see review.Service.ReviewPool), so a page refresh after Keep
// shows the correct state with no extra request.
const isSelected = ref(props.entry.Ranked.Candidate.Status === 'selected')
const keeping = ref(false)
const keepFailed = ref(false)

async function onKeep() {
  keeping.value = true
  keepFailed.value = false
  const ok = await keepCandidate(props.entry.Ranked.Candidate.ID)
  if (ok) {
    isSelected.value = true
  } else {
    keepFailed.value = true
  }
  keeping.value = false
}

// AvailableWeight is the sum of weights of scoring factors actually
// evaluated for this candidate — a low value (e.g. only Freshness, 0.10)
// means the score reflects a sliver of the full model, not a weak
// candidate. Distinguishing this from a fully-evaluated score is required
// so a curator never reads a partial score as a complete evaluation.
const isFullyScored = computed(() => props.entry.Ranked.Score.AvailableWeight >= 0.999)
const availablePercent = computed(() => Math.round(props.entry.Ranked.Score.AvailableWeight * 100))

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
          <CandidateTrackMetadata
            :title="entry.Ranked.Candidate.TrackTitle"
            :artist="entry.Ranked.Candidate.TrackArtist"
            :album="entry.Ranked.Candidate.Metadata?.Album ?? null"
          />
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
            <TooltipContent>
              {{
                isFullyScored
                  ? 'Internal ranking signal — not a quality rating.'
                  : `Partial score — only ${availablePercent}% of the scoring signal is available.`
              }}
            </TooltipContent>
          </Tooltip>
          <p v-if="entry.Ranked.Score.FinalScore !== null && !isFullyScored" class="text-[10px] text-muted-foreground">
            Partial · {{ availablePercent }}% signal
          </p>
          <p v-else-if="entry.Ranked.Score.FinalScore === null" class="text-xs text-muted-foreground">Not yet scored</p>
        </div>
      </div>
    </CardHeader>

    <CardContent class="flex flex-col gap-3">
      <p class="text-sm leading-relaxed text-foreground/90">{{ entry.Explanation.Text }}</p>

      <BridgeEvidence
        v-if="entry.Bridge?.PotentialBridge"
        :bridge-track="entry.BridgeTrack"
        :candidate-title="entry.Ranked.Candidate.TrackTitle"
        :bridge="entry.Bridge"
      />

      <Separator />
      <CandidateFactors :factors="entry.Ranked.Score.Factors" />

      <p v-if="provenanceText" class="text-xs text-muted-foreground">{{ provenanceText }}</p>
    </CardContent>

    <CardFooter class="flex-col items-end gap-1">
      <div class="flex justify-end gap-1">
        <Button
          type="button"
          :variant="isSelected ? 'secondary' : 'ghost'"
          size="sm"
          :disabled="keeping"
          @click="onKeep"
        >
          {{ isSelected ? 'Kept ✓' : 'Keep' }}
        </Button>
        <Button
          v-if="entry.Ranked.Candidate.Metadata?.SpotifyURL"
          as="a"
          :href="entry.Ranked.Candidate.Metadata.SpotifyURL"
          target="_blank"
          rel="noopener noreferrer"
          variant="ghost"
          size="sm"
        >
          Listen on Spotify
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          @click="() => console.log('Open details', entry.Ranked.Candidate.ID)"
        >
          Open details <span aria-hidden="true">→</span>
        </Button>
      </div>
      <p v-if="keepFailed" class="text-xs text-destructive">Failed to keep this candidate. Try again.</p>
    </CardFooter>
  </Card>
</template>
