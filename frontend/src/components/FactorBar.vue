<script setup lang="ts">
defineProps<{
  label: string
  value: number | null
}>()
</script>

<template>
  <div v-if="value !== null" class="factor-row">
    <span class="factor-label">{{ label }}</span>
    <div class="factor-track">
      <div class="factor-fill" :style="{ width: `${value * 100}%` }"></div>
    </div>
    <span class="factor-value">{{ value.toFixed(2) }}</span>
  </div>
</template>

<style scoped>
.factor-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.85rem;
}

.factor-label {
  flex: 0 0 9rem;
  color: var(--text-h, #666);
}

.factor-track {
  flex: 1 1 auto;
  height: 0.4rem;
  background: var(--track-bg, #e5e5e5);
  border-radius: 0.2rem;
  overflow: hidden;
}

.factor-fill {
  height: 100%;
  background: var(--track-fill, #555);
}

.factor-value {
  flex: 0 0 2.5rem;
  text-align: right;
  font-variant-numeric: tabular-nums;
}
</style>
