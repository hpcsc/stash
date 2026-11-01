<script lang="ts">
  import { onMount } from 'svelte'
  import { createNoteEditor, type NoteEditor, type NoteUpload } from '../editor'

  interface Props {
    saving: boolean
    onsave: (upload: NoteUpload) => void
    onattach: (file: File) => void
  }

  let { saving, onsave, onattach }: Props = $props()

  let container: HTMLDivElement | undefined = $state()
  let imageEl: HTMLInputElement | undefined = $state()
  let fileEl: HTMLInputElement | undefined = $state()
  let editor: NoteEditor | undefined = $state()
  let hasContent = $state(false)
  let charCount = $state(0)
  let imageCount = $state(0)

  onMount(() => {
    let disposed = false
    createNoteEditor(container!, (markdown) => {
      hasContent = markdown.trim().length > 0
      charCount = markdown.length
      imageCount = (markdown.match(/!\[[^\]]*\]\([^)]*\)/g) ?? []).length
    }).then((created) => {
      if (disposed) {
        created.destroy()
        return
      }
      editor = created
    })

    return () => {
      disposed = true
      editor?.destroy()
    }
  })

  function submit() {
    if (!editor || saving) return
    const upload = editor.upload()
    if (upload.markdown.trim() === '' && upload.images.length === 0) return
    onsave(upload)
    editor.clear()
  }

  function pickImage() {
    imageEl?.click()
  }

  function pickFile() {
    fileEl?.click()
  }

  function handlePicked(e: Event) {
    const input = e.target as HTMLInputElement
    const file = input.files?.[0]
    input.value = ''
    if (!file) return
    onattach(file)
  }

  function handlePickedImage(e: Event) {
    const input = e.target as HTMLInputElement
    const file = input.files?.[0]
    input.value = ''
    if (!file || !editor) return
    editor.insertImage(file)
  }
</script>

<form class="capture" onsubmit={(e) => { e.preventDefault(); submit() }}>
  <div bind:this={container} class="editor"></div>
  <div class="capture-footer">
    <div class="footer-left">
      <button type="button" class="attach-btn" onclick={pickImage} disabled={saving} title="Attach an image">
        <svg width="18" height="18" viewBox="0 0 16 16" fill="none">
          <path d="M1.5 3A1.5 1.5 0 0 1 3 1.5h10A1.5 1.5 0 0 1 14.5 3v10a1.5 1.5 0 0 1-1.5 1.5H3A1.5 1.5 0 0 1 1.5 13V3z" stroke="currentColor" stroke-width="1.3" stroke-linejoin="round" />
          <path d="M5 6a1 1 0 1 0 0-2 1 1 0 0 0 0 2zM14.5 11l-4-4L3 14.5" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </button>
      <input bind:this={imageEl} type="file" accept="image/*" onchange={handlePickedImage} hidden />
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
    transition: border-color var(--duration-fast) var(--ease-out),
                box-shadow var(--duration-fast) var(--ease-out);
  }

  .editor:focus-within {
    border-color: var(--color-accent);
    box-shadow: 0 0 0 3px var(--color-accent-subtle);
  }

  .editor :global(.milkdown) {
    min-height: 48px;
  }

  .editor :global(.ProseMirror) {
    outline: none;
    min-height: 48px;
    overflow-wrap: anywhere;
  }

  .editor :global(.ProseMirror p) {
    margin: 0 0 0.5em;
  }

  .editor :global(.ProseMirror h1),
  .editor :global(.ProseMirror h2),
  .editor :global(.ProseMirror h3) {
    margin: 0.6em 0 0.3em;
    line-height: 1.3;
  }

  .editor :global(.ProseMirror h1) { font-size: 1.4em; }
  .editor :global(.ProseMirror h2) { font-size: 1.25em; }
  .editor :global(.ProseMirror h3) { font-size: 1.1em; }

  .editor :global(.ProseMirror ul),
  .editor :global(.ProseMirror ol) {
    margin: 0 0 0.5em;
    padding-left: 1.5em;
  }

  .editor :global(.ProseMirror blockquote) {
    margin: 0 0 0.5em;
    padding-left: 0.8em;
    border-left: 3px solid var(--color-border);
    color: var(--color-text-secondary);
  }

  .editor :global(.ProseMirror code) {
    font-family: ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace;
    font-size: 0.9em;
    background: var(--color-bg);
    padding: 0.1em 0.3em;
    border-radius: var(--radius-sm);
  }

  .editor :global(.ProseMirror pre) {
    margin: 0 0 0.5em;
    padding: var(--space-3);
    background: var(--color-bg);
    border-radius: var(--radius-md);
    overflow-x: auto;
  }

  .editor :global(.ProseMirror pre code) {
    padding: 0;
    background: none;
  }

  .editor :global(.ProseMirror img) {
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
