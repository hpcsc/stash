import { browser } from './browser.js'

export const defaultApiBaseUrl = 'https://stash.saphix.app'
export const apiPrefix = '/api'

const baseUrlKey = 'apiBaseUrl'
const anyScheme = /^[a-z][a-z0-9+.-]*:\/\//i
const httpScheme = /^https?:\/\//i
const localHost = /^(localhost|127\.0\.0\.1|\[::1\])(:\d+)?(\/|$)/i

export class SettingsError extends Error {
  constructor(message) {
    super(message)
    this.name = 'SettingsError'
  }
}

export function normalizeApiBaseUrl(raw) {
  const value = (raw ?? '').trim()
  if (!value) {
    throw new SettingsError('Enter the Stash server URL')
  }
  if (anyScheme.test(value) && !httpScheme.test(value)) {
    throw new SettingsError('Use an http or https URL')
  }

  const scheme = anyScheme.test(value) ? '' : localHost.test(value) ? 'http://' : 'https://'
  const withScheme = `${scheme}${value}`
  let url
  try {
    url = new URL(withScheme)
  } catch {
    throw new SettingsError('Enter a valid server URL')
  }
  if (!url.hostname) {
    throw new SettingsError('Enter a valid server URL')
  }

  url.hash = ''
  url.search = ''
  let path = url.pathname.replace(/\/+$/, '')
  if (path === apiPrefix) {
    path = ''
  } else if (path.endsWith(apiPrefix)) {
    path = path.slice(0, -apiPrefix.length)
  }
  url.pathname = path
  return url.toString().replace(/\/$/, '')
}

export async function getApiBaseUrl() {
  const stored = await browser.storage.sync.get(baseUrlKey)
  return stored[baseUrlKey] || defaultApiBaseUrl
}

export async function setApiBaseUrl(raw) {
  const url = normalizeApiBaseUrl(raw)
  const previous = await getApiBaseUrl()
  await browser.storage.sync.set({ [baseUrlKey]: url })
  return { url, changed: url !== previous }
}

export async function apiUrl(path) {
  return `${await getApiBaseUrl()}${apiPrefix}${path}`
}

async function fetchClientId(base) {
  let response
  try {
    response = await fetch(`${base}${apiPrefix}/auth/config`)
  } catch {
    throw new SettingsError(`Could not reach ${base}. Check the server URL in settings.`)
  }
  if (!response.ok) {
    throw new SettingsError(`${base} returned HTTP ${response.status}`)
  }

  let envelope
  try {
    envelope = await response.json()
  } catch {
    throw new SettingsError(`${base} did not return JSON`)
  }

  const clientId = envelope?.data?.googleClientId
  if (envelope?.successful !== true || !clientId) {
    throw new SettingsError(`${base} has no Google client ID configured`)
  }
  return clientId
}

export async function getGoogleClientId() {
  return fetchClientId(await getApiBaseUrl())
}

export async function checkServer(raw) {
  const url = normalizeApiBaseUrl(raw)
  const clientId = await fetchClientId(url)
  return { url, clientId }
}
