import { afterEach, describe, expect, it, vi } from 'vitest'
import { readSession, signIn, signOut } from './session'

function jsonResponse(body: object, status = 200): Response {
  return new Response(JSON.stringify(body), { status })
}

describe('session', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('reads the signed-in state from the CLI', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({
        successful: true,
        messages: [],
        data: { signedIn: true, googleClientId: 'client-id', canSignOut: true },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const session = await readSession()

    expect(session).toEqual({ signedIn: true, googleClientId: 'client-id', canSignOut: true })
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/session')
    expect(init.method).toBe('GET')
  })

  it('sends the Google ID token to the CLI', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({
        successful: true,
        messages: [],
        data: { signedIn: true, googleClientId: 'client-id', canSignOut: true },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    await signIn('google-id-token')

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/session')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body)).toEqual({ idToken: 'google-id-token' })
  })

  it('deletes the session on sign-out', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ successful: true, messages: [] }))
    vi.stubGlobal('fetch', fetchMock)

    await signOut()

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/session')
    expect(init.method).toBe('DELETE')
  })

  it('throws the CLI message when sign-in fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse({ successful: false, messages: ['sign-in failed'] }, 401)),
    )

    await expect(signIn('bad-token')).rejects.toThrow('sign-in failed')
  })

  it('throws when the CLI does not answer with JSON', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('<html>Bad Gateway</html>', { status: 502 })))

    await expect(readSession()).rejects.toThrow('The event browser returned an unexpected response.')
  })
})
