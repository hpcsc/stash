<script lang="ts">
  import { buildNoteUpload, readSegments, type NoteUpload } from '../note'

  interface Props {
    saving: boolean
    onsave: (upload: NoteUpload) => void
    onattach: (file: File) => void
  }

  let { saving, onsave, onattach }: Props = $props()

  let editorEl: HTMLDivElement | undefined = $state()
  let imageEl: HTMLInputElement | undefined = $state()
  let fileEl: HTMLInputElement | undefined = $state()
  let hasContent = $state(false)
  let charCount = $state(0)
  let imageCount = $state(0)

  let keySeed = 0
  let lastRange: Range | null = null
  const filesByKey = new Map<string, File>()
  const objectUrls = new Map<string, string>()

  function refreshState() {
    if (!editorEl) return

    const segments = readSegments(editorEl)
    hasContent = segments.length > 0
    imageCount = segments.reduce((count, segment) => count + (segment.type === 'image' ? 1 : 0), 0)
    charCount = segments.reduce((count, segment) => count + (segment.type === 'text' ? segment.text.length : 0), 0)
  }

  function rememberCaret() {
    if (!editorEl) return

    const selection = window.getSelection()
    if (!selection || selection.rangeCount === 0) return

    const range = selection.getRangeAt(0)
    if (editorEl.contains(range.commonAncestorContainer)) {
      lastRange = range.cloneRange()
    }
  }

  function focusEnd() {
    if (!editorEl) return

    const range = document.createRange()
    range.selectNodeContents(editorEl)
    range.collapse(false)

    const selection = window.getSelection()
    selection?.removeAllRanges()
    selection?.addRange(range)
    editorEl.focus()
  }

  function insertAtCaret(node: Node) {
    if (!editorEl) return

    const selection = window.getSelection()
    const current = selection && selection.rangeCount > 0 ? selection.getRangeAt(0) : null
    const range = current && editorEl.contains(current.commonAncestorContainer) ? current : lastRange

    if (!range || !editorEl.contains(range.commonAncestorContainer)) {
      editorEl.appendChild(node)
      lastRange = null
      focusEnd()
      return
    }

    range.deleteContents()
    range.insertNode(node)
    range.setStartAfter(node)
    range.collapse(true)
    lastRange = range.cloneRange()

    selection?.removeAllRanges()
    selection?.addRange(range)
  }

  function insertTextAtCaret(text: string) {
    insertAtCaret(document.createTextNode(text))
    refreshState()
  }

  function insertImage(file: File) {
    const key = `image-${++keySeed}`
    const url = URL.createObjectURL(file)
    filesByKey.set(key, file)
    objectUrls.set(key, url)

    const image = document.createElement('img')
    image.src = url
    image.dataset.key = key
    image.alt = file.name || 'Image'

    insertAtCaret(image)
    refreshState()
  }

  function clearEditor() {
    editorEl?.replaceChildren()
    for (const url of objectUrls.values()) URL.revokeObjectURL(url)
    objectUrls.clear()
    filesByKey.clear()
    lastRange = null
    refreshState()
  }

  function submit() {
    if (!editorEl || saving) return

    const upload = buildNoteUpload(readSegments(editorEl), filesByKey)
    if (upload.document.blocks.length === 0) return

    onsave(upload)
    clearEditor()
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.isComposing) return

    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
      e.preventDefault()
      submit()
      return
    }
    if (e.key === 'Enter') {
      e.preventDefault()
      insertTextAtCaret('\n')
    }
  }

  function handlePaste(e: ClipboardEvent) {
    const clipboard = e.clipboardData
    if (!clipboard) return

    const files = Array.from(clipboard.files ?? [])
    const images = files.filter((file) => file.type.startsWith('image/'))
    if (images.length > 0) {
      e.preventDefault()
      for (const image of images) insertImage(image)
      return
    }
    if (files.length > 0) {
      e.preventDefault()
      for (const file of files) onattach(file)
      return
    }

    e.preventDefault()
    const text = clipboard.getData('text/plain')
    if (text) insertTextAtCaret(text)
  }

  function handleDragOver(e: DragEvent) {
    if (e.dataTransfer?.types.includes('Files')) e.preventDefault()
  }

  function handleDrop(e: DragEvent) {
    const transfer = e.dataTransfer
    if (!transfer) return

    const files = Array.from(transfer.files ?? [])
    if (files.length === 0) return

    e.preventDefault()
    const dropped = rangeFromPoint(e.clientX, e.clientY)
    if (dropped) selectRange(dropped)

    for (const file of files) {
      if (file.type.startsWith('image/')) insertImage(file)
      else onattach(file)
    }
  }

  function rangeFromPoint(x: number, y: number): Range | null {
    const doc = document as Document & {
      caretRangeFromPoint?: (x: number, y: number) => Range | null
    }
    const range = doc.caretRangeFromPoint?.(x, y) ?? null
    if (!range || !editorEl) return null
    return editorEl.contains(range.commonAncestorContainer) ? range : null
  }

  function selectRange(range: Range) {
    const selection = window.getSelection()
    selection?.removeAllRanges()
    selection?.addRange(range)
    editorEl?.focus()
  }

  function pickImage() {
    rememberCaret()
    imageEl?.click()
  }

  function pickFile() {
    rememberCaret()
    fileEl?.click()
  }

  function handlePicked(e: Event) {
    const input = e.target as HTMLInputElement
    const file = input.files?.[0]
    input.value = ''
    if (!file) return

    if (file.type.startsWith('image/')) {
      insertImage(file)
      return
    }
    onattach(file)
  }
