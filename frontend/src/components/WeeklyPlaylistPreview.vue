<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import type { CandidateReviewEntry } from '../types/candidateReview'
import { confirmEdition } from '../services/edition'
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

// Confirm/lock step (Card #61). `confirmed` distinguishes the "editable
// playlist" (current Kept tracks + current `order`, above) from the
// "confirmed playlist" a future publishing flow would read. There is no
// second, frozen copy of `order`: nothing can change `order`'s membership
// or sequence while `confirmed` is true — the reorder controls only render
// when `!confirmed`, `moveTo` itself refuses to run while `confirmed`, and
// the watcher below drops `confirmed` back to false the instant a Keep/
// Maybe/Skip change elsewhere would change membership. Reading
// `keptEntries` while `confirmed` is true therefore always *is* the
// confirmed state. Confirming never reads or writes `CandidateTrack.Status`
// — that stays Card #56/#57/#58's field alone.
const confirmed = ref(false)

// Reconciles `order` against Keep/Maybe/Skip changes without ever
// resetting a manually-set position: newly-Kept IDs are appended at the
// end; IDs that stopped being Kept are dropped, preserving the relative
// order of everyone else.
watch(
  keptIds,
  (ids) => {
    const idSet = new Set(ids)
    const next = order.value.filter((id) => idSet.has(id))
    next.push(...ids.filter((id) => !order.value.includes(id)))
    // `keptIds` recomputes (a new array) on ANY candidate's Status write,
    // not just one that changes the Kept set — `.filter` reads every
    // entry's Status, so e.g. an unrelated Maybe→Skip flip elsewhere also
    // triggers this watcher. A real diff against the current `order` is
    // required here so a Keep/Maybe/Skip change that actually affects
    // membership or sequence invalidates a prior confirmation, while an
    // unrelated one doesn't un-confirm the playlist for no visible reason.
    const membershipChanged = next.length !== order.value.length || next.some((id, i) => id !== order.value[i])
    if (confirmed.value && membershipChanged) confirmed.value = false
    order.value = next
  },
  { immediate: true },
)

const keptEntries = computed(() => {
  const byId = new Map(props.entries.map((e) => [e.Ranked.Candidate.ID, e]))
  return order.value.map((id) => byId.get(id)).filter((e): e is CandidateReviewEntry => !!e)
})

function moveTo(from: number, to: number) {
  if (confirmed.value || to < 0 || to >= order.value.length || from === to) return
  const next = order.value.slice()
  const [moved] = next.splice(from, 1)
  next.splice(to, 0, moved)
  order.value = next
}

const confirmButtonRef = ref<{ $el: HTMLElement } | null>(null)
const editButtonRef = ref<{ $el: HTMLElement } | null>(null)

// Confirming (Card #139) is no longer a local-only flip: it must actually
// persist as the backend Edition's confirmed snapshot before the UI claims
// "confirmed" — otherwise reloading the page would silently lose a playlist
// the curator believes is locked in. `confirming` disables the button for
// the round trip; `confirmError` surfaces a failure instead of pretending
// it succeeded.
const confirming = ref(false)
const confirmError = ref<string | null>(null)

async function confirmPlaylist() {
  if (keptEntries.value.length === 0 || confirmed.value || confirming.value) return
  confirming.value = true
  confirmError.value = null
  const persisted = await confirmEdition(keptEntries.value)
  confirming.value = false
  if (!persisted) {
    confirmError.value = 'Could not save the confirmed playlist. Please try again.'
    return
  }
  confirmed.value = true
  nextTick(() => editButtonRef.value?.$el?.focus())
}

function editPlaylist() {
  if (!confirmed.value) return
  confirmed.value = false
  confirmError.value = null
  nextTick(() => confirmButtonRef.value?.$el?.focus())
}

// The one integration point a future Spotify-publishing flow would read
// (via a template ref on this component) — null whenever nothing is
// confirmed, so a confirmed *empty* playlist can never exist. Extends
// `keptEntries` rather than inventing a second order concept or a
// precomputed shape (e.g. a track-ID list) nothing calls yet — a future
// caller derives `SpotifyTrackID` from each entry when it exists.
const confirmedPlaylist = computed(() => (confirmed.value ? keptEntries.value : null))

defineExpose({ confirmedPlaylist })

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
      <div class="mb-3 flex items-center justify-between gap-4 rounded-md border border-border bg-card/60 p-3">
        <p role="status" class="text-xs" :class="confirmed ? 'text-emerald-300' : 'text-muted-foreground'">
          {{
            confirmed
              ? `Final playlist confirmed · ${keptEntries.length} ${keptEntries.length === 1 ? 'track' : 'tracks'} · Ready to publish`
              : "Lock in this week's playlist once you're happy with the Kept tracks and their order."
          }}
        </p>
        <Button
          v-if="!confirmed"
          ref="confirmButtonRef"
          type="button"
          variant="default"
          size="sm"
          :disabled="keptEntries.length === 0 || confirming"
          @click="confirmPlaylist"
        >
          Confirm final playlist
        </Button>
        <Button v-else ref="editButtonRef" type="button" variant="outline" size="sm" @click="editPlaylist">
          Edit playlist
        </Button>
      </div>

      <p v-if="confirmError" role="alert" class="mb-3 text-xs text-destructive">{{ confirmError }}</p>

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
            v-if="!confirmed"
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
          <div v-if="!confirmed" class="flex shrink-0 flex-col gap-0.5">
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
