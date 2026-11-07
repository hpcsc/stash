<script lang="ts">
  import type { Item } from '../api/client'
  import { editNote, fileUrl } from '../api/client'
  import { formatDate } from '../time'
  import { formatFileSize } from '../format'
  import { renderMarkdown } from '../markdown'
  import { createNoteEditor, type NoteEditor } from '../editor'

  interface Props {
    item: Item
    onback: () => void
    onkeep?: () => void
    ondrop?: () => void
    onrestore?: () => void
    ondelete?: () => void
  }

  let { item, onback, onkeep, ondrop, onrestore, ondelete }: Props = $props()

  const typeMeta: Record<string, string> = {
    link: 'Link',
    image: 'Image',
    file: 'File',
    note: 'Note',
  }

  const fullImage = $derived(item.filePath ? fileUrl(item.filePath) : null)
  const editableNote = $derived(item.type === 'note' && (item.state === 'inbox' || item.state === 'kept'))

  let editing = $state(false)
  let saving = $state(false)
  let saveError = $state('')
  let editorEl = $state<HTMLElement>()
  let editor: NoteEditor | null = $state(null)

  $effect(() => {
    if (!editing || !editorEl) return
    createNoteEditor(editorEl, () => {}, {
      initialMarkdown: item.body,
      fileUrl,
    }).then((ed) => {
      editor = ed
    })
    return () => {
      editor?.destroy()
      editor = null
    }
  })

  async function handleSave() {
    if (!editor) return
    saveError = ''
    saving = true
    try {
      const { markdown, images } = editor.upload()
      const updated = await editNote(item.id, markdown, item.version, images)
      item = updated
      editor.destroy()
      editor = null
      editing = false
    } catch (e) {
      if (e instanceof Error && e.message.includes('Conflict') || e instanceof Error && String((e as any).status) === '409') {
        saveError = 'The note changed after you opened it. Your edits are still here — reload and try again.'
        return
      }
      saveError = e instanceof Error ? e.message : 'Failed to save'
    } finally {
      saving = false
    }
  }

  function handleCancel() {
    editor?.destroy()
    editor = null
    editing = false
    saveError = ''
  }
</script>