</script>

<form class="capture" onsubmit={(e) => { e.preventDefault(); submit() }}>
  <div
    bind:this={editorEl}
    class="editor"
    contenteditable="true"
    role="textbox"
    aria-multiline="true"
    aria-label="Capture"
    tabindex="0"
    data-placeholder="Paste or type anything to stash…"
    data-empty={hasContent ? 'false' : 'true'}
    oninput={refreshState}
    onkeydown={handleKeydown}
    onkeyup={rememberCaret}
    onmouseup={rememberCaret}
    onblur={rememberCaret}
    onpaste={handlePaste}
    ondragover={handleDragOver}
    ondrop={handleDrop}
  ></div>
  <div class="capture-footer">
    <div class="footer-left">
      <button type="button" class="attach-btn" onclick={pickImage} disabled={saving} title="Attach an image">
        <svg width="18" height="18" viewBox="0 0 16 16" fill="none">
          <path d="M1.5 3A1.5 1.5 0 0 1 3 1.5h10A1.5 1.5 0 0 1 14.5 3v10a1.5 1.5 0 0 1-1.5 1.5H3A1.5 1.5 0 0 1 1.5 13V3z" stroke="currentColor" stroke-width="1.3" stroke-linejoin="round" />
          <path d="M5 6a1 1 0 1 0 0-2 1 1 0 0 0 0 2zM14.5 11l-4-4L3 14.5" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </button>
      <input bind:this={imageEl} type="file" accept="image/*" onchange={handlePicked} hidden />
      <button type="button" class="attach-btn" onclick={pickFile} disabled={saving} title="Attach a file">
        <svg width="18" height="18" viewBox="0 0 16 16" fill="none">
          <path d="M13.5 7.5l-5.6 5.6a3.5 3.5 0 0 1-5-5l6-6a2.3 2.3 0 0 1 3.3 3.3l-6 6a1.2 1.2 0 0 1-1.7-1.7l5.3-5.3" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </button>
      <input bind:this={fileEl} type="file" onchange={handlePicked} hidden />
      <span class="hint">
        {charCount} character{charCount !== 1 ? 's' : ''}{imageCount > 0 ? ` · ${imageCount} image${imageCount !== 1 ? 's' : ''}` : ''}
      </span>
    </div>
    <button type="submit" class="save-btn" disabled={saving || !hasContent}>
      {#if saving}
        <span class="spinner"></span>
        Saving…
      {:else}
        Save
      {/if}
    </button>
  </div>
</form>

<style>
  .capture {
    margin-bottom: var(--space-8);
  }

  .editor {
    position: relative;
    width: 100%;
    min-height: 72px;
    max-height: 50vh;
    padding: var(--space-3) var(--space-4);
    box-sizing: border-box;
    overflow-y: auto;
    font-family: inherit;
    font-size: var(--text-base);
    line-height: var(--leading);
    color: var(--color-text);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-xs);
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    cursor: text;
    transition: border-color var(--duration-fast) var(--ease-out),
                box-shadow var(--duration-fast) var(--ease-out);
  }

  .editor:focus {
    outline: none;
    border-color: var(--color-accent);
    box-shadow: 0 0 0 3px var(--color-accent-subtle);
  }

  .editor[data-empty='true']::before {
    content: attr(data-placeholder);
    position: absolute;
    top: var(--space-3);
    left: var(--space-4);
    color: var(--color-text-tertiary);
    pointer-events: none;
  }

  .editor :global(img) {
    display: inline-block;
    max-width: 100%;
    max-height: 200px;
    margin: 2px 0;
    border-radius: var(--radius-md);
    object-fit: contain;
    vertical-align: middle;
  }

  .capture-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-top: var(--space-3);
  }

  .footer-left {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .attach-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 34px;
    height: 34px;
    color: var(--color-text-tertiary);
    background: transparent;
    border-radius: var(--radius-sm);
    transition: color var(--duration-fast) var(--ease-out),
                background var(--duration-fast) var(--ease-out);
  }

  .attach-btn:hover:not(:disabled) {
    color: var(--color-accent);
    background: var(--color-accent-subtle);
  }

  .hint {
    font-size: var(--text-xs);
    color: var(--color-text-tertiary);
  }

  .save-btn {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-3) var(--space-6);
    font-size: var(--text-sm);
    font-weight: var(--weight-semibold);
    color: white;
    background: var(--color-accent);
    border-radius: var(--radius-md);
    transition: background var(--duration-fast) var(--ease-out),
                opacity var(--duration-fast) var(--ease-out);
  }

  .save-btn:hover:not(:disabled) {
    background: var(--color-accent-hover);
  }

  .save-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .spinner {
    display: inline-block;
    width: 14px;
    height: 14px;
    border: 2px solid rgba(255, 255, 255, 0.3);
    border-top-color: white;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>
