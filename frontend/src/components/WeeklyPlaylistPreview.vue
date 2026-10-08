<script setup lang="ts">
import { computed } from 'vue'
import type { CandidateReviewEntry } from '../types/candidateReview'
import CandidateTrackMetadata from './CandidateTrackMetadata.vue'
import { Card, CardContent, CardHeader } from '@/components/ui/card'

const props = defineProps<{
  entries: CandidateReviewEntry[]
}>()

// Kept candidates only (Status === 'selected') — the only status Keep ever
// sets. No second "playlist selection" state: this is a filtered view over
// the same CandidateTrack.Status the Candidate Review cards already show
// and mutate (see CandidateCard.vue's setDecision).
const keptEntries = computed(() => props.entries.filter((e) => e.Ranked.Candidate.Status === 'selected'))

// `entries` already arrives in the backend's deterministic Rank order
// (scoring.Rank — FinalScore descending, ID ascending tiebreak); filtering
// it preserves that order, so no second ordering rule is introduced here.

// No weekly-track-cap constant exists anywhere in this codebase (backend or
// frontend) today — this is a display-only, non-enforcing assumption local
// to this component, per the project's target of a ~15-track weekly edition.
// Exceeding it is reported, never used to truncate or discard candidates.
const WEEKLY_TRACK_TARGET = 15
</script>

<template>
  <Card>
    <CardHeader>
      <div class="flex items-baseline justify-between gap-4">
        <p class="text-xs font-medium uppercase tracking-widest text-muted-foreground">Weekly Playlist Preview</p>
        <p class="text-xs text-muted-foreground">{{ keptEntries.length }} tracks</p>
      </div>
      <p v-if="keptEntries.length > WEEKLY_TRACK_TARGET" class="text-xs text-amber-300">
        {{ keptEntries.length }} tracks kept — above the {{ WEEKLY_TRACK_TARGET }}-track weekly target.
      </p>
    </CardHeader>
    <CardContent>
      <p v-if="keptEntries.length === 0" class="text-sm text-muted-foreground">
        No tracks kept yet. Keep a candidate below to add it to this week's playlist.
      </p>
      <ol v-else class="flex flex-col gap-3">
        <li v-for="(e, i) in keptEntries" :key="e.Ranked.Candidate.ID" class="flex items-center gap-3">
          <span class="w-5 shrink-0 text-right text-sm text-muted-foreground tabular-nums">{{ i + 1 }}.</span>
          <CandidateTrackMetadata
            :title="e.Ranked.Candidate.TrackTitle"
            :artist="e.Ranked.Candidate.TrackArtist"
            :album="e.Ranked.Candidate.Metadata?.Album ?? null"
          />
        </li>
      </ol>
    </CardContent>
  </Card>
</template>
