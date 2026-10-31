import { beforeEach, describe, expect, it } from 'vitest'
import { installMenus, menus } from '../src/menu.js'
import { installChromeMock } from './chrome-mock.js'

describe('installMenus', () => {
  let chrome
  beforeEach(() => {
    ({ chrome } = installChromeMock())
  })

  it('creates a context menu for each capture type', () => {
    installMenus()

    expect(chrome.contextMenus.create).toHaveBeenCalledTimes(menus.length)
    for (const menu of menus) {
      expect(chrome.contextMenus.create).toHaveBeenCalledWith(menu)
    }
  })
})
