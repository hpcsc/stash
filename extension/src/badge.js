import { browser } from './browser.js'

export async function markSaved() {
  await browser.action.setBadgeBackgroundColor({ color: '#2f855a' })
  await browser.action.setBadgeText({ text: '✓' })
}

export async function markFailed() {
  await browser.action.setBadgeBackgroundColor({ color: '#c53030' })
  await browser.action.setBadgeText({ text: '!' })
}