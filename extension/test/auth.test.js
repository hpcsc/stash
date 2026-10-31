import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { SignInFailed, signInWithGoogle, signOut } from '../src/auth.js'
import { installChromeMock } from './chrome-mock.js'

vi.mock('../src/config.js', () => ({
  apiBaseUrl: 'http://test',
  googleClientId: 'client-id.apps.googleusercontent.com',
}))

function jsonResponse(body, status = 200) {
  return new Response(JSON.stringify(body), { status })
}

describe('auth', () => {
  let chrome
  beforeEach(() => {
    ({ chrome } = installChromeMock())
  })
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('signs in with Google and stores the session token', async () => {
    chrome.identity.launchWebAuthFlow.mockResolvedValue(
      'https://test-extension-id.chromiumapp.org/#id_token=google-token&scope=openid%20email',
    )
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ successful: true, messages: [], data: { token: 'session-token' } }),
    )
    vi.stubGlobal('fetch', fetchMock)

    await signInWithGoogle()

    const { url, interactive } = chrome.identity.launchWebAuthFlow.mock.calls[0][0]
    const auth = new URL(url)
    expect(interactive).toBe(true)
    expect(auth.searchParams.get('client_id')).toBe('client-id.apps.googleusercontent.com')
    expect(auth.searchParams.get('redirect_uri')).toBe('https://test-extension-id.chromiumapp.org/')
    expect(auth.searchParams.get('response_type')).toBe('id_token')
    expect(auth.searchParams.get('scope')).toBe('openid email profile')
    expect(auth.searchParams.get('nonce')).toBeTruthy()

    const [apiUrl, init] = fetchMock.mock.calls[0]
    expect(apiUrl).toBe('http://test/auth/signin')
    expect(JSON.parse(init.body)).toEqual({ provider: 'google', idToken: 'google-token' })

    const stored = await chrome.storage.local.get('sessionToken')
    expect(stored.sessionToken).toBe('session-token')
  })

  it('rejects when Google does not return an ID token', async () => {
    chrome.identity.launchWebAuthFlow.mockResolvedValue('https://test-extension-id.chromiumapp.org/')

    await expect(signInWithGoogle()).rejects.toBeInstanceOf(SignInFailed)
  })

  it('signOut clears the local token even when the API call fails', async () => {
    await chrome.storage.local.set({ sessionToken: 'session-token' })
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('offline')))

    await signOut()

    const stored = await chrome.storage.local.get('sessionToken')
    expect(stored.sessionToken).toBeUndefined()
  })
})