<div class="detail">
  <div class="detail-top">
    <button type="button" class="back-btn" onclick={onback}>
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
        <path d="M10 3L5 8l5 5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
      </svg>
      Back
    </button>
    <span class="type-label">{typeMeta[item.type] ?? 'Text'}</span>
  </div>

  {#if editing}
    <div bind:this={editorEl} class="edit-editor"></div>
    {#if saveError}
      <p class="edit-error" role="alert">{saveError}</p>
    {/if}
    <div class="edit-actions">
      <button type="button" class="btn btn-edit-save" onclick={handleSave} disabled={saving}>
        {saving ? 'Saving\u2026' : 'Save'}
      </button>
      <button type="button" class="btn btn-edit-cancel" onclick={handleCancel} disabled={saving}>Cancel</button>
    </div>
  {:else}
    {#if item.type === 'note'}
      <div class="markdown">{@html renderMarkdown(item.body)}</div>
    {:else if item.type === 'image' && fullImage}
      <img class="full-image" src={fullImage} alt="" />
    {:else if item.type === 'file'}
      <div class="file-detail">
        <span class="file-icon">📎</span>
        <div class="file-info">
          <p class="file-name">{item.fileName || 'File'}</p>
          {#if item.fileSize}
            <p class="file-size">{formatFileSize(item.fileSize)}</p>
          {/if}
        </div>
        {#if item.filePath}
          <a class="download-btn" href={fileUrl(item.filePath)} download={item.fileName}>Download</a>
        {/if}
      </div>
    {:else}
      <p class="body">{item.title || item.body}</p>
    {/if}

    {#if item.type === 'link'}
      <a class="link-url" href={item.body} target="_blank" rel="noopener noreferrer">
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
          <path d="M12 8.5V13a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V5a1 1 0 0 1 1-1h4.5M10 2h4v4M7 9l7-7" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
        {item.body}
      </a>
    {/if}

    {#if item.description}
      <p class="description">{item.description}</p>
    {/if}

    <div class="meta">
      <span class="meta-item">{formatDate(item.capturedAt)}</span>
    </div>

    <div class="actions">
      {#if item.state === 'dropped'}
        <button type="button" class="btn btn-keep" onclick={onrestore}>
          <svg width="20" height="20" viewBox="0 0 16 16" fill="none">
            <path d="M3 8a5 5 0 1 0 1.5-3.5M3 3v3h3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          Restore
        </button>
        <button type="button" class="btn btn-drop" onclick={ondelete}>
          <svg width="20" height="20" viewBox="0 0 16 16" fill="none">
            <path d="M3 4h10M5.5 4v8a1 1 0 0 0 1 1h3a1 1 0 0 0 1-1V4M6.5 4V3a1 1 0 0 1 1-1h1a1 1 0 0 1 1 1v1" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          Delete forever
        </button>
      {:else}
        {#if editableNote}
          <button type="button" class="btn btn-edit" onclick={() => (editing = true)}>
            <svg width="20" height="20" viewBox="0 0 16 16" fill="none">
              <path d="M11 2l3 3-9 9H2v-3l9-9zM2 14h2.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
            </svg>
            Edit
          </button>
        {/if}
        {#if item.state !== 'kept'}
          <button type="button" class="btn btn-keep" onclick={onkeep}>
            <svg width="20" height="20" viewBox="0 0 16 16" fill="none">
              <path d="M3 8.5l3 3 7-7" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
            </svg>
            Keep
          </button>
        {/if}
        <button type="button" class="btn btn-drop" onclick={ondrop}>
          <svg width="20" height="20" viewBox="0 0 16 16" fill="none">
            <path d="M3 4h10M5.5 4v8a1 1 0 0 0 1 1h3a1 1 0 0 0 1-1V4M6.5 4V3a1 1 0 0 1 1-1h1a1 1 0 0 1 1 1v1" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          Drop
        </button>
      {/if}
    </div>
  {/if}
</div>

<style>
  .detail {
    animation: slide-in var(--duration-normal) var(--ease-out);
  }

  @keyframes slide-in {
    from {
      opacity: 0;
      transform: translateX(8px);
    }
    to {
      opacity: 1;
      transform: translateX(0);
    }
  }

  .detail-top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--space-4);
  }

  .back-btn {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-3);
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    color: var(--color-text-secondary);
    background: transparent;
    border-radius: var(--radius-sm);
  }

  .back-btn:hover {
    color: var(--color-text);
    background: var(--color-surface-hover);
  }

  .type-label {
    font-size: var(--text-xs);
    font-weight: var(--weight-medium);
    color: var(--color-accent);
    background: var(--color-accent-subtle);
    padding: var(--space-1) var(--space-3);
    border-radius: var(--radius-sm);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .body {
    margin: 0 0 var(--space-6);
    max-width: 68ch;
    font-size: var(--text-lg);
    line-height: 1.7;
    color: var(--color-text);
    white-space: pre-wrap;
    word-break: break-word;
  }

  .markdown {
    margin-bottom: var(--space-6);
    max-width: 68ch;
    font-size: var(--text-base);
    line-height: 1.7;
    color: var(--color-text);
    overflow-wrap: anywhere;
  }

  .markdown :global(h1),
  .markdown :global(h2),
  .markdown :global(h3) {
    margin: 1.2em 0 0.5em;
    line-height: 1.3;
  }

  .markdown :global(h1) { font-size: 1.5em; }
  .markdown :global(h2) { font-size: 1.3em; }
  .markdown :global(h3) { font-size: 1.15em; }

  .markdown :global(p) { margin: 0 0 0.8em; }

  .markdown :global(ul),
  .markdown :global(ol) {
    margin: 0 0 0.8em;
    padding-left: 1.5em;
  }

  .markdown :global(li) { margin: 0.2em 0; }

  .markdown :global(blockquote) {
    margin: 0 0 0.8em;
    padding-left: 0.9em;
    border-left: 3px solid var(--color-border);
    color: var(--color-text-secondary);
  }

  .markdown :global(code) {
    font-family: ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace;
    font-size: 0.9em;
    background: var(--color-bg);
    padding: 0.1em 0.3em;
    border-radius: var(--radius-sm);
  }

  .markdown :global(pre) {
    margin: 0 0 0.8em;
    padding: var(--space-3);
    background: var(--color-bg);
    border-radius: var(--radius-md);
    overflow-x: auto;
  }

  .markdown :global(pre code) {
    padding: 0;
    background: none;
  }

  .markdown :global(a) {
    color: var(--color-accent);
    text-decoration: none;
  }

  .markdown :global(a:hover) {
    text-decoration: underline;
  }

  .markdown :global(img) {
    max-width: 100%;
    max-height: 60vh;
    object-fit: contain;
    border-radius: var(--radius-md);
    margin: 0.5em 0;
  }

  .full-image {
    width: 100%;
    max-height: 60vh;
    object-fit: contain;
    border-radius: var(--radius-md);
    margin-bottom: var(--space-6);
  }

  .file-detail {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    margin-bottom: var(--space-6);
    padding: var(--space-4);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
  }

  .file-icon {
    font-size: var(--text-2xl);
  }

  .file-info {
    flex: 1;
    min-width: 0;
  }

  .file-name {
    margin: 0;
    font-size: var(--text-base);
    font-weight: var(--weight-medium);
    color: var(--color-text);
    word-break: break-all;
  }

  .file-size {
    margin: var(--space-1) 0 0;
    font-size: var(--text-sm);
    color: var(--color-text-tertiary);
  }

  .download-btn {
    padding: var(--space-2) var(--space-4);
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    color: white;
    background: var(--color-accent);
    border-radius: var(--radius-md);
    text-decoration: none;
    white-space: nowrap;
  }

  .download-btn:hover {
    background: var(--color-accent-hover);
  }

  .link-url {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    margin-bottom: var(--space-6);
    font-size: var(--text-sm);
    color: var(--color-accent);
    text-decoration: none;
    word-break: break-all;
  }

  .link-url:hover {
    text-decoration: underline;
  }

  .description {
    margin: 0 0 var(--space-6);
    max-width: 68ch;
    font-size: var(--text-base);
    line-height: 1.7;
    color: var(--color-text-secondary);
  }

  .meta {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-1);
    margin-bottom: var(--space-8);
    font-size: var(--text-sm);
    color: var(--color-text-tertiary);
  }

  .meta-item {
    color: var(--color-text-secondary);
  }

  .actions {
    display: flex;
    gap: var(--space-3);
  }

  .edit-editor {
    margin-bottom: var(--space-4);
    min-height: 200px;
    max-width: 68ch;
  }

  .edit-error {
    margin: 0 0 var(--space-3);
    padding: var(--space-2) var(--space-3);
    font-size: var(--text-sm);
    color: var(--color-danger);
    background: var(--color-danger-subtle);
    border-radius: var(--radius-sm);
  }

  .edit-actions {
    display: flex;
    gap: var(--space-3);
    margin-top: var(--space-4);
  }

  .btn {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-4) var(--space-6);
    font-size: var(--text-base);
    font-weight: var(--weight-semibold);
    border-radius: var(--radius-md);
    transition: all var(--duration-fast) var(--ease-out);
  }

  .btn-keep {
    color: var(--color-success);
    background: var(--color-success-subtle);
    border: 1px solid transparent;
  }

  .btn-keep:hover {
    background: var(--color-success);
    color: white;
  }

  .btn-edit {
    color: var(--color-accent);
    background: var(--color-accent-subtle);
    border: 1px solid transparent;
  }

  .btn-edit:hover {
    background: var(--color-accent);
    color: white;
  }

  .btn-drop {
    color: var(--color-text-secondary);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
  }

  .btn-drop:hover {
    color: var(--color-danger);
    border-color: var(--color-danger);
    background: var(--color-danger-subtle);
  }

  .btn-edit-save {
    color: var(--color-success);
    background: var(--color-success-subtle);
    border: 1px solid transparent;
  }

  .btn-edit-save:hover:not(:disabled) {
    background: var(--color-success);
    color: white;
  }

  .btn-edit-save:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-edit-cancel {
    color: var(--color-text-secondary);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
  }

  .btn-edit-cancel:hover:not(:disabled) {
    color: var(--color-text);
    border-color: var(--color-text);
  }

  .btn-edit-cancel:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>