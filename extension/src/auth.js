import { apiBaseUrl, googleClientId } from './config.js'
import { browser } from './browser.js'
import { clearToken, getToken, setToken } from './storage.js'

export class SignInFailed extends Error {
  constructor(message) {
    super(message)
    this.name = 'SignInFailed'
  }
}

export async function signInWithGoogle() {
  const redirectUri = browser.identity.getRedirectURL
    ? browser.identity.getRedirectURL()
    : `https://${browser.runtime.id}.chromiumapp.org/`
  const nonce = crypto.randomUUID()
  const authUrl = new URL('https://accounts.google.com/o/oauth2/v2/auth')
  authUrl.searchParams.set('client_id', googleClientId)
  authUrl.searchParams.set('redirect_uri', redirectUri)
  authUrl.searchParams.set('response_type', 'id_token')
  authUrl.searchParams.set('scope', 'openid email profile')
  authUrl.searchParams.set('nonce', nonce)

  const resultUrl = await browser.identity.launchWebAuthFlow({ url: authUrl.toString(), interactive: true })
  const fragment = (resultUrl ?? '').split('#')[1] ?? ''
  const idToken = new URLSearchParams(fragment).get('id_token')
  if (!idToken) {
    throw new SignInFailed('Google sign-in did not return an ID token')
  }

  const response = await fetch(`${apiBaseUrl}/auth/signin`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ provider: 'google', idToken }),
  })

  let envelope
  try {
    envelope = await response.json()
  } catch {
    throw new SignInFailed('sign-in failed')
  }
  if (envelope.successful !== true) {
    const messages = envelope.messages ?? []
    throw new SignInFailed(messages.length > 0 ? messages.join('\n') : 'sign-in failed')
  }

  await setToken(envelope.data.token)
  return envelope.data.token
}

export async function signOut() {
  const token = await getToken()
  if (token) {
    try {
      await fetch(`${apiBaseUrl}/auth/signout`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      })
    } catch {
      // a failed sign-out must still clear the local token
    }
  }
  await clearToken()
}
