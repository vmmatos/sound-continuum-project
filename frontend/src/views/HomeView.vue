<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { checkHealth } from '../services/health'

const backendStatus = ref<'checking' | 'ok' | 'unreachable'>('checking')

onMounted(async () => {
  const health = await checkHealth()
  backendStatus.value = health?.status === 'ok' ? 'ok' : 'unreachable'
})
</script>

<template>
  <main>
    <h1>Sound Continuum</h1>
    <p>Foundation for the weekly music curation workflow.</p>
    <p class="backend-status">Backend: {{ backendStatus }}</p>
  </main>
</template>

<style scoped>
main {
  text-align: center;
  padding: 4rem 1rem;
}

.backend-status {
  color: var(--text-h, #666);
  font-size: 0.9rem;
}
</style>
