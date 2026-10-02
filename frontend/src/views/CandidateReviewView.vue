<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getCandidateReviewPool } from '../services/candidateReview'
import type { CandidateReviewEntry } from '../types/candidateReview'
import CandidateCard from '../components/CandidateCard.vue'

const status = ref<'loading' | 'ok' | 'empty' | 'error'>('loading')
const editionContext = ref('')
const entries = ref<CandidateReviewEntry[]>([])

onMounted(async () => {
  const pool = await getCandidateReviewPool()
  if (!pool) {
    status.value = 'error'
    return
  }
  editionContext.value = pool.EditionContext
  entries.value = pool.Entries
  status.value = pool.Entries.length === 0 ? 'empty' : 'ok'
})
</script>

<template>
  <section class="candidate-review">
    <h2>Candidate Review</h2>

    <p v-if="status === 'loading'">Loading candidate pool...</p>
    <p v-else-if="status === 'error'">Could not load the candidate pool.</p>
    <p v-else-if="status === 'empty'">No candidates available for review.</p>
    <template v-else>
      <p class="context">{{ editionContext }}</p>
      <p class="count">{{ entries.length }} candidates</p>
      <div class="list">
        <CandidateCard v-for="e in entries" :key="e.Ranked.Candidate.ID" :entry="e" />
      </div>
    </template>
  </section>
</template>

<style scoped>
.candidate-review {
  max-width: 960px;
  margin: 2rem auto 0;
  padding: 0 1rem;
  text-align: left;
}

.context {
  color: var(--text-h, #666);
  margin-bottom: 0.1rem;
}

.count {
  color: var(--text-h, #666);
  font-size: 0.9rem;
  margin-top: 0;
}

.list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
</style>
