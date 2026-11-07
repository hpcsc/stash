// Logic for appending a selection to an existing note. Does not import DOM modules so
// it can be unit-tested without a browser environment.
import { listItems, getItem, editNote, ConflictError } from '../src/api.js'
import { downloadSelectionImages, resolveImageTokens } from '../src/dispatch.js'
import { readStaged, clearStaged } from '../src/staged.js'

const MAX_PREVIEW_LEN = 200

export async function loadNotes() {
  const items = await listItems(['inbox', 'kept'])
  return items
    .filter((i) => i.type === 'note')
    .map((i) => ({
      id: i.id,
      body: i.body || '',
      version: i.version,
      capturedAt: i.capturedAt,
      preview: previewText(i.body || ''),
    }))
    .sort((a, b) => new Date(b.capturedAt) - new Date(a.capturedAt))
}

export async function getStagedCapture() {
  const staged = await readStaged()
  if (!staged) return null
  return staged
}

export async function appendToNote(noteId, staged, onBeforeSave) {
  // Re-read the note to get the latest body and version.
  const note = await getItem(noteId)
  if (!note || note.type !== 'note') {
    throw new Error('the note is no longer available')
  }

  const selectionBody = staged.markdown?.trim() || ''
  if (!selectionBody && staged.images.length === 0) {
    throw new Error('the selection is empty')
  }

  const appended = note.body.trimEnd() + '\n\n' + selectionBody

  // Download images and resolve tokens.
  const downloads = await downloadSelectionImages(staged.images)
  const { markdown, blobs } = resolveImageTokens(appended, downloads)

  if (onBeforeSave) onBeforeSave()

  await editNote(noteId, markdown, note.version, blobs)
  await clearStaged()
}

export async function cancelAppend() {
  await clearStaged()
}

function previewText(markdown) {
  // Strip markdown syntax for a readable preview: images, links, headings, formatting.
  let text = markdown
    .replace(/!\[[^\]]*\]\([^)]*\)/g, '')  // images
    .replace(/\[([^\]]*)\]\([^)]*\)/g, '$1') // links keep text
    .replace(/^#{1,6}\s+/gm, '')             // heading markers
    .replace(/(\*{1,3}|_{1,3}|`{1,3})/g, '')  // bold, italic, code
    .replace(/>\s?/g, '')                     // blockquote
    .replace(/\n{2,}/g, ' ')                  // blank lines to space
    .replace(/\n/g, ' ')                      // single newlines to space
    .trim()
  if (text.length > MAX_PREVIEW_LEN) {
    text = text.slice(0, MAX_PREVIEW_LEN) + '\u2026'
  }
  return text || '(empty note)'
}