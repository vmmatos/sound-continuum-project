import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import CandidateReviewView from './CandidateReviewView.vue'
import CandidateCard from '../components/CandidateCard.vue'
import type { CandidateReviewEntry, CandidateReviewPool } from '../types/candidateReview'
import {
  getCandidateReviewPool,
  keepCandidate,
  maybeCandidate,
  skipCandidate,
  clearCandidateDecision,
} from '../services/candidateReview'
import { makeEntry } from '../test/makeEntry'
import { confirmEdition } from '../services/edition'

vi.mock('../services/candidateReview', () => ({
  getCandidateReviewPool: vi.fn(),
  keepCandidate: vi.fn(),
  maybeCandidate: vi.fn(),
  skipCandidate: vi.fn(),
  clearCandidateDecision: vi.fn(),
}))
vi.mock('../services/edition', () => ({ confirmEdition: vi.fn() }))

function pool(entries: CandidateReviewEntry[], extra: Partial<CandidateReviewPool> = {}): CandidateReviewPool {
  return { Entries: entries, ...extra } as CandidateReviewPool
}

const generateButton = (w: ReturnType<typeof mount>) =>
  w.findAll('button').find((b) => /Generat|Regenerate/.test(b.text()))!
const previewTitles = (w: ReturnType<typeof mount>) => w.findAll('li').map((li) => li.find('h3').text())

const card = (w: ReturnType<typeof mount>, title: string) =>
  w.findAllComponents(CandidateCard).find((c) => c.props('entry').Ranked.Candidate.TrackTitle === title)!
const cardButton = (w: ReturnType<typeof mount>, title: string, label: string) =>
  card(w, title).findAll('button').find((b) => b.text() === label)!

async function generate(w: ReturnType<typeof mount>) {
  await generateButton(w).trigger('click')
  await flushPromises()
}

