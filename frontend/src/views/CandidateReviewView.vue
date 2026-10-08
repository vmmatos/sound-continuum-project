<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getCandidateReviewPool } from '../services/candidateReview'
import type { CandidateReviewEntry } from '../types/candidateReview'
import CandidateCard from '../components/CandidateCard.vue'
import WeeklyPlaylistPreview from '../components/WeeklyPlaylistPreview.vue'
import { Skeleton } from '@/components/ui/skeleton'
import { TooltipProvider } from '@/components/ui/tooltip'

const status = ref<'loading' | 'ok' | 'empty' | 'degraded' | 'error'>('loading')
const entries = ref<CandidateReviewEntry[]>([])

onMounted(async () => {
  const pool = await getCandidateReviewPool()
  if (!pool) {
    status.value = 'error'
    return
  }
  entries.value = pool.Entries
  if (pool.Entries.length > 0) {
    status.value = 'ok'
    return
  }
  const hasFailures = (pool.WorkflowErrors?.length ?? 0) > 0 || (pool.Failures?.length ?? 0) > 0
  status.value = hasFailures ? 'degraded' : 'empty'
})
</script>

<template>
  <section class="mx-auto mt-8 max-w-3xl px-4 text-left">
    <p class="text-xs font-medium uppercase tracking-widest text-muted-foreground">Candidate Review</p>

    <div v-if="status === 'loading'" class="mt-6 flex flex-col gap-4">
      <div v-for="i in 3" :key="i" class="flex flex-col gap-3 rounded-lg border border-border p-4">
        <Skeleton class="h-5 w-48" />
        <Skeleton class="h-3 w-32" />
        <Skeleton class="mt-2 h-2 w-full" />
        <Skeleton class="h-2 w-full" />
      </div>
    </div>
    <p v-else-if="status === 'error'" class="mt-6 text-sm text-muted-foreground">Could not load the candidate pool.</p>
    <p v-else-if="status === 'empty'" class="mt-6 text-sm text-muted-foreground">No candidates available for review.</p>
    <p v-else-if="status === 'degraded'" class="mt-6 text-sm text-muted-foreground">
      Candidates could not be generated right now — discovery is degraded or temporarily failing. Try again later.
    </p>
    <template v-else>
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
