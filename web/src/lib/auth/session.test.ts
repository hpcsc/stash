import { beforeEach, describe, expect, it, vi } from 'vitest'
import { get } from 'svelte/store'

const client = vi.hoisted(() => {
  let unauthorizedHandler: (() => void) | null = null
  return {
    registerUnauthorizedHandler: (handler: (() => void) | null) => {
      unauthorizedHandler = handler
    },
    unauthorizedHandler: () => unauthorizedHandler,
  }
})

vi.mock('../api/client', () => ({
  setToken: vi.fn(),
  setUnauthorizedHandler: client.registerUnauthorizedHandler,
  signIn: vi.fn(async () => 'session-token'),
  signOut: vi.fn(async () => {}),
}))

import { signIn, signOut, token } from './session'

describe('session', () => {
  beforeEach(() => {
    localStorage.clear()
    token.set(null)
  })

  it('signIn stores the token and persists it', async () => {
    await signIn('google-token')

    expect(get(token)).toBe('session-token')
    expect(localStorage.getItem('session_token')).toBe('session-token')
  })

  it('signOut clears the token and persistence', async () => {
    await signIn('google-token')
    await signOut()

    expect(get(token)).toBeNull()
    expect(localStorage.getItem('session_token')).toBeNull()
  })

  it('clears the token when the client reports an unauthorized request', async () => {
    await signIn('google-token')
    expect(get(token)).toBe('session-token')

    const handler = client.unauthorizedHandler()
    expect(handler).not.toBeNull()
    handler!()

    expect(get(token)).toBeNull()
    expect(localStorage.getItem('session_token')).toBeNull()
  })
})
