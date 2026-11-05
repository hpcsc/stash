import { createHash } from 'node:crypto'

export const firefoxId = 'stash@saphix.app'

export function chromeExtensionId(publicKeyBase64) {
  const digest = createHash('sha256').update(Buffer.from(publicKeyBase64, 'base64')).digest('hex')
  return [...digest.slice(0, 32)]
    .map((digit) => String.fromCharCode(97 + Number.parseInt(digit, 16)))
    .join('')
}

export function buildManifest(base, target, { version, chromeKey }) {
  const manifest = structuredClone(base)
  manifest.version = version

  if (target === 'chrome') {
    delete manifest.browser_specific_settings
    manifest.background = { service_worker: 'src/background.js', type: 'module' }
    if (chromeKey) manifest.key = chromeKey
    return manifest
  }

  if (target === 'firefox') {
    manifest.background = { scripts: ['src/background.js'], type: 'module' }
    manifest.browser_specific_settings = {
      ...manifest.browser_specific_settings,
      gecko: {
        ...manifest.browser_specific_settings?.gecko,
        id: firefoxId,
      },
    }
    return manifest
  }

  throw new Error(`unknown build target: ${target}`)
}
