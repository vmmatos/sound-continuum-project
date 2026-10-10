import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import WeeklyPlaylistPreview from './WeeklyPlaylistPreview.vue'
import type { CandidateReviewEntry } from '../types/candidateReview'
import { makeEntry } from '../test/makeEntry'
import { confirmEdition } from '../services/edition'

vi.mock('../services/edition', () => ({ confirmEdition: vi.fn() }))

function titles(wrapper: ReturnType<typeof mount>) {
  return wrapper.findAll('li').map((li) => li.find('h3').text())
}

async function clickUp(wrapper: ReturnType<typeof mount>, id: string) {
  await wrapper.find(`[aria-label="Move Track ${id} up"]`).trigger('click')
}

function findButton(wrapper: ReturnType<typeof mount>, text: string) {
  return wrapper.findAll('button').find((b) => b.text() === text)
}
async function confirm(wrapper: ReturnType<typeof mount>) {
  await findButton(wrapper, 'Confirm final playlist')!.trigger('click')
  await flushPromises()
}
async function edit(wrapper: ReturnType<typeof mount>) {
  await findButton(wrapper, 'Edit playlist')!.trigger('click')
}

function confirmedIds(wrapper: ReturnType<typeof mount>) {
  const entries = (wrapper.vm as any).confirmedPlaylist as CandidateReviewEntry[] | null
  return entries?.map((e) => e.Ranked.Candidate.ID) ?? null
}

