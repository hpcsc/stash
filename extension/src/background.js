import { browser } from './browser.js'
import { installMenus } from './menu.js'
import { handleMenuClick } from './dispatch.js'

browser.runtime.onInstalled.addListener(installMenus)
browser.contextMenus.onClicked.addListener(handleMenuClick)
