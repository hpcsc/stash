import { afterEach, describe, expect, it, vi } from 'vitest'
import { setToken, signIn, signOut, listItems, captureText } from './client'

function jsonResponse(body: object, status = 200): Response {
  return new Response(JSON.stringify(body), { status })
}

describe('client', () => {
  afterEach(() => {
    setToken(null)
    vi.unstubAllGlobals()
  })

  it('signIn returns the session token', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ successful: true, messages: [], data: { token: 'session-token' } }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const token = await signIn('google-token')

    expect(token).toBe('session-token')
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://127.0.0.1:3333/auth/signin')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body)).toEqual({ provider: 'google', idToken: 'google-token' })
  })

  it('signIn throws the backend message on failure', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse({ successful: false, messages: ['sign-in failed'] }, 401)),
    )

    await expect(signIn('bad-token')).rejects.toThrow('sign-in failed')
  })

  it('signOut sends the bearer token', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ successful: true, messages: [] }))
    vi.stubGlobal('fetch', fetchMock)
    setToken('session-token')

    await signOut()

    const [, init] = fetchMock.mock.calls[0]
    expect(init.headers.Authorization).toBe('Bearer session-token')
  })

  it('listItems returns the items', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        jsonResponse({
          successful: true,
          messages: [],
          data: [{ id: '1', type: 'text', body: 'hello', state: 'inbox', capturedAt: '2026-01-01T00:00:00Z' }],
        }),
      ),
    )

    const items = await listItems()

    expect(items).toHaveLength(1)
    expect(items[0].body).toBe('hello')
  })

  it('captureText posts the body and returns the id', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ successful: true, messages: [], data: { id: 'item-1' } }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const id = await captureText('remember this')

    expect(id).toBe('item-1')
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://127.0.0.1:3333/items')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body)).toEqual({ body: 'remember this' })
  })
})
