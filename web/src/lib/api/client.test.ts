import { afterEach, describe, expect, it, vi } from 'vitest'
import { setToken, signIn, signOut } from './client'

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
})
