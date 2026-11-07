import { browser } from './browser.js'

const stagedKey = 'pending-selection'
const stagingWindowMs = 10 * 60 * 1000 // 10 minutes

export async function stageCapture(capture) {
  await browser.storage.session.set({
    [stagedKey]: { ...capture, capturedAt: Date.now() },
  })
}

export async function readStaged() {
  const stored = await browser.storage.session.get(stagedKey)
  const data = stored[stagedKey]
  if (!data) return null

  if (Date.now() - data.capturedAt > stagingWindowMs) {
    await clearStaged()
    return null
  }
  return data
}

export async function clearStaged() {
  await browser.storage.session.remove(stagedKey)
}