<script setup lang="ts">
import { computed } from 'vue'
import type { CandidateReviewEntry } from '../types/candidateReview'
import FactorBar from './FactorBar.vue'

const props = defineProps<{
  entry: CandidateReviewEntry
}>()

const factorRows = computed(() => {
  const factors = props.entry.Ranked.Score.Factors
  return [
    { label: 'Fit', value: factors.Fit },
    { label: 'Playlist Fit', value: factors.PlaylistFit },
    { label: 'Discovery Bonus', value: factors.DiscoveryBonus },
    { label: 'Diversity', value: factors.Diversity },
    { label: 'Freshness', value: factors.Freshness },
    { label: 'Repetition Penalty', value: factors.RepetitionPenalty },
  ]
})

const bridgeDimensions = computed(
  () => props.entry.Bridge?.Dimensions.filter((d) => d.Evidence) ?? [],
)
const bridgeSignals = computed(
  () => props.entry.Bridge?.Signals.filter((s) => s.Present) ?? [],
)

const provenance = computed(
  () => props.entry.Ranked.Candidate.Provenance.find((p) => p.Method !== 'manual') ?? null,
)
</script>

<template>
  <article class="candidate-card">
    <header class="card-header">
      <div class="identity">
        <span class="rank">#{{ entry.Ranked.Rank }}</span>
        <div>
          <h3>{{ entry.Ranked.Candidate.TrackTitle }}</h3>
          <p class="artist">{{ entry.Ranked.Candidate.TrackArtist }}</p>
          <p v-if="entry.Ranked.Candidate.Metadata" class="album">{{ entry.Ranked.Candidate.Metadata.Album.Name }}</p>
        </div>
      </div>
      <div class="tags">
        <span class="tag">{{ entry.Ranked.Candidate.Category }}</span>
        <span class="tag">{{ entry.Ranked.Candidate.Type }}</span>
      </div>
    </header>

    <p class="score">
      <template v-if="entry.Ranked.Score.FinalScore !== null">Score: {{ entry.Ranked.Score.FinalScore.toFixed(2) }}</template>
      <template v-else>Not yet scored</template>
    </p>

    <p class="explanation">{{ entry.Explanation.Text }}</p>

    <section class="factors">
      <FactorBar v-for="row in factorRows" :key="row.label" :label="row.label" :value="row.value" />
    </section>

    <section v-if="entry.Bridge?.PotentialBridge" class="bridge">
      <h4>Potential bridge</h4>
      <p v-if="entry.BridgeTrack">{{ entry.BridgeTrack }} → {{ entry.Ranked.Candidate.TrackTitle }}</p>
      <ul>
        <li v-for="dim in bridgeDimensions" :key="dim.Dimension">{{ dim.Dimension }}: {{ dim.Relationship }}</li>
        <li v-for="sig in bridgeSignals" :key="sig.Signal">{{ sig.Signal.replace(/_/g, ' ') }}</li>
      </ul>
    </section>

    <p v-if="provenance" class="provenance">
      {{ provenance.Method.replace(/_/g, ' ') }}
      <template v-if="provenance.Seed">— seed: {{ provenance.Seed.Name }}</template>
      <template v-if="provenance.DiscoveredArtist"> → {{ provenance.DiscoveredArtist.Name }}</template>
      <template v-if="provenance.LastFMMatch !== null"> (match {{ provenance.LastFMMatch.toFixed(2) }})</template>
    </p>

    <footer>
      <button type="button" @click="() => console.log('Open details', entry.Ranked.Candidate.ID)">Open details</button>
    </footer>
  </article>
</template>

<style scoped>
.candidate-card {
  border: 1px solid var(--border, #ddd);
  border-radius: 0.5rem;
  padding: 1rem 1.25rem;
  text-align: left;
}

.card-header {
  display: flex;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.identity {
  display: flex;
  gap: 0.75rem;
}

.rank {
  font-weight: bold;
  color: var(--text-h, #666);
}

.identity h3 {
  margin: 0;
}

.artist,
.album {
  margin: 0.15rem 0 0;
  color: var(--text-h, #666);
  font-size: 0.9rem;
}

.tags {
  display: flex;
  gap: 0.4rem;
  align-items: flex-start;
}

.tag {
  font-size: 0.75rem;
  border: 1px solid var(--border, #ddd);
  border-radius: 0.3rem;
  padding: 0.1rem 0.4rem;
  color: var(--text-h, #666);
}

.score {
  font-weight: bold;
  margin: 0.75rem 0 0.25rem;
}

.explanation {
  margin: 0 0 0.75rem;
}

.factors {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  margin-bottom: 0.75rem;
}

.bridge {
  margin-bottom: 0.75rem;
  font-size: 0.9rem;
}

.bridge h4 {
  margin: 0 0 0.25rem;
}

.bridge ul {
  margin: 0.25rem 0 0;
  padding-left: 1.2rem;
}

.provenance {
  font-size: 0.85rem;
  color: var(--text-h, #666);
  margin: 0 0 0.75rem;
}

footer button {
  font-size: 0.85rem;
}
</style>
