import { browser } from './browser.js'

export const menus = [
  { id: 'save-selection', title: 'Save selection to Stash', contexts: ['selection'] },
  { id: 'append-selection', title: 'Add selection to a note\u2026', contexts: ['selection'] },
  { id: 'save-image', title: 'Save image to Stash', contexts: ['image'] },
  { id: 'save-page', title: 'Save page to Stash', contexts: ['page'] },
]

export function installMenus() {
  for (const menu of menus) {
    browser.contextMenus.create(menu)
  }
}
