import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError, saveImage, saveLink, saveText } from '../src/api.js'
import { installChromeMock } from './chrome-mock.js'

vi.mock('../src/config.js', () => ({
  apiBaseUrl: 'http://test',
  googleClientId: '',
}))

function jsonResponse(body, status = 200) {
  return new Response(JSON.stringify(body), { status })
}

describe('api', () => {
  beforeEach(() => {
    installChromeMock()
  })
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('saveText posts the text to /items', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ successful: true, messages: [] }))
    vi.stubGlobal('fetch', fetchMock)

    await saveText('remember this')

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://test/items')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body)).toEqual({ body: 'remember this' })
  })

  it('saveLink posts the URL as a link', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ successful: true, messages: [] }))
    vi.stubGlobal('fetch', fetchMock)

    await saveLink('https://example.com')

    const [, init] = fetchMock.mock.calls[0]
    expect(JSON.parse(init.body)).toEqual({ body: 'https://example.com', type: 'link' })
  })

  it('saveImage uploads the image as multipart', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ successful: true, messages: [] }))
    vi.stubGlobal('fetch', fetchMock)

    await saveImage(new Blob(['bytes'], { type: 'image/png' }), 'photo.png')

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://test/items')
    expect(init.method).toBe('POST')
    expect(init.body).toBeInstanceOf(FormData)
    expect(init.body.get('file').name).toBe('photo.png')
  })

  it('sends the bearer token when signed in', async () => {
    await chrome.storage.local.set({ sessionToken: 'session-token' })
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ successful: true, messages: [] }))
    vi.stubGlobal('fetch', fetchMock)

    await saveText('hello')

    const [, init] = fetchMock.mock.calls[0]
    expect(init.headers.Authorization).toBe('Bearer session-token')
  })

  it('throws ApiError with the backend message on failure', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ successful: false, messages: ['bad request'] }, 400)))

    await expect(saveText('hello')).rejects.toThrow('bad request')
    await expect(saveText('hello')).rejects.toBeInstanceOf(ApiError)
  })
})
