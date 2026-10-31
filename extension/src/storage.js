const tokenKey = 'sessionToken'

export async function getToken() {
  const stored = await chrome.storage.local.get(tokenKey)
  return stored[tokenKey] ?? null
}

export async function setToken(token) {
  await chrome.storage.local.set({ [tokenKey]: token })
}

export async function clearToken() {
  await chrome.storage.local.remove(tokenKey)
}
