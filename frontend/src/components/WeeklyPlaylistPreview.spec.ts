import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import WeeklyPlaylistPreview from './WeeklyPlaylistPreview.vue'
import type { CandidateReviewEntry } from '../types/candidateReview'

function makeEntry(id: string, status: CandidateReviewEntry['Ranked']['Candidate']['Status'], rank: number): CandidateReviewEntry {
  return {
    Ranked: {
      Rank: rank,
      Candidate: {
        ID: id,
        SpotifyTrackID: id,
        Source: 'Spotify',
        Category: 'Past',
        Type: 'Classic',
        Status: status,
        TrackTitle: `Track ${id}`,
        TrackArtist: `Artist ${id}`,
        CreatedAt: '',
        UpdatedAt: '',
        Metadata: null,
        Provenance: [],
      },
      Score: {
        CandidateID: id,
        ModelVersion: 'v1',
        Weights: { Fit: 0, Freshness: 0, DiscoveryBonus: 0, Diversity: 0, PlaylistFit: 0, RepetitionWeight: 0 },
        Factors: { Fit: null, Freshness: null, DiscoveryBonus: null, Diversity: null, PlaylistFit: null, RepetitionPenalty: null },
        FinalScore: null,
        AvailableWeight: 0,
      },
    },
    Explanation: { Text: '', Reasons: [] },
    Bridge: null,
    BridgeTrack: null,
  }
}

function titles(wrapper: ReturnType<typeof mount>) {
  return wrapper.findAll('li').map((li) => li.find('h3').text())
}

async function clickUp(wrapper: ReturnType<typeof mount>, id: string) {
  await wrapper.find(`[aria-label="Move Track ${id} up"]`).trigger('click')
}

describe('WeeklyPlaylistPreview', () => {
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
})
