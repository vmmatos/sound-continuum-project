<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Ref } from 'vue'
import type { CandidateCategory, CandidateReviewEntry, DiscoveryMethod } from '../types/candidateReview'
import { keepCandidate, maybeCandidate, skipCandidate, clearCandidateDecision } from '../services/candidateReview'
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

// isSelected/isMaybe/isSkipped mirror the candidate's persisted Status (the
// backend overlays StatusSelected/StatusUnderReview/StatusRejected onto a
// freshly-discovered candidate once Keep/Maybe/Skip has been called — see
// review.Service.ReviewPool), so a page refresh shows the correct state with
// no extra request. The backend's single-row-per-candidate selection table
// guarantees these three are never more than one true at once.
const isSelected = ref(props.entry.Ranked.Candidate.Status === 'selected')
const isMaybe = ref(props.entry.Ranked.Candidate.Status === 'under review')
const isSkipped = ref(props.entry.Ranked.Candidate.Status === 'rejected')
const keeping = ref(false)
const keepFailed = ref(false)
const maybeing = ref(false)
const maybeFailed = ref(false)
const skipping = ref(false)
const skipFailed = ref(false)

// Clicking an already-active action undoes it (Clear); clicking one of the
// other two actions overwrites it — the backend's upsert makes Keep/Maybe/
// Skip mutually exclusive. setDecision captures the shared shape (toggle,
// clear the other two refs on success, surface failure) once; onKeep/
// onMaybe/onSkip just supply which refs/endpoint apply to them.
async function setDecision(
  active: Ref<boolean>,
  busy: Ref<boolean>,
  failed: Ref<boolean>,
  others: Ref<boolean>[],
  apply: (id: string) => Promise<boolean>,
) {
  busy.value = true
  failed.value = false
  const ok = active.value
    ? await clearCandidateDecision(props.entry.Ranked.Candidate.ID)
    : await apply(props.entry.Ranked.Candidate.ID)
  if (ok) {
    active.value = !active.value
    if (active.value) others.forEach((other) => (other.value = false))
  } else {
    failed.value = true
  }
  busy.value = false
}

function onKeep() {
  return setDecision(isSelected, keeping, keepFailed, [isMaybe, isSkipped], keepCandidate)
}

function onMaybe() {
  return setDecision(isMaybe, maybeing, maybeFailed, [isSelected, isSkipped], maybeCandidate)
}

function onSkip() {
  return setDecision(isSkipped, skipping, skipFailed, [isSelected, isMaybe], skipCandidate)
}

// AvailableWeight is the sum of weights of scoring factors actually
// evaluated for this candidate — a low value (e.g. only Freshness, 0.10)
// means the score reflects a sliver of the full model, not a weak
// candidate. Distinguishing this from a fully-evaluated score is required
// so a curator never reads a partial score as a complete evaluation.
const isFullyScored = computed(() => props.entry.Ranked.Score.AvailableWeight >= 0.999)
const availablePercent = computed(() => Math.round(props.entry.Ranked.Score.AvailableWeight * 100))

// Soft, per-category tint (cool-toned palette, deliberately distinct from
// the warm/green decision-button colors below) so Past/Present/Emerging are
// distinguishable at a glance, not just by reading the label — layered on
// top of the Badge's existing "outline" variant via the class prop (Badge
// merges it in through cn()/twMerge, same mechanism CandidateCard already
// uses for its text-size override).
const categoryBadgeClass: Record<CandidateCategory, string> = {
  Past: 'bg-blue-500/10 text-blue-300 border-blue-500/30',
  Present: 'bg-cyan-500/10 text-cyan-300 border-cyan-500/30',
  Emerging: 'bg-violet-500/10 text-violet-300 border-violet-500/30',
}

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
            <Badge variant="outline" :class="['text-[0.65rem] uppercase tracking-wide', categoryBadgeClass[entry.Ranked.Candidate.Category]]">{{ entry.Ranked.Candidate.Category }}</Badge>
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
          variant="ghost"
          size="sm"
          :disabled="keeping"
          :class="isSelected && 'bg-emerald-500/15 text-emerald-300 border-emerald-500/40 hover:bg-emerald-500/25 hover:text-emerald-300'"
          @click="onKeep"
        >
          {{ isSelected ? 'Kept ✓' : 'Keep' }}
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          :disabled="maybeing"
          :class="isMaybe && 'bg-amber-500/15 text-amber-300 border-amber-500/40 hover:bg-amber-500/25 hover:text-amber-300'"
          @click="onMaybe"
        >
          {{ isMaybe ? 'Maybe ✓' : 'Maybe' }}
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          :disabled="skipping"
          :class="isSkipped && 'bg-rose-500/15 text-rose-300 border-rose-500/40 hover:bg-rose-500/25 hover:text-rose-300'"
          @click="onSkip"
        >
          {{ isSkipped ? 'Skipped ✓' : 'Skip' }}
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
      <p v-if="maybeFailed" class="text-xs text-destructive">Failed to update this candidate. Try again.</p>
      <p v-if="skipFailed" class="text-xs text-destructive">Failed to skip this candidate. Try again.</p>
    </CardFooter>
  </Card>
</template>
