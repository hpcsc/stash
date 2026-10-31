import { handleMenuClick } from './dispatch.js'
import { installMenus } from './menu.js'

chrome.runtime.onInstalled.addListener(installMenus)
chrome.contextMenus.onClicked.addListener(handleMenuClick)
