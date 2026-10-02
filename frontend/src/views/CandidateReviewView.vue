<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getCandidateReviewPool } from '../services/candidateReview'
import type { CandidateReviewEntry } from '../types/candidateReview'
import CandidateCard from '../components/CandidateCard.vue'
import { Skeleton } from '@/components/ui/skeleton'
import { TooltipProvider } from '@/components/ui/tooltip'

const status = ref<'loading' | 'ok' | 'empty' | 'error'>('loading')
const entries = ref<CandidateReviewEntry[]>([])

onMounted(async () => {
  const pool = await getCandidateReviewPool()
  if (!pool) {
    status.value = 'error'
    return
  }
  entries.value = pool.Entries
  status.value = pool.Entries.length === 0 ? 'empty' : 'ok'
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
    <template v-else>
      <p class="mt-1 text-xs text-muted-foreground">{{ entries.length }} candidates</p>
      <TooltipProvider>
        <div class="mt-4 flex flex-col gap-4">
          <CandidateCard v-for="e in entries" :key="e.Ranked.Candidate.ID" :entry="e" />
        </div>
      </TooltipProvider>
    </template>
  </section>
</template>