describe('CandidateReviewView', () => {
  beforeEach(() => {
    for (const fn of [getCandidateReviewPool, keepCandidate, maybeCandidate, skipCandidate, clearCandidateDecision]) vi.mocked(fn).mockReset()
    vi.mocked(confirmEdition).mockReset().mockResolvedValue(true)
  })
  afterEach(() => vi.clearAllMocks())

  it('does not run discovery on mount and offers an explicit generate action', async () => {
    const w = mount(CandidateReviewView)
    await flushPromises()
    expect(getCandidateReviewPool).not.toHaveBeenCalled()
    expect(generateButton(w).text()).toBe('Generate candidate pool')
    expect(w.text()).toContain("Candidates aren't saved between page loads")
  })

  it('shows loading and ignores duplicate clicks while a generation is running', async () => {
    let resolve!: (p: CandidateReviewPool) => void
    vi.mocked(getCandidateReviewPool).mockReturnValue(new Promise((r) => (resolve = r)))
    const w = mount(CandidateReviewView)

    await generateButton(w).trigger('click')
    expect(generateButton(w).text()).toBe('Generating…')
    expect(generateButton(w).attributes('disabled')).toBeDefined()
    await generateButton(w).trigger('click')
    expect(getCandidateReviewPool).toHaveBeenCalledTimes(1)

    resolve(pool([makeEntry('a', 'discovered', 1)]))
    await flushPromises()
    expect(generateButton(w).text()).toBe('Regenerate pool')
  })

  it('makes generated candidates available for review', async () => {
    vi.mocked(getCandidateReviewPool).mockResolvedValue(pool([makeEntry('a', 'discovered', 1), makeEntry('b', 'selected', 2)]))
    const w = mount(CandidateReviewView)
    await generate(w)
    expect(w.text()).toContain('2 candidates generated.')
    expect(w.text()).toContain('Weekly Playlist Preview')
    expect(previewTitles(w)).toEqual(['Track b'])
  })

  it('reports partial results when some discovery sources failed', async () => {
    vi.mocked(getCandidateReviewPool).mockResolvedValue(
      pool([makeEntry('a', 'discovered', 1)], { WorkflowErrors: [{ Workflow: 'emerging', Err: 'x' }] }),
    )
    const w = mount(CandidateReviewView)
    await generate(w)
    expect(w.text()).toContain('some discovery sources failed')
  })

  it('distinguishes a clean empty result from a degraded one', async () => {
    vi.mocked(getCandidateReviewPool).mockResolvedValueOnce(pool([]))
    const w = mount(CandidateReviewView)
    await generate(w)
    expect(w.text()).toContain('found no eligible candidates')

    vi.mocked(getCandidateReviewPool).mockResolvedValueOnce(pool([], { Failures: [{} as never] }))
    await generate(w)
    expect(w.text()).toContain('discovery is degraded or temporarily failing')
  })

  it('a failed run keeps the current pool, shows an error and allows retry', async () => {
    vi.mocked(getCandidateReviewPool)
      .mockResolvedValueOnce(pool([makeEntry('a', 'selected', 1)]))
      .mockResolvedValueOnce(null)
      .mockResolvedValueOnce(pool([makeEntry('a', 'selected', 1)]))
    const w = mount(CandidateReviewView)
    await generate(w)
    await generate(w)
    expect(w.find('[role="alert"]').text()).toContain('Could not generate the candidate pool')
    expect(previewTitles(w)).toEqual(['Track a'])

    await generate(w)
    expect(w.find('[role="alert"]').exists()).toBe(false)
    expect(w.text()).toContain('1 candidate generated.')
  })

  it('regeneration keeps decided candidates, manual order and confirmation', async () => {
    vi.mocked(getCandidateReviewPool).mockResolvedValueOnce(
      pool([makeEntry('a', 'selected', 1), makeEntry('b', 'selected', 2), makeEntry('c', 'discovered', 3)]),
    )
    const w = mount(CandidateReviewView)
    await generate(w)

    await w.find('[aria-label="Move Track b up"]').trigger('click')
    expect(previewTitles(w)).toEqual(['Track b', 'Track a'])
    await w.findAll('button').find((b) => b.text() === 'Confirm final playlist')!.trigger('click')
    await flushPromises()
    expect(w.text()).toContain('Final playlist confirmed')

    // second run doesn't rediscover b (e.g. a rate-limited workflow) and returns a duplicate-free fresh a
    vi.mocked(getCandidateReviewPool).mockResolvedValueOnce(pool([makeEntry('a', 'selected', 1), makeEntry('d', 'discovered', 2)]))
    await generate(w)

    expect(previewTitles(w)).toEqual(['Track b', 'Track a'])
    expect(w.text()).toContain('Final playlist confirmed')
    expect(w.text()).toContain('3 candidates') // a, d fresh + b retained; undecided c dropped, no duplicates
    expect(confirmEdition).toHaveBeenCalledTimes(1)
  })

  it('a failed first run keeps offering Generate, not Regenerate', async () => {
    vi.mocked(getCandidateReviewPool).mockResolvedValue(null)
    const w = mount(CandidateReviewView)
    await generate(w)
    expect(generateButton(w).text()).toBe('Generate candidate pool')
  })

  it('makes candidate decisions unavailable while a regeneration is running', async () => {
    vi.mocked(getCandidateReviewPool).mockResolvedValueOnce(pool([makeEntry('a', 'discovered', 1)]))
    const w = mount(CandidateReviewView)
    await generate(w)

    let resolve!: (p: CandidateReviewPool) => void
    vi.mocked(getCandidateReviewPool).mockReturnValueOnce(new Promise((r) => (resolve = r)))
    await generateButton(w).trigger('click')
    expect(w.find('[inert]').exists()).toBe(true)

    resolve(pool([makeEntry('a', 'discovered', 1)]))
    await flushPromises()
    expect(w.find('[inert]').exists()).toBe(false)
  })

  it('carries over only Kept candidates, and cards resync to the fresh status', async () => {
    vi.mocked(getCandidateReviewPool).mockResolvedValueOnce(
      pool([makeEntry('a', 'selected', 1), makeEntry('m', 'under review', 2), makeEntry('k', 'selected', 3)]),
    )
    const w = mount(CandidateReviewView)
    await generate(w)
    expect(w.text()).toContain('Kept ✓')

    // fresh run: a's Keep was cleared elsewhere; m and k not rediscovered
    vi.mocked(getCandidateReviewPool).mockResolvedValueOnce(pool([makeEntry('a', 'discovered', 1)]))
    await generate(w)

    expect(w.text()).not.toContain('Track m')
    expect(previewTitles(w)).toEqual(['Track k'])
    const aCard = card(w, 'Track a')
    expect(aCard.text()).toContain('Keep')
    expect(aCard.text()).not.toContain('Kept ✓')
  })

  it('restores persisted Maybe/Skip on cards without adding them to the playlist', async () => {
    vi.mocked(getCandidateReviewPool).mockResolvedValue(
      pool([makeEntry('k', 'selected', 1), makeEntry('m', 'under review', 2), makeEntry('s', 'rejected', 3)]),
    )
    const w = mount(CandidateReviewView)
    await generate(w)
    expect(card(w, 'Track m').text()).toContain('Maybe ✓')
    expect(card(w, 'Track s').text()).toContain('Skipped ✓')
    expect(previewTitles(w)).toEqual(['Track k'])
  })

  it('Keep and Skip on a generated candidate update the playlist preview', async () => {
    vi.mocked(getCandidateReviewPool).mockResolvedValue(pool([makeEntry('a', 'discovered', 1)]))
    vi.mocked(keepCandidate).mockResolvedValue(true)
    vi.mocked(skipCandidate).mockResolvedValue(true)
    const w = mount(CandidateReviewView)
    await generate(w)
    expect(previewTitles(w)).toEqual([])

    await cardButton(w, 'Track a', 'Keep').trigger('click')
    await flushPromises()
    expect(keepCandidate).toHaveBeenCalledWith('a')
    expect(previewTitles(w)).toEqual(['Track a'])

    await cardButton(w, 'Track a', 'Skip').trigger('click')
    await flushPromises()
    expect(previewTitles(w)).toEqual([])
  })

  it('a failed decision save is reported and leaves the playlist unchanged', async () => {
    vi.mocked(getCandidateReviewPool).mockResolvedValue(pool([makeEntry('a', 'discovered', 1)]))
    vi.mocked(keepCandidate).mockResolvedValue(false)
    const w = mount(CandidateReviewView)
    await generate(w)

    await cardButton(w, 'Track a', 'Keep').trigger('click')
    await flushPromises()
    expect(w.text()).toContain('Failed to keep this candidate')
    expect(previewTitles(w)).toEqual([])
  })
})
