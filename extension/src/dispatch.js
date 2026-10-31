import { browser } from './browser.js'
import { saveImage, saveLink, saveText } from './api.js'
import { getToken } from './storage.js'

export async function handleMenuClick(info, tab) {
  try {
    if (!(await getToken())) {
      await openPopup()
      await markFailed()
      return
    }
    await capture(info, tab)
    await markSaved()
  } catch (error) {
    console.error(error)
    await markFailed()
  }
}

async function capture(info, tab) {
  switch (info.menuItemId) {
    case 'save-selection':
      await saveText(info.selectionText)
      break
    case 'save-image':
      await saveImageFromUrl(info.srcUrl)
      break
    case 'save-page':
      await saveLink(info.pageUrl || tab?.url)
      break
  }
}

async function saveImageFromUrl(url) {
  const response = await fetch(url)
  if (!response.ok) {
    throw new Error('failed to download the image')
  }
  const blob = await response.blob()
  await saveImage(blob, imageName(url))
}

function imageName(url) {
  try {
    const path = new URL(url).pathname
    return path.substring(path.lastIndexOf('/') + 1) || 'image'
  } catch {
    return 'image'
  }
}

async function markSaved() {
  await browser.action.setBadgeBackgroundColor({ color: '#2f855a' })
  await browser.action.setBadgeText({ text: '✓' })
}

async function markFailed() {
  await browser.action.setBadgeBackgroundColor({ color: '#c53030' })
  await browser.action.setBadgeText({ text: '!' })
}

async function openPopup() {
  if (browser.action.openPopup) {
    await browser.action.openPopup()
  }
}
