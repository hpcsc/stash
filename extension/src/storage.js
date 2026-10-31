import { browser } from './browser.js'

const tokenKey = 'sessionToken'

export async function getToken() {
  const stored = await browser.storage.local.get(tokenKey)
  return stored[tokenKey] ?? null
}

export async function setToken(token) {
  await browser.storage.local.set({ [tokenKey]: token })
}

export async function clearToken() {
  await browser.storage.local.remove(tokenKey)
}
