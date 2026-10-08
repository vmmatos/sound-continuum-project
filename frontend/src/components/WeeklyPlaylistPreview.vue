<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { CandidateReviewEntry } from '../types/candidateReview'
import CandidateTrackMetadata from './CandidateTrackMetadata.vue'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Button } from '@/components/ui/button'

const props = defineProps<{
  entries: CandidateReviewEntry[]
}>()

// Kept candidate IDs (Status === 'selected') in the backend's deterministic
// Rank order (scoring.Rank — FinalScore descending, ID ascending tiebreak).
// This is only ever used as the *starting point*/sync source for manual
// order below — it is never displayed directly, since rank is a ranking
// signal, not an editorial sequence.
const keptIds = computed(() =>
  props.entries.filter((e) => e.Ranked.Candidate.Status === 'selected').map((e) => e.Ranked.Candidate.ID),
)

// The curator's manual playlist sequence — editorial/narrative order
// (transitions, bridges, journey), deliberately independent of Rank/
// FinalScore. Lives here, not in CandidateReviewView, since this component
// is never torn down/recreated during a session (CandidateReviewView's
// `status` only ever moves forward out of 'loading'), so it survives every
// Keep/Maybe/Skip click made elsewhere on the same shared `entries` array.
const order = ref<string[]>([])

// Reconciles `order` against Keep/Maybe/Skip changes without ever
// resetting a manually-set position: newly-Kept IDs are appended at the
// end; IDs that stopped being Kept are dropped, preserving the relative
// order of everyone else.
watch(
  keptIds,
  (ids) => {
    const idSet = new Set(ids)
    const next = order.value.filter((id) => idSet.has(id))
    const known = new Set(next)
    for (const id of ids) {
      if (!known.has(id)) next.push(id)
    }
    order.value = next
  },
  { immediate: true },
)

const keptEntries = computed(() => {
  const byId = new Map(props.entries.map((e) => [e.Ranked.Candidate.ID, e]))
  return order.value.map((id) => byId.get(id)).filter((e): e is CandidateReviewEntry => !!e)
})

function moveTo(from: number, to: number) {
  if (to < 0 || to >= order.value.length || from === to) return
  const next = order.value.slice()
  const [moved] = next.splice(from, 1)
  next.splice(to, 0, moved)
  order.value = next
}

// Native HTML5 Drag and Drop state — no new dependency (no sortable
// primitive exists in reka-ui, and @vueuse/core's useDraggable/useSorted
// are free-form pointer dragging / comparator sorting, not this).
// dragEnabled gates the `draggable` attribute so a drag only starts from
// the handle (mousedown), not anywhere on the row — avoids turning the
// whole row into an accidental drag surface.
const dragEnabled = ref(false)
const draggedIndex = ref<number | null>(null)
const dropTargetIndex = ref<number | null>(null)

function onDragStart(i: number, e: DragEvent) {
  draggedIndex.value = i
  e.dataTransfer?.setData('text/plain', String(i)) // required for Firefox to allow the drag
  if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move'
}
function onDragOver(i: number, e: DragEvent) {
  e.preventDefault() // required for drop to fire at all
  dropTargetIndex.value = i
}
function onDrop(i: number, e: DragEvent) {
  e.preventDefault()
  if (draggedIndex.value !== null) moveTo(draggedIndex.value, i)
  draggedIndex.value = null
  dropTargetIndex.value = null
}
function onDragEnd() {
  draggedIndex.value = null
  dropTargetIndex.value = null
  dragEnabled.value = false
}

// No weekly-track-cap constant exists anywhere in this codebase (backend or
// frontend) today — this is a display-only, non-enforcing assumption local
// to this component, per the project's target of a ~15-track weekly edition.
// Exceeding it is reported, never used to truncate or discard candidates.
const WEEKLY_TRACK_TARGET = 15

// Test-only escape hatch: tests drive reordering through the up/down
// buttons (reliable in jsdom), which already call this function directly.
defineExpose({ moveTo })
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
        <li
          v-for="(e, i) in keptEntries"
          :key="e.Ranked.Candidate.ID"
          :draggable="dragEnabled"
          class="flex items-center gap-3 rounded-md"
          :class="[
            draggedIndex === i && 'opacity-40',
            dropTargetIndex === i && draggedIndex !== null && draggedIndex !== i && 'border-t-2 border-primary',
          ]"
          @dragstart="onDragStart(i, $event)"
          @dragover="onDragOver(i, $event)"
          @drop="onDrop(i, $event)"
          @dragend="onDragEnd"
        >
          <span
            class="cursor-grab select-none text-muted-foreground active:cursor-grabbing"
            aria-hidden="true"
            title="Drag to reorder"
            @mousedown="dragEnabled = true"
            @mouseup="dragEnabled = false"
            @mouseleave="dragEnabled = false"
            >⋮⋮</span
          >
          <span class="w-5 shrink-0 text-right text-sm text-muted-foreground tabular-nums">{{ i + 1 }}.</span>
          <CandidateTrackMetadata
            class="min-w-0 flex-1"
            :title="e.Ranked.Candidate.TrackTitle"
            :artist="e.Ranked.Candidate.TrackArtist"
            :album="e.Ranked.Candidate.Metadata?.Album ?? null"
          />
          <div class="flex shrink-0 flex-col gap-0.5">
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              :disabled="i === 0"
              :aria-label="`Move ${e.Ranked.Candidate.TrackTitle} up`"
              @click="moveTo(i, i - 1)"
              >↑</Button
            >
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              :disabled="i === keptEntries.length - 1"
              :aria-label="`Move ${e.Ranked.Candidate.TrackTitle} down`"
              @click="moveTo(i, i + 1)"
              >↓</Button
            >
          </div>
        </li>
      </ol>
    </CardContent>
  </Card>
</template>
