import { afterEach, describe, expect, it } from 'vitest'
import { browser } from '../src/browser.js'
import { installBrowserMock, installChromeMock } from './webext-mock.js'

describe('browser', () => {
  afterEach(() => {
    delete globalThis.browser
    delete globalThis.chrome
  })

  it('falls back to chrome when browser is absent', () => {
    const { chrome } = installChromeMock()

    expect(browser.runtime.id).toBe(chrome.runtime.id)
  })

  it('prefers browser over chrome', () => {
    const { browser: firefox } = installBrowserMock()

    expect(browser.runtime.id).toBe(firefox.runtime.id)
  })
})
