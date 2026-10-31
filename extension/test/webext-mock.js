import { vi } from 'vitest'

function buildMock() {
  const data = new Map()

  const api = {
    runtime: {
      id: 'test-extension-id',
      onInstalled: { addListener: vi.fn() },
    },
    storage: {
      local: {
        async get(key) {
          if (typeof key === 'string') {
            return data.has(key) ? { [key]: data.get(key) } : {}
          }
          const out = {}
          for (const k of key ?? []) {
            if (data.has(k)) out[k] = data.get(k)
          }
          return out
        },
        async set(obj) {
          for (const [k, v] of Object.entries(obj)) data.set(k, v)
        },
        async remove(key) {
          data.delete(key)
        },
      },
    },
    identity: {
      launchWebAuthFlow: vi.fn(async () => ''),
    },
    contextMenus: {
      create: vi.fn(),
      onClicked: { addListener: vi.fn() },
    },
    action: {
      setBadgeText: vi.fn(async () => {}),
      setBadgeBackgroundColor: vi.fn(async () => {}),
      openPopup: vi.fn(async () => {}),
    },
  }

  return { api, data }
}

export function installChromeMock() {
  const { api, data } = buildMock()
  globalThis.chrome = api
  delete globalThis.browser
  return { chrome: api, data }
}

export function installBrowserMock() {
  const { api, data } = buildMock()
  globalThis.browser = api
  delete globalThis.chrome
  return { browser: api, data }
}