describe('WeeklyPlaylistPreview', () => {
  beforeEach(() => {
    vi.mocked(confirmEdition).mockReset().mockResolvedValue(true)
  })
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('shows only Kept tracks, in rank order', () => {
    const entries = [makeEntry('a', 'selected', 1), makeEntry('b', 'rejected', 2), makeEntry('c', 'selected', 3)]
    const wrapper = mount(WeeklyPlaylistPreview, { props: { entries } })
    expect(titles(wrapper)).toEqual(['Track a', 'Track c'])
  })

  it('reorders tracks when the up button is clicked, and numbering updates', async () => {
    const entries = [makeEntry('a', 'selected', 1), makeEntry('b', 'selected', 2), makeEntry('c', 'selected', 3)]
    const wrapper = mount(WeeklyPlaylistPreview, { props: { entries } })

    // move track c from position #3 to position #1
    await clickUp(wrapper, 'c')
    await clickUp(wrapper, 'c')

    expect(titles(wrapper)).toEqual(['Track c', 'Track a', 'Track b'])
    const numbers = wrapper.findAll('li').map((li) => li.findAll('span')[1].text())
    expect(numbers).toEqual(['1.', '2.', '3.'])
  })

  it('appends a newly Kept track at the end without disturbing manual order', async () => {
    const entries = [makeEntry('a', 'selected', 1), makeEntry('b', 'selected', 2), makeEntry('c', 'discovered', 3)]
    const wrapper = mount(WeeklyPlaylistPreview, { props: { entries } })

    await clickUp(wrapper, 'b') // manual order becomes [b, a]
    expect(titles(wrapper)).toEqual(['Track b', 'Track a'])

    entries[2].Ranked.Candidate.Status = 'selected' // c becomes Kept
    await wrapper.setProps({ entries: [...entries] })

    expect(titles(wrapper)).toEqual(['Track b', 'Track a', 'Track c'])
  })

  it('removes a track that is no longer Kept without corrupting the remaining order', async () => {
    const entries = [makeEntry('a', 'selected', 1), makeEntry('b', 'selected', 2), makeEntry('c', 'selected', 3)]
    const wrapper = mount(WeeklyPlaylistPreview, { props: { entries } })

    await clickUp(wrapper, 'c')
    await clickUp(wrapper, 'c') // manual order becomes [c, a, b]
    expect(titles(wrapper)).toEqual(['Track c', 'Track a', 'Track b'])

    entries[1].Ranked.Candidate.Status = 'rejected' // b is skipped
    await wrapper.setProps({ entries: [...entries] })

    expect(titles(wrapper)).toEqual(['Track c', 'Track a'])
  })

  it('never mutates candidate Status while reordering', async () => {
    const entries = [makeEntry('a', 'selected', 1), makeEntry('b', 'selected', 2)]
    const wrapper = mount(WeeklyPlaylistPreview, { props: { entries } })

    await clickUp(wrapper, 'b')

    expect(entries[0].Ranked.Candidate.Status).toBe('selected')
    expect(entries[1].Ranked.Candidate.Status).toBe('selected')
  })

  it('shows the empty state when nothing is Kept', () => {
    const entries = [makeEntry('a', 'rejected', 1)]
    const wrapper = mount(WeeklyPlaylistPreview, { props: { entries } })
    expect(wrapper.text()).toContain('No tracks kept yet')
  })

  it('confirming captures the current manual order as the confirmed playlist', async () => {
    const entries = [makeEntry('a', 'selected', 1), makeEntry('b', 'selected', 2), makeEntry('c', 'selected', 3)]
    const wrapper = mount(WeeklyPlaylistPreview, { props: { entries } })

    await confirm(wrapper)

    expect(wrapper.text()).toContain('Final playlist confirmed')
    expect(wrapper.text()).toContain('3 tracks')
    expect(confirmedIds(wrapper)).toEqual(['a', 'b', 'c'])
  })

  it('a reorder made before confirming is reflected in the confirmed order, not rank order', async () => {
    const entries = [makeEntry('a', 'selected', 1), makeEntry('b', 'selected', 2), makeEntry('c', 'selected', 3)]
    const wrapper = mount(WeeklyPlaylistPreview, { props: { entries } })

    await clickUp(wrapper, 'c')
    await clickUp(wrapper, 'c') // manual order becomes [c, a, b]
    await confirm(wrapper)

    expect(confirmedIds(wrapper)).toEqual(['c', 'a', 'b'])
  })

  it('confirmed order is stable across unrelated re-renders', async () => {
    const entries = [makeEntry('a', 'selected', 1), makeEntry('b', 'selected', 2)]
    const wrapper = mount(WeeklyPlaylistPreview, { props: { entries } })

    await confirm(wrapper)
    await wrapper.setProps({ entries: [...entries] }) // same statuses, new array reference

    expect(wrapper.text()).toContain('Final playlist confirmed')
    expect(confirmedIds(wrapper)).toEqual(['a', 'b'])
  })

  it('reordering is unavailable once the playlist is locked', async () => {
    const entries = [makeEntry('a', 'selected', 1), makeEntry('b', 'selected', 2)]
    const wrapper = mount(WeeklyPlaylistPreview, { props: { entries } })

    await confirm(wrapper)

    expect(wrapper.find('[aria-label="Move Track a up"]').exists()).toBe(false)
    expect(wrapper.find('[aria-label="Move Track b down"]').exists()).toBe(false)
    expect(confirmedIds(wrapper)).toEqual(['a', 'b'])
  })

  it('a Keep/Maybe/Skip change elsewhere invalidates the confirmation instead of silently diverging', async () => {
    const entries = [makeEntry('a', 'selected', 1), makeEntry('b', 'selected', 2), makeEntry('c', 'discovered', 3)]
    const wrapper = mount(WeeklyPlaylistPreview, { props: { entries } })

    await confirm(wrapper)
    expect(wrapper.text()).toContain('Final playlist confirmed')

    entries[2].Ranked.Candidate.Status = 'selected' // Keep clicked on CandidateCard elsewhere
    await wrapper.setProps({ entries: [...entries] })

    expect(wrapper.text()).not.toContain('Final playlist confirmed')
    expect((wrapper.vm as any).confirmedPlaylist).toBeNull()
    expect(titles(wrapper)).toEqual(['Track a', 'Track b', 'Track c'])
  })

  it('Edit playlist returns to editable state and invalidates the confirmation', async () => {
    const entries = [makeEntry('a', 'selected', 1), makeEntry('b', 'selected', 2)]
    const wrapper = mount(WeeklyPlaylistPreview, { props: { entries } })

    await confirm(wrapper)
    await edit(wrapper)

    expect(wrapper.text()).not.toContain('Final playlist confirmed')
    expect((wrapper.vm as any).confirmedPlaylist).toBeNull()
    expect(wrapper.find('[aria-label="Move Track a up"]').exists()).toBe(true)
  })

  it('reordering after Edit playlist requires reconfirmation', async () => {
    const entries = [makeEntry('a', 'selected', 1), makeEntry('b', 'selected', 2)]
    const wrapper = mount(WeeklyPlaylistPreview, { props: { entries } })

    await confirm(wrapper)
    await edit(wrapper)
    await clickUp(wrapper, 'b')

    expect((wrapper.vm as any).confirmedPlaylist).toBeNull()
    expect(wrapper.text()).not.toContain('Final playlist confirmed')

    await confirm(wrapper)
    expect(confirmedIds(wrapper)).toEqual(['b', 'a'])
  })

  it('a membership change after Edit playlist requires reconfirmation', async () => {
    const entries = [makeEntry('a', 'selected', 1), makeEntry('b', 'selected', 2)]
    const wrapper = mount(WeeklyPlaylistPreview, { props: { entries } })

    await confirm(wrapper)
    await edit(wrapper)

    entries[1].Ranked.Candidate.Status = 'rejected'
    await wrapper.setProps({ entries: [...entries] })

    expect((wrapper.vm as any).confirmedPlaylist).toBeNull()
    expect(titles(wrapper)).toEqual(['Track a'])

    await confirm(wrapper)
    expect(confirmedIds(wrapper)).toEqual(['a'])
  })

  it('confirm is unavailable with nothing Kept, and never produces a confirmed empty playlist', async () => {
    const entries = [makeEntry('a', 'rejected', 1)]
    const wrapper = mount(WeeklyPlaylistPreview, { props: { entries } })

    const button = findButton(wrapper, 'Confirm final playlist')!
    expect(button.attributes('disabled')).toBeDefined()

    await button.trigger('click')

    expect((wrapper.vm as any).confirmedPlaylist).toBeNull()
    expect(wrapper.text()).toContain('No tracks kept yet')
  })

  it('exposes the confirmed ordered tracks for an external consumer', async () => {
    const entries = [makeEntry('a', 'selected', 1), makeEntry('b', 'selected', 2)]
    const wrapper = mount(WeeklyPlaylistPreview, { props: { entries } })

    expect((wrapper.vm as any).confirmedPlaylist).toBeNull()

    await confirm(wrapper)

    expect(confirmedIds(wrapper)).toEqual(['a', 'b'])
  })

  it('confirming persists the edition via the backend before locking', async () => {
    const entries = [makeEntry('a', 'selected', 1), makeEntry('b', 'selected', 2)]
    const wrapper = mount(WeeklyPlaylistPreview, { props: { entries } })

    await confirm(wrapper)

    expect(confirmEdition).toHaveBeenCalledTimes(1)
    const sentEntries = vi.mocked(confirmEdition).mock.calls[0][0]
    expect(sentEntries.map((e) => e.Ranked.Candidate.ID)).toEqual(['a', 'b'])
    expect(wrapper.text()).toContain('Final playlist confirmed')
  })

  it('a failed confirm keeps the playlist editable and shows an error', async () => {
    vi.mocked(confirmEdition).mockResolvedValue(false)
    const entries = [makeEntry('a', 'selected', 1), makeEntry('b', 'selected', 2)]
    const wrapper = mount(WeeklyPlaylistPreview, { props: { entries } })

    await confirm(wrapper)

    expect(wrapper.text()).not.toContain('Final playlist confirmed')
    expect((wrapper.vm as any).confirmedPlaylist).toBeNull()
    expect(wrapper.text()).toContain('Could not save the confirmed playlist')
    // reorder controls must still be available — the playlist never locked
    expect(wrapper.find('[aria-label="Move Track a down"]').exists()).toBe(true)
  })

  it('retrying confirm after a failure succeeds once the backend call succeeds', async () => {
    vi.mocked(confirmEdition).mockResolvedValueOnce(false).mockResolvedValueOnce(true)
    const entries = [makeEntry('a', 'selected', 1), makeEntry('b', 'selected', 2)]
    const wrapper = mount(WeeklyPlaylistPreview, { props: { entries } })

    await confirm(wrapper)
    expect(wrapper.text()).toContain('Could not save the confirmed playlist')

    await confirm(wrapper)
    expect(wrapper.text()).toContain('Final playlist confirmed')
    expect(confirmEdition).toHaveBeenCalledTimes(2)
  })
})
