import { redirectUri } from '../src/auth.js'
import { clearToken } from '../src/storage.js'
import {
  checkServer,
  defaultApiBaseUrl,
  getApiBaseUrl,
  setApiBaseUrl,
} from '../src/settings.js'

const form = document.getElementById('settings')
const serverInput = document.getElementById('server')
const statusEl = document.getElementById('status')
const redirectEl = document.getElementById('redirect')

function show(message, kind = '') {
  statusEl.textContent = message
  statusEl.className = `status ${kind}`
}

async function load() {
  serverInput.value = await getApiBaseUrl()
  redirectEl.textContent = redirectUri()
}

form.addEventListener('submit', async (event) => {
  event.preventDefault()
  try {
    const { url, changed } = await setApiBaseUrl(serverInput.value)
    serverInput.value = url
    if (changed) {
      await clearToken()
      show('Saved. Sign in again from the toolbar popup.', 'ok')
    } else {
      show('Saved.', 'ok')
    }
  } catch (error) {
    show(error.message, 'error')
  }
})

document.getElementById('reset').addEventListener('click', () => {
  serverInput.value = defaultApiBaseUrl
  show('Save to use the default server.')
})

document.getElementById('test').addEventListener('click', async () => {
  show('Checking…')
  try {
    const { url } = await checkServer(serverInput.value)
    show(`Connected to ${url}.`, 'ok')
  } catch (error) {
    show(error.message, 'error')
  }
})

load()
