import { appendToNote, cancelAppend, getStagedCapture, loadNotes } from '../src/append.js'
import { markSaved } from '../src/badge.js'

const preview = document.getElementById('preview')
const status = document.getElementById('status')
const noteList = document.getElementById('note-list')
const errorElt = document.getElementById('error')
const emptyElt = document.getElementById('empty')
const expiredElt = document.getElementById('expired')
const cancelBtn = document.getElementById('cancel')

let staged = null
let selectedId = null
let saving = false

async function init() {
  staged = await getStagedCapture()
  if (!staged) {
    showExpired()
    return
  }

  const previewText = staged.markdown
    ? staged.markdown.length > 100 ? staged.markdown.slice(0, 100) + '\u2026' : staged.markdown
    : staged.images.length > 0 ? `${staged.images.length} image(s)\u2026` : ''
  preview.textContent = previewText

  try {
    const notes = await loadNotes()
    if (notes.length === 0) {
      showEmpty()
      return
    }

    renderList(notes)
    status.textContent = `${notes.length} note(s)`
  } catch (e) {
    status.textContent = ''
    showError('Could not load notes.')
  }
}

function renderList(notes) {
  noteList.innerHTML = ''
  for (const note of notes) {
    const li = document.createElement('li')
    li.className = 'note-row'
    li.setAttribute('role', 'button')
    li.tabIndex = 0

    const body = document.createElement('div')
    body.className = 'note-body'
    body.textContent = note.preview

    const meta = document.createElement('div')
    meta.className = 'note-meta'
    meta.textContent = formatDate(note.capturedAt)

    li.append(body, meta)

    li.addEventListener('click', () => select(note, li))
    li.addEventListener('keydown', (e) => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault()
        select(note, li)
      }
    })

    noteList.appendChild(li)
  }
}

async function select(note, row) {
  if (saving) return

  // Deselect previous.
  const prev = noteList.querySelector('.selected')
  if (prev) prev.classList.remove('selected')

  // Clear previous inline actions.
  noteList.querySelectorAll('.note-actions, .note-progress, .done').forEach((e) => e.remove())

  row.classList.add('selected')
  selectedId = note.id

  // Show progress.
  const progress = document.createElement('div')
  progress.className = 'note-progress'
  progress.textContent = 'Adding selection\u2026'
  row.appendChild(progress)

  saving = true
  setButtonsDisabled(true)

  try {
    await appendToNote(note.id, staged)
    progress.remove()
    const done = document.createElement('div')
    done.className = 'done'
    done.textContent = 'Added \u2714'
    row.appendChild(done)
    await markSaved()
  } catch (e) {
    progress.remove()
    if (e.name === 'ConflictError') {
      showRetry(row, note)
    } else {
      showError(e.message || 'Something went wrong.')
      row.classList.remove('selected')
    }
  } finally {
    saving = false
    setButtonsDisabled(false)
  }
}

function showRetry(row, note) {
  const actions = document.createElement('div')
  actions.className = 'note-actions'

  const retryBtn = document.createElement('button')
  retryBtn.className = 'primary'
  retryBtn.textContent = 'Retry'
  retryBtn.addEventListener('click', (e) => {
    e.stopPropagation()
    actions.remove()
    select(note, row)
  })

  const cancelRowBtn = document.createElement('button')
  cancelRowBtn.textContent = 'Cancel'
  cancelRowBtn.addEventListener('click', (e) => {
    e.stopPropagation()
    actions.remove()
    row.classList.remove('selected')
    hideError()
  })

  actions.append(retryBtn, cancelRowBtn)
  row.appendChild(actions)
  showError('The note changed after you opened this list. Try again.')
}

function showError(message) {
  errorElt.textContent = message
  errorElt.hidden = false
}

function hideError() {
  errorElt.hidden = true
}

function showEmpty() {
  status.hidden = true
  emptyElt.hidden = false
}

function showExpired() {
  status.hidden = true
  expiredElt.hidden = false
}

function setButtonsDisabled(disabled) {
  for (const btn of document.querySelectorAll('button')) {
    btn.disabled = disabled
  }
}

function formatDate(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: d.getFullYear() !== new Date().getFullYear() ? 'numeric' : undefined })
}

cancelBtn.addEventListener('click', async () => {
  await cancelAppend()
  window.close()
})

init()