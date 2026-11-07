import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { appendToNote, cancelAppend, getStagedCapture, loadNotes } from '../src/append.js'
import { stageCapture } from '../src/staged.js'
import { installChromeMock } from './webext-mock.js'

function jsonResponse(body, status = 200) {
  return new Response(JSON.stringify(body), { status })
}

function noteItem(overrides = {}) {
  return {
    id: 'n1',
    type: 'note',
    body: 'original text',
    version: 1,
    state: 'inbox',
    capturedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

describe('append', () => {
  let chrome, session

  beforeEach(async () => {
    ({ chrome, session } = installChromeMock())
    await chrome.storage.sync.set({ apiBaseUrl: 'http://test' })
    await chrome.storage.local.set({ sessionToken: 'session-token' })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  describe('loadNotes', () => {
    it('returns only notes from inbox and kept, sorted newest first', async () => {
      const fetchMock = vi.fn().mockResolvedValue(jsonResponse({
        successful: true,
        messages: [],
        data: [
          noteItem({ id: 'n1', type: 'note', body: 'first', capturedAt: '2026-01-01T00:00:00Z' }),
          noteItem({ id: 'n2', type: 'note', body: 'second', capturedAt: '2026-06-01T00:00:00Z' }),
          { id: 'l1', type: 'link', body: 'https://example.com', version: 1, state: 'inbox', capturedAt: '2026-03-01T00:00:00Z' },
        ],
      }))
      vi.stubGlobal('fetch', fetchMock)

      const notes = await loadNotes()

      expect(notes).toHaveLength(2)
      expect(notes[0].id).toBe('n2') // newest first
      expect(notes[1].id).toBe('n1')
    })
  })

  describe('getStagedCapture', () => {
    it('returns the staged capture', async () => {
      await stageCapture({ markdown: 'selected text', images: [] })
      const staged = await getStagedCapture()
      expect(staged).toBeTruthy()
      expect(staged.markdown).toBe('selected text')
    })

    it('returns null when nothing is staged', async () => {
      const staged = await getStagedCapture()
      expect(staged).toBeNull()
    })
  })

  describe('appendToNote', () => {
    it('appends after existing text with a blank line', async () => {
      await stageCapture({ markdown: 'added content', images: [] })
      const fetchMock = vi
        .fn()
        // getItem
        .mockResolvedValueOnce(jsonResponse({ successful: true, messages: [], data: noteItem({ body: 'original text', version: 1 }) }))
        // editNote (PATCH)
        .mockResolvedValueOnce(jsonResponse({ successful: true, messages: [] }))
      vi.stubGlobal('fetch', fetchMock)

      await appendToNote('n1', await getStagedCapture())

      const [getUrl] = fetchMock.mock.calls[0]
      expect(getUrl).toBe('http://test/api/items/n1')

      const [patchUrl, patchInit] = fetchMock.mock.calls[1]
      expect(patchUrl).toBe('http://test/api/items/n1')
      expect(JSON.parse(patchInit.body)).toEqual({
        body: 'original text\n\nadded content',
        version: 1,
      })
    })

    it('clears staged capture on success', async () => {
      await stageCapture({ markdown: 'x', images: [] })
      const fetchMock = vi
        .fn()
        .mockResolvedValueOnce(jsonResponse({ successful: true, messages: [], data: noteItem() }))
        .mockResolvedValueOnce(jsonResponse({ successful: true, messages: [] }))
      vi.stubGlobal('fetch', fetchMock)

      await appendToNote('n1', await getStagedCapture())

      const staged = await getStagedCapture()
      expect(staged).toBeNull()
    })

    it('keeps the staged capture on 409', async () => {
      await stageCapture({ markdown: 'added', images: [] })
      const fetchMock = vi
        .fn()
        .mockResolvedValueOnce(jsonResponse({ successful: true, messages: [], data: noteItem() }))
        .mockResolvedValueOnce(jsonResponse({ successful: false, messages: ['conflict'] }, 409))
      vi.stubGlobal('fetch', fetchMock)

      try {
        await appendToNote('n1', await getStagedCapture())
      } catch (e) {
        expect(e.name).toBe('ConflictError')
      }

      const staged = await getStagedCapture()
      expect(staged).toBeTruthy()
      expect(staged.markdown).toBe('added')
    })

    it('appends selection with images', async () => {
      await stageCapture({
        markdown: '![](@stash-image-0)',
        images: [{ src: 'https://example.com/img.png', alt: '' }],
      })
      const fetchMock = vi
        .fn()
        .mockResolvedValueOnce(jsonResponse({ successful: true, messages: [], data: noteItem({ body: 'existing', version: 1 }) }))
        // Download image
        .mockResolvedValueOnce(new Response(new Blob(['bytes'], { type: 'image/png' }), { status: 200 }))
        // PATCH multipart
        .mockResolvedValueOnce(jsonResponse({ successful: true, messages: [] }))
      vi.stubGlobal('fetch', fetchMock)

      const staged = await getStagedCapture()
      try {
        await appendToNote('n1', staged)
      } catch (e) {
        console.error('append error:', e.message)
        throw e
      }

      expect(fetchMock).toHaveBeenCalledTimes(3)
      const form = fetchMock.mock.calls[2][1].body
      expect(form.get('version')).toBe('1')
    })
  })

  describe('cancelAppend', () => {
    it('clears the staged capture', async () => {
      await stageCapture({ markdown: 'x', images: [] })
      await cancelAppend()
      const staged = await getStagedCapture()
      expect(staged).toBeNull()
    })
  })
})