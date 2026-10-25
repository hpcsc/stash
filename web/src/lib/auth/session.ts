import { writable } from 'svelte/store'
import { setToken, signIn as apiSignIn, signOut as apiSignOut } from '../api/client'

const STORAGE_KEY = 'session_token'

export const token = writable<string | null>(restore())

function restore(): string | null {
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored) setToken(stored)
  return stored
}

export async function signIn(idToken: string): Promise<void> {
  const newToken = await apiSignIn(idToken)
  setToken(newToken)
  localStorage.setItem(STORAGE_KEY, newToken)
  token.set(newToken)
}

export async function signOut(): Promise<void> {
  try {
    await apiSignOut()
  } finally {
    setToken(null)
    localStorage.removeItem(STORAGE_KEY)
    token.set(null)
  }
}
