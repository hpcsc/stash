import { browser } from './browser.js'
import { saveImage, saveLink } from './api.js'
import { markSaved, markFailed } from './badge.js'
import { getToken } from './storage.js'
import { stageCapture } from './staged.js'

export async function handleMenuClick(info, tab) {
  try {
    if (!(await getToken())) {
      await openPopup()
      await markFailed()
      return
    }
    const shouldMark = await capture(info, tab)
    if (shouldMark !== false) {
      await markSaved()
    }
  } catch (error) {
    console.error(error)
    await markFailed()
  }
}

async function capture(info, tab) {
  switch (info.menuItemId) {
    case 'save-selection':
      await stageSelection(info, tab)
      return false // The picker handles the badge.
    case 'save-image':
      await saveImageFromUrl(info.srcUrl)
      return true
    case 'save-page':
      await saveLink(info.pageUrl || tab?.url)
      return true
  }
}

async function readPageSelection(tabId, frameId) {
  try {
    const targets = [{ tabId }]
    if (frameId !== undefined && frameId !== 0) {
      targets[0].frameIds = [frameId]
    }

    const { readSelection } = await import('./selection.js')
    const [injection] = await browser.scripting.executeScript({
      target: targets[0],
      func: readSelection,
    })
    return injection?.result
  } catch (err) {
    // Injection is refused on browser-internal pages; fall back.
    console.error('readPageSelection failed:', err)
    return null
  }
}

export async function downloadSelectionImages(images) {
  const results = []
  for (const img of images) {
    // If the injected reader captured blob bytes, use them directly.
    if (img._blobBytes) {
      const blob = new Blob([new Uint8Array(img._blobBytes)], { type: img._blobType || 'image/png' })
      results.push({ ok: true, blob, name: img._blobName || imageName(img.src) })
      continue
    }
    try {
      const response = await fetch(img.src)
      if (!response.ok) throw new Error(`download failed: ${response.status}`)
      const blob = await response.blob()
      results.push({ ok: true, blob, name: imageName(img.src) })
    } catch {
      results.push({ ok: false, src: img.src })
    }
  }
  return results
}

export function resolveImageTokens(markdown, downloads) {
  const blobs = []
  let k = 0
  const resolved = markdown.replace(/@stash-image-(\d+)/g, (_match, n) => {
    const idx = Number.parseInt(n, 10)
    const result = downloads[idx]
    if (result && result.ok) {
      blobs.push({ blob: result.blob, name: result.name })
      return `@stash-image-${k++}`
    }
    return result?.src || ''
  })
  return { markdown: resolved, blobs }
}

async function saveImageFromUrl(url) {
  const response = await fetch(url)
  if (!response.ok) {
    throw new Error('failed to download the image')
  }
  const blob = await response.blob()
  await saveImage(blob, imageName(url))
}

async function stageSelection(info, tab) {
  // Read the selection, same as saveSelection but stage it instead of saving.
  const result = await readPageSelection(tab.id, info.frameId)

  let markdown = ''
  let images = []

  if (result && (result.markdown || result.images.length > 0)) {
    markdown = result.markdown
    images = result.images
  } else if (info.selectionText) {
    markdown = info.selectionText
  }

  if (!markdown && images.length === 0) return

  await stageCapture({ markdown, images, tabUrl: tab?.url || info.pageUrl })
  await openPopup()
  // Do not mark saved — the picker handles the badge.
}

function imageName(url) {
  try {
    const path = new URL(url).pathname
    return path.substring(path.lastIndexOf('/') + 1) || 'image'
  } catch {
    return 'image'
  }
}

async function openPopup() {
  if (browser.action.openPopup) {
    await browser.action.openPopup()
  }
}