import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import { buildManifest, chromeExtensionId } from '../scripts/manifest.mjs'

const base = JSON.parse(readFileSync(new URL('../manifest.json', import.meta.url), 'utf8'))
const chromeKey = readFileSync(new URL('../chrome-key.pub', import.meta.url), 'utf8').trim()

const options = { version: '2.3.4', chromeKey }

describe('buildManifest', () => {
  it('builds a Chrome manifest with the fixed key', () => {
    const manifest = buildManifest(base, 'chrome', options)

    expect(manifest.version).toBe('2.3.4')
    expect(manifest.background).toEqual({ service_worker: 'src/background.js', type: 'module' })
    expect(manifest.key).toBe(chromeKey)
    expect(manifest.browser_specific_settings).toBeUndefined()
  })

  it('builds a Firefox manifest with an event page', () => {
    const manifest = buildManifest(base, 'firefox', options)

    expect(manifest.version).toBe('2.3.4')
    expect(manifest.background).toEqual({ scripts: ['src/background.js'], type: 'module' })
    expect(manifest.browser_specific_settings.gecko.id).toBe('stash@saphix.app')
    expect(manifest.browser_specific_settings.gecko.data_collection_permissions).toEqual({
      required: ['authenticationInfo', 'browsingActivity', 'websiteContent'],
    })
    expect(manifest.key).toBeUndefined()
  })

  it('does not change the source manifest', () => {
    buildManifest(base, 'chrome', options)

    expect(base.background).toEqual({
      scripts: ['src/background.js'],
      service_worker: 'src/background.js',
      type: 'module',
    })
  })

  it('rejects an unknown target', () => {
    expect(() => buildManifest(base, 'safari', options)).toThrow('unknown build target: safari')
  })
})

describe('chromeExtensionId', () => {
  it('stays fixed for the committed key', () => {
    expect(chromeExtensionId(chromeKey)).toBe('kdfpkmjkeamfogldcphiajinboajjced')
  })
})
