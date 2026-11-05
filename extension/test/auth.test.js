import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { SignInFailed, signInWithGoogle, signOut } from '../src/auth.js'
import { installBrowserMock, installChromeMock } from './webext-mock.js'

function jsonResponse(body, status = 200) {
  return new Response(JSON.stringify(body), { status })
}

function configResponse(googleClientId = 'client-id.apps.googleusercontent.com') {
  return jsonResponse({ successful: true, messages: [], data: { googleClientId } })
}

describe('auth', () => {
  let chrome
  beforeEach(async () => {
    ({ chrome } = installChromeMock())
    await chrome.storage.sync.set({ apiBaseUrl: 'http://test' })
  })
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('signs in with Google and stores the session token', async () => {
    chrome.identity.launchWebAuthFlow.mockResolvedValue(
      'https://test-extension-id.chromiumapp.org/#id_token=google-token&scope=openid%20email',
    )
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(configResponse())
      .mockResolvedValueOnce(
        jsonResponse({ successful: true, messages: [], data: { token: 'session-token' } }),
      )
    vi.stubGlobal('fetch', fetchMock)

    await signInWithGoogle()

    expect(fetchMock.mock.calls[0][0]).toBe('http://test/api/auth/config')

    const { url, interactive } = chrome.identity.launchWebAuthFlow.mock.calls[0][0]
    const auth = new URL(url)
    expect(interactive).toBe(true)
    expect(auth.searchParams.get('client_id')).toBe('client-id.apps.googleusercontent.com')
    expect(auth.searchParams.get('redirect_uri')).toBe('https://test-extension-id.chromiumapp.org/')
    expect(auth.searchParams.get('response_type')).toBe('id_token')
    expect(auth.searchParams.get('scope')).toBe('openid email profile')
    expect(auth.searchParams.get('nonce')).toBeTruthy()

    const [apiUrl, init] = fetchMock.mock.calls[1]
    expect(apiUrl).toBe('http://test/api/auth/signin')
    expect(JSON.parse(init.body)).toEqual({ provider: 'google', idToken: 'google-token' })

    const stored = await chrome.storage.local.get('sessionToken')
    expect(stored.sessionToken).toBe('session-token')
  })

  it('rejects when Google does not return an ID token', async () => {
    chrome.identity.launchWebAuthFlow.mockResolvedValue('https://test-extension-id.chromiumapp.org/')
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(configResponse()))

    await expect(signInWithGoogle()).rejects.toBeInstanceOf(SignInFailed)
  })

  it('does not open Google when the server has no client ID', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(configResponse('')))

    await expect(signInWithGoogle()).rejects.toThrow('http://test has no Google client ID configured')
    expect(chrome.identity.launchWebAuthFlow).not.toHaveBeenCalled()
  })

  it('signOut clears the local token even when the API call fails', async () => {
    await chrome.storage.local.set({ sessionToken: 'session-token' })
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('offline')))

    await signOut()

    const stored = await chrome.storage.local.get('sessionToken')
    expect(stored.sessionToken).toBeUndefined()
  })

  it('uses getRedirectURL for the redirect URI when running in Firefox', async () => {
    const { browser } = installBrowserMock()
    await browser.storage.sync.set({ apiBaseUrl: 'http://test' })
    browser.identity.getRedirectURL = vi.fn(() => 'https://test-extension-id.extensions.allizom.org/')
    browser.identity.launchWebAuthFlow.mockResolvedValue(
      'https://test-extension-id.extensions.allizom.org/#id_token=google-token',
    )
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(configResponse())
      .mockResolvedValueOnce(
        jsonResponse({ successful: true, messages: [], data: { token: 'session-token' } }),
      )
    vi.stubGlobal('fetch', fetchMock)

    await signInWithGoogle()

    const { url } = browser.identity.launchWebAuthFlow.mock.calls[0][0]
    expect(new URL(url).searchParams.get('redirect_uri')).toBe('https://test-extension-id.extensions.allizom.org/')
  })
})
