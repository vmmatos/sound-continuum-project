<script setup lang="ts">
import { computed } from 'vue'
import type { Factors } from '../types/candidateReview'
import { Progress } from '@/components/ui/progress'

const props = defineProps<{
  factors: Factors
}>()

const rows = computed(() =>
  (
    [
      { label: 'Fit', value: props.factors.Fit },
      { label: 'Playlist Fit', value: props.factors.PlaylistFit },
      { label: 'Discovery Bonus', value: props.factors.DiscoveryBonus },
      { label: 'Diversity', value: props.factors.Diversity },
      { label: 'Freshness', value: props.factors.Freshness },
      { label: 'Repetition Penalty', value: props.factors.RepetitionPenalty },
    ] satisfies { label: string; value: number | null }[]
  ).filter((row): row is { label: string; value: number } => row.value !== null),
)
</script>

<template>
  <div v-if="rows.length" class="flex flex-col gap-1.5">
    <div v-for="row in rows" :key="row.label" class="flex items-center gap-2 text-xs">
      <span class="w-28 shrink-0 text-muted-foreground">{{ row.label }}</span>
      <Progress :model-value="row.value * 100" class="flex-1" />
      <span class="w-10 shrink-0 text-right tabular-nums">{{ row.value.toFixed(2) }}</span>
    </div>
  </div>
</template>
