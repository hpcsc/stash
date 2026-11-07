import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { handleMenuClick } from '../src/dispatch.js'
import { installBrowserMock, installChromeMock } from './webext-mock.js'

function jsonResponse(body, status = 200) {
  return new Response(JSON.stringify(body), { status })
}

describe('handleMenuClick', () => {
  let chrome
  beforeEach(async () => {
    ({ chrome } = installChromeMock())
    await chrome.storage.sync.set({ apiBaseUrl: 'http://test' })
  })
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  async function signIn() {
    await chrome.storage.local.set({ sessionToken: 'session-token' })
  }

  it('saves the current page as a link', async () => {
    await signIn()
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ successful: true, messages: [] }))
    vi.stubGlobal('fetch', fetchMock)

    await handleMenuClick({ menuItemId: 'save-page', pageUrl: 'https://example.com' }, {})

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://test/api/items')
    expect(JSON.parse(init.body)).toEqual({ body: 'https://example.com', type: 'link' })
    expect(chrome.action.setBadgeText).toHaveBeenCalledWith({ text: '✓' })
  })

  it('saves the selection as text when injection returns nothing', async () => {
    await signIn()
    chrome.scripting.executeScript.mockResolvedValue([{ result: null }])
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ successful: true, messages: [] }))
    vi.stubGlobal('fetch', fetchMock)

    await handleMenuClick({ menuItemId: 'save-selection', selectionText: 'hello' }, { id: 1 })

    const [, init] = fetchMock.mock.calls[0]
    expect(JSON.parse(init.body)).toEqual({ body: 'hello' })
  })

  it('stages the selection and opens the popup for append', async () => {
    await signIn()
    chrome.scripting.executeScript.mockResolvedValue([{
      result: { markdown: 'selected text', images: [] },
    }])
    vi.stubGlobal('fetch', vi.fn())

    await handleMenuClick({ menuItemId: 'append-selection', selectionText: 'selected text' }, { id: 1 })

    // Should open the popup.
    expect(chrome.action.openPopup).toHaveBeenCalled()
    // Should NOT mark as saved — the picker does that.
    expect(chrome.action.setBadgeText).not.toHaveBeenCalledWith({ text: '✓' })
    // Should NOT send a request.
    const fetchMock = globalThis.fetch
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('falls back to selectionText for append when injection returns nothing', async () => {
    await signIn()
    chrome.scripting.executeScript.mockResolvedValue([{ result: null }])
    vi.stubGlobal('fetch', vi.fn())

    await handleMenuClick({ menuItemId: 'append-selection', selectionText: 'fallback' }, { id: 1 })

    expect(chrome.action.openPopup).toHaveBeenCalled()
    const staged = await chrome.storage.session.get('pending-selection')
    expect(staged['pending-selection'].markdown).toBe('fallback')
  })

  it('saves a selection with no images as JSON', async () => {
    await signIn()
    chrome.scripting.executeScript.mockResolvedValue([{
      result: { markdown: 'just text', images: [] },
    }])
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ successful: true, messages: [] }))
    vi.stubGlobal('fetch', fetchMock)

    await handleMenuClick({ menuItemId: 'save-selection' }, { id: 1 })

    const [, init] = fetchMock.mock.calls[0]
    expect(JSON.parse(init.body)).toEqual({ body: 'just text' })
  })

  it('saves a selection with images as multipart', async () => {
    await signIn()
    chrome.scripting.executeScript.mockResolvedValue([{
      result: {
        markdown: 'before\n\n![](@stash-image-0)\n\nafter',
        images: [{ src: 'https://example.com/photo.png', alt: '' }],
      },
    }])
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(new Response(new Blob(['image-data'], { type: 'image/png' }), { status: 200 }))
      .mockResolvedValueOnce(jsonResponse({ successful: true, messages: [] }))
    vi.stubGlobal('fetch', fetchMock)

    await handleMenuClick({ menuItemId: 'save-selection' }, { id: 1 })

    // First call: downloads the image.
    expect(fetchMock).toHaveBeenCalledWith('https://example.com/photo.png')
    // Second call: multipart upload.
    const [, init] = fetchMock.mock.calls[1]
    expect(init.body).toBeInstanceOf(FormData)
    const form = init.body
    expect(form.get('markdown')).toBe('before\n\n![](@stash-image-0)\n\nafter')
    expect(form.get('images').name).toBe('photo.png')
  })

  it('falls back to the image URL when download fails', async () => {
    await signIn()
    chrome.scripting.executeScript.mockResolvedValue([{
      result: {
        markdown: 'text\n\n![](@stash-image-0)',
        images: [{ src: 'https://example.com/bad.png', alt: '' }],
      },
    }])
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(new Response('not found', { status: 404 }))
      .mockResolvedValueOnce(jsonResponse({ successful: true, messages: [] }))
    vi.stubGlobal('fetch', fetchMock)

    await handleMenuClick({ menuItemId: 'save-selection' }, { id: 1 })

    const [, init] = fetchMock.mock.calls[1]
    expect(JSON.parse(init.body)).toEqual({ body: 'text\n\n![](https://example.com/bad.png)' })
  })

  it('falls back to selectionText when injection fails', async () => {
    await signIn()
    chrome.scripting.executeScript.mockRejectedValue(new Error('Cannot access page'))
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ successful: true, messages: [] }))
    vi.stubGlobal('fetch', fetchMock)

    await handleMenuClick({ menuItemId: 'save-selection', selectionText: 'fallback text' }, { id: 1 })

    const [, init] = fetchMock.mock.calls[0]
    expect(JSON.parse(init.body)).toEqual({ body: 'fallback text' })
  })

  it('downloads and uploads a right-clicked image', async () => {
    await signIn()
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(new Response(new Blob(['bytes'], { type: 'image/png' }), { status: 200 }))
      .mockResolvedValueOnce(jsonResponse({ successful: true, messages: [] }))
    vi.stubGlobal('fetch', fetchMock)

    await handleMenuClick({ menuItemId: 'save-image', srcUrl: 'https://example.com/photo.png' }, {})

    expect(fetchMock).toHaveBeenCalledWith('https://example.com/photo.png')
    const [, init] = fetchMock.mock.calls[1]
    expect(init.body).toBeInstanceOf(FormData)
    expect(init.body.get('file').name).toBe('photo.png')
  })

  it('opens the popup and skips capture when signed out', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)

    await handleMenuClick({ menuItemId: 'save-page', pageUrl: 'https://example.com' }, {})

    expect(chrome.action.openPopup).toHaveBeenCalled()
    expect(fetchMock).not.toHaveBeenCalled()
    expect(chrome.action.setBadgeText).toHaveBeenCalledWith({ text: '!' })
  })

  it('saves the page under the browser namespace (Firefox)', async () => {
    const { browser } = installBrowserMock()
    await browser.storage.sync.set({ apiBaseUrl: 'http://test' })
    await browser.storage.local.set({ sessionToken: 'session-token' })
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ successful: true, messages: [] }))
    vi.stubGlobal('fetch', fetchMock)

    await handleMenuClick({ menuItemId: 'save-page', pageUrl: 'https://example.com' }, {})

    const [, init] = fetchMock.mock.calls[0]
    expect(JSON.parse(init.body)).toEqual({ body: 'https://example.com', type: 'link' })
    expect(browser.action.setBadgeText).toHaveBeenCalledWith({ text: '✓' })
  })
})