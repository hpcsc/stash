import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { handleMenuClick } from '../src/dispatch.js'
import { installBrowserMock, installChromeMock } from './webext-mock.js'

vi.mock('../src/config.js', () => ({
  apiBaseUrl: 'http://test',
  googleClientId: '',
}))

function jsonResponse(body, status = 200) {
  return new Response(JSON.stringify(body), { status })
}

describe('handleMenuClick', () => {
  let chrome
  beforeEach(() => {
    ({ chrome } = installChromeMock())
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

    const [, init] = fetchMock.mock.calls[0]
    expect(JSON.parse(init.body)).toEqual({ body: 'https://example.com', type: 'link' })
    expect(chrome.action.setBadgeText).toHaveBeenCalledWith({ text: '✓' })
  })

  it('saves the selection as text', async () => {
    await signIn()
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ successful: true, messages: [] }))
    vi.stubGlobal('fetch', fetchMock)

    await handleMenuClick({ menuItemId: 'save-selection', selectionText: 'hello' }, {})

    const [, init] = fetchMock.mock.calls[0]
    expect(JSON.parse(init.body)).toEqual({ body: 'hello' })
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
    await browser.storage.local.set({ sessionToken: 'session-token' })
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ successful: true, messages: [] }))
    vi.stubGlobal('fetch', fetchMock)

    await handleMenuClick({ menuItemId: 'save-page', pageUrl: 'https://example.com' }, {})

    const [, init] = fetchMock.mock.calls[0]
    expect(JSON.parse(init.body)).toEqual({ body: 'https://example.com', type: 'link' })
    expect(browser.action.setBadgeText).toHaveBeenCalledWith({ text: '✓' })
  })
})
