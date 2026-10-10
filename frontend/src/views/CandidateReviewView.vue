<script setup lang="ts">
import { ref } from 'vue'
import { getCandidateReviewPool } from '../services/candidateReview'
import type { CandidateReviewEntry } from '../types/candidateReview'
import CandidateCard from '../components/CandidateCard.vue'
import WeeklyPlaylistPreview from '../components/WeeklyPlaylistPreview.vue'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { TooltipProvider } from '@/components/ui/tooltip'

// Generation is an explicit curator action (Card #63): nothing here runs
// discovery on mount, so loading or reloading the page never calls Spotify.
// The pool itself is not persisted — only Keep/Maybe/Skip decisions are —
// so after a reload the curator generates again and the decisions re-attach
// by candidate ID (= Spotify track ID).
const outcome = ref<'idle' | 'done' | 'error'>('idle')
const message = ref(
  "Candidates aren't saved between page loads — generate the pool to start reviewing. Your Keep, Maybe and Skip decisions are kept.",
)
const generating = ref(false)
const entries = ref<CandidateReviewEntry[]>([])

async function generate() {
  if (generating.value) return
  generating.value = true
  const pool = await getCandidateReviewPool()
  generating.value = false
  if (!pool) {
    // Keep whatever is already on screen — a failed refresh never discards
    // the current pool or the curator's work on it.
    outcome.value = 'error'
    message.value = 'Could not generate the candidate pool. Check the Spotify connection and try again.'
    return
  }

  // A candidate the curator already decided on stays on screen even if this
  // run didn't rediscover it (e.g. a rate-limited workflow), so Kept tracks,
  // their manual order and a local confirmation are never silently dropped.
  // Fresh data wins whenever the same candidate is present in both.
  const freshIds = new Set(pool.Entries.map((e) => e.Ranked.Candidate.ID))
  const retained = entries.value.filter(
    (e) => e.Ranked.Candidate.Status !== 'discovered' && !freshIds.has(e.Ranked.Candidate.ID),
  )
  entries.value = [...pool.Entries, ...retained]
  outcome.value = 'done'

  const n = pool.Entries.length
  const generated = `${n} candidate${n === 1 ? '' : 's'} generated`
  const hasFailures = (pool.WorkflowErrors?.length ?? 0) > 0 || (pool.Failures?.length ?? 0) > 0
  if (n > 0) {
    message.value = hasFailures ? `${generated}, but some discovery sources failed — results may be incomplete.` : `${generated}.`
  } else {
    message.value = hasFailures
      ? 'Candidates could not be generated right now — discovery is degraded or temporarily failing. Try again later.'
      : 'Discovery ran successfully but found no eligible candidates.'
  }
}
</script>

<template>
  <section class="mx-auto mt-8 max-w-3xl px-4 text-left">
    <div class="flex items-center justify-between gap-4">
      <p class="text-xs font-medium uppercase tracking-widest text-muted-foreground">Candidate Review</p>
      <Button type="button" size="sm" :disabled="generating" :aria-busy="generating" @click="generate">
        {{ generating ? 'Generating…' : outcome === 'idle' ? 'Generate candidate pool' : 'Regenerate pool' }}
      </Button>
    </div>

    <p
      :role="outcome === 'error' ? 'alert' : 'status'"
      class="mt-2 text-sm"
      :class="outcome === 'error' ? 'text-destructive' : 'text-muted-foreground'"
    >
      {{ generating ? 'Generating candidate pool…' : message }}
    </p>

    <div v-if="generating && entries.length === 0" class="mt-6 flex flex-col gap-4">
      <div v-for="i in 3" :key="i" class="flex flex-col gap-3 rounded-lg border border-border p-4">
        <Skeleton class="h-5 w-48" />
        <Skeleton class="h-3 w-32" />
        <Skeleton class="mt-2 h-2 w-full" />
        <Skeleton class="h-2 w-full" />
      </div>
    </div>
    <template v-else-if="entries.length > 0">
      <p class="mt-1 text-xs text-muted-foreground">{{ entries.length }} candidates</p>
      <WeeklyPlaylistPreview class="mt-4" :entries="entries" />
      <TooltipProvider>
        <div class="mt-4 flex flex-col gap-4">
          <CandidateCard v-for="e in entries" :key="e.Ranked.Candidate.ID" :entry="e" />
        </div>
      </TooltipProvider>
    </template>
  </section>
</template>
