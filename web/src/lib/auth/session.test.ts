import { beforeEach, describe, expect, it, vi } from 'vitest'
import { get } from 'svelte/store'

vi.mock('../api/client', () => ({
  setToken: vi.fn(),
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
})
