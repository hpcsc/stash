export const menus = [
  { id: 'save-selection', title: 'Save selection to Stash', contexts: ['selection'] },
  { id: 'save-image', title: 'Save image to Stash', contexts: ['image'] },
  { id: 'save-page', title: 'Save page to Stash', contexts: ['page'] },
]

export function installMenus() {
  for (const menu of menus) {
    chrome.contextMenus.create(menu)
  }
}
