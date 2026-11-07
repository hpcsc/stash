import { signInWithGoogle, signOut } from '../src/auth.js'
import { browser } from '../src/browser.js'
import { getApiBaseUrl } from '../src/settings.js'
import { getToken } from '../src/storage.js'
import { readStaged, clearStaged } from '../src/staged.js'
import { appendToNote, loadNotes } from '../src/append.js'
import { markSaved } from '../src/badge.js'

// Home view elements
const homeEl = document.getElementById('home')
const statusEl = document.getElementById('status')
const serverEl = document.getElementById('server')
const signInBtn = document.getElementById('sign-in')
const signOutBtn = document.getElementById('sign-out')
const settingsBtn = document.getElementById('settings')

// Picker view elements
const pickerEl = document.getElementById('picker')
const pickerBack = document.getElementById('picker-back')
const pickerPreview = document.getElementById('picker-preview')
const pickerLoading = document.getElementById('picker-loading')
const pickerError = document.getElementById('picker-error')
const pickerList = document.getElementById('picker-list')
const pickerEmpty = document.getElementById('picker-empty')

let staged = null

async function init() {
  await browser.action.setBadgeText({ text: '' })
  staged = await readStaged()

  if (staged) {
    showPicker()
    await loadPickerNotes()
  } else {
    showHome()
    await renderHome()
  }
}

// ---- Home ----

function showHome() {
  homeEl.hidden = false
  pickerEl.hidden = true
}

async function renderHome() {
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
  await renderHome()
})

signOutBtn.addEventListener('click', async () => {
  await signOut()
  await renderHome()
})

settingsBtn.addEventListener('click', async () => {
  await browser.runtime.openOptionsPage()
  window.close()
})

// ---- Picker ----

pickerBack.addEventListener('click', async () => {
  await clearStaged()
  staged = null
  showHome()
  await renderHome()
})

function showPicker() {
  homeEl.hidden = true
  pickerEl.hidden = false
}

function showPickerLoading() {
  pickerLoading.hidden = false
  pickerList.innerHTML = ''
  pickerEmpty.hidden = true
  pickerError.hidden = true
}

function showPickerError(message) {
  pickerError.textContent = message
  pickerError.hidden = false
  pickerLoading.hidden = true
}

async function loadPickerNotes() {
  showPickerLoading()

  const preview = staged.markdown
    ? staged.markdown.length > 80 ? staged.markdown.slice(0, 80) + '\u2026' : staged.markdown
    : staged.images.length > 0 ? `${staged.images.length} image(s)` : ''
  pickerPreview.textContent = preview || '(empty selection)'

  try {
    const notes = await loadNotes()
    if (notes.length === 0) {
      pickerLoading.hidden = true
      pickerEmpty.hidden = false
      return
    }
    renderNotes(notes)
  } catch (e) {
    showPickerError('Could not load notes.')
  }
}

function renderNotes(notes) {
  pickerLoading.hidden = true
  pickerList.innerHTML = ''

  for (const note of notes) {
    const li = document.createElement('li')
    li.className = 'picker-row'

    const body = document.createElement('div')
    body.className = 'picker-row-body'
    body.textContent = note.preview

    const meta = document.createElement('div')
    meta.className = 'picker-row-meta'
    meta.textContent = formatDate(note.capturedAt)

    li.append(body, meta)
    li.addEventListener('click', () => selectNote(note, li))
    pickerList.appendChild(li)
  }
}

async function selectNote(note, row) {
  // Disable all rows during the operation.
  const prev = pickerList.querySelector('.picking')
  if (prev) return

  row.classList.add('picking')
  const progress = document.createElement('div')
  progress.className = 'picker-row-progress'
  progress.textContent = 'Adding\u2026'
  row.appendChild(progress)

  try {
    await appendToNote(note.id, staged)
    staged = null
    await markSaved()
    window.close()
  } catch (e) {
    progress.remove()
    if (e.name === 'ConflictError') {
      showPickerError('The note changed. Try again — your selection is kept.')
    } else {
      showPickerError(e.message || 'Something went wrong.')
    }
    row.classList.remove('picking')
  }
}

function formatDate(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: d.getFullYear() !== new Date().getFullYear() ? 'numeric' : undefined })
}

init()