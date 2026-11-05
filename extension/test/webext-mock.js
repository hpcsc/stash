import { vi } from 'vitest'

function storageArea(store) {
  return {
    async get(key) {
      if (typeof key === 'string') {
        return store.has(key) ? { [key]: store.get(key) } : {}
      }
      const out = {}
      for (const k of key ?? []) {
        if (store.has(k)) out[k] = store.get(k)
      }
      return out
    },
    async set(obj) {
      for (const [k, v] of Object.entries(obj)) store.set(k, v)
    },
    async remove(key) {
      for (const k of Array.isArray(key) ? key : [key]) store.delete(k)
    },
  }
}

function buildMock() {
  const local = new Map()
  const sync = new Map()

  const api = {
    runtime: {
      id: 'test-extension-id',
      onInstalled: { addListener: vi.fn() },
      openOptionsPage: vi.fn(async () => {}),
    },
    storage: {
      local: storageArea(local),
      sync: storageArea(sync),
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

  return { api, local, sync }
}

export function installChromeMock() {
  const { api, local, sync } = buildMock()
  globalThis.chrome = api
  delete globalThis.browser
  return { chrome: api, local, sync }
}

export function installBrowserMock() {
  const { api, local, sync } = buildMock()
  globalThis.browser = api
  delete globalThis.chrome
  return { browser: api, local, sync }
}
