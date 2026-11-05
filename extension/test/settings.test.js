import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  apiUrl,
  checkServer,
  defaultApiBaseUrl,
  getApiBaseUrl,
  getGoogleClientId,
  normalizeApiBaseUrl,
  setApiBaseUrl,
  SettingsError,
} from '../src/settings.js'
import { installChromeMock } from './webext-mock.js'

function jsonResponse(body, status = 200) {
  return new Response(JSON.stringify(body), { status })
}

describe('settings', () => {
  let chrome
  beforeEach(() => {
    ({ chrome } = installChromeMock())
  })
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  describe('normalizeApiBaseUrl', () => {
    it('adds https to a bare host', () => {
      expect(normalizeApiBaseUrl('stash.saphix.app')).toBe('https://stash.saphix.app')
    })

    it('keeps http for a local host', () => {
      expect(normalizeApiBaseUrl('localhost:3333')).toBe('http://localhost:3333')
      expect(normalizeApiBaseUrl('http://127.0.0.1:3333')).toBe('http://127.0.0.1:3333')
    })

    it('drops a trailing slash, query, and fragment', () => {
      expect(normalizeApiBaseUrl('https://stash.saphix.app/?x=1#top')).toBe('https://stash.saphix.app')
    })

    it('drops a trailing /api so the prefix is not doubled', () => {
      expect(normalizeApiBaseUrl('https://stash.saphix.app/api')).toBe('https://stash.saphix.app')
      expect(normalizeApiBaseUrl('https://example.com/stash/api/')).toBe('https://example.com/stash')
    })

    it('keeps a base path', () => {
      expect(normalizeApiBaseUrl('https://example.com/stash')).toBe('https://example.com/stash')
    })

    it('rejects an empty value', () => {
      expect(() => normalizeApiBaseUrl('   ')).toThrow(SettingsError)
      expect(() => normalizeApiBaseUrl('   ')).toThrow('Enter the Stash server URL')
    })

    it('rejects a non-http scheme', () => {
      expect(() => normalizeApiBaseUrl('ftp://example.com')).toThrow('Use an http or https URL')
    })

    it('rejects a malformed URL', () => {
      expect(() => normalizeApiBaseUrl('https://exa mple.com')).toThrow('Enter a valid server URL')
    })
  })

  describe('stored server', () => {
    it('defaults to the hosted server when nothing is stored', async () => {
      expect(await getApiBaseUrl()).toBe(defaultApiBaseUrl)
    })

    it('stores the normalized URL and reports the change', async () => {
      const first = await setApiBaseUrl('stash.example.com/')
      expect(first).toEqual({ url: 'https://stash.example.com', changed: true })
      expect(await getApiBaseUrl()).toBe('https://stash.example.com')

      const second = await setApiBaseUrl('https://stash.example.com')
      expect(second.changed).toBe(false)
    })

    it('builds API URLs from the stored server', async () => {
      await chrome.storage.sync.set({ apiBaseUrl: 'http://test' })

      expect(await apiUrl('/items')).toBe('http://test/api/items')
    })
  })

  describe('google client id', () => {
    it('reads the client ID from the configured server', async () => {
      await chrome.storage.sync.set({ apiBaseUrl: 'https://stash.example.com' })
      const fetchMock = vi.fn().mockResolvedValue(
        jsonResponse({ successful: true, messages: [], data: { googleClientId: 'client-id' } }),
      )
      vi.stubGlobal('fetch', fetchMock)

      expect(await getGoogleClientId()).toBe('client-id')
      expect(fetchMock).toHaveBeenCalledWith('https://stash.example.com/api/auth/config')
    })

    it('reports an unreachable server', async () => {
      await chrome.storage.sync.set({ apiBaseUrl: 'https://stash.example.com' })
      vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('offline')))

      await expect(getGoogleClientId()).rejects.toThrow(
        'Could not reach https://stash.example.com. Check the server URL in settings.',
      )
    })

    it('reports an HTTP error', async () => {
      await chrome.storage.sync.set({ apiBaseUrl: 'https://stash.example.com' })
      vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({}, 502)))

      await expect(getGoogleClientId()).rejects.toThrow('https://stash.example.com returned HTTP 502')
    })

    it('reports a server without a client ID', async () => {
      await chrome.storage.sync.set({ apiBaseUrl: 'https://stash.example.com' })
      vi.stubGlobal('fetch', vi.fn().mockResolvedValue(
        jsonResponse({ successful: true, messages: [], data: { googleClientId: '' } }),
      ))

      await expect(getGoogleClientId()).rejects.toThrow(
        'https://stash.example.com has no Google client ID configured',
      )
    })
  })

  describe('checkServer', () => {
    it('normalizes the URL without storing it', async () => {
      vi.stubGlobal('fetch', vi.fn().mockResolvedValue(
        jsonResponse({ successful: true, messages: [], data: { googleClientId: 'client-id' } }),
      ))

      const result = await checkServer('stash.example.com/api')

      expect(result).toEqual({ url: 'https://stash.example.com', clientId: 'client-id' })
      expect(await getApiBaseUrl()).toBe(defaultApiBaseUrl)
    })
  })
})
