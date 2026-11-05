import { signInWithGoogle, signOut } from '../src/auth.js'
import { browser } from '../src/browser.js'
import { getApiBaseUrl } from '../src/settings.js'
import { getToken } from '../src/storage.js'

const statusEl = document.getElementById('status')
const serverEl = document.getElementById('server')
const signInBtn = document.getElementById('sign-in')
const signOutBtn = document.getElementById('sign-out')
const settingsBtn = document.getElementById('settings')

async function render() {
  await browser.action.setBadgeText({ text: '' })
  const token = await getToken()
  const signedIn = Boolean(token)
  serverEl.textContent = (await getApiBaseUrl()).replace(/^https?:\/\//, '')
  statusEl.textContent = signedIn ? 'Signed in' : 'Not signed in'
  signInBtn.hidden = signedIn
  signOutBtn.hidden = !signedIn
  signInBtn.disabled = false
}

signInBtn.addEventListener('click', async () => {
  signInBtn.disabled = true
  statusEl.textContent = 'Signing in…'
  try {
    await signInWithGoogle()
  } catch (error) {
    statusEl.textContent = error.message ?? 'sign-in failed'
    signInBtn.disabled = false
    return
  }
  await render()
})

signOutBtn.addEventListener('click', async () => {
  await signOut()
  await render()
})

settingsBtn.addEventListener('click', async () => {
  await browser.runtime.openOptionsPage()
  window.close()
})

render()
