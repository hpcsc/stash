import { signInWithGoogle, signOut } from '../src/auth.js'
import { browser } from '../src/browser.js'
import { getToken } from '../src/storage.js'

const statusEl = document.getElementById('status')
const signInBtn = document.getElementById('sign-in')
const signOutBtn = document.getElementById('sign-out')

async function render() {
  await browser.action.setBadgeText({ text: '' })
  const token = await getToken()
  const signedIn = Boolean(token)
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

render()
