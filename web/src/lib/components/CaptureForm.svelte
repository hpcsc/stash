<script lang="ts">
  interface Props {
    saving: boolean
    onsave: (body: string) => void
    onattach: (file: File) => void
  }

  let { saving, onsave, onattach }: Props = $props()

  let body = $state('')
  let textareaEl: HTMLTextAreaElement | undefined = $state()
  let imageEl: HTMLInputElement | undefined = $state()
  let fileEl: HTMLInputElement | undefined = $state()

  function handleInput() {
    if (!textareaEl) return
    textareaEl.style.height = 'auto'
    textareaEl.style.height = `${textareaEl.scrollHeight}px`
  }

  function handleKeydown(e: KeyboardEvent) {
    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
      e.preventDefault()
      submit()
    }
  }

  function submit() {
    const text = body.trim()
    if (!text || saving) return
    onsave(text)
    body = ''
    if (textareaEl) {
      textareaEl.style.height = 'auto'
    }
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
    if (!file) return
    onattach(file)
    input.value = ''
  }
</script>

<form class="capture" onsubmit={(e) => { e.preventDefault(); submit() }}>
  <textarea
    bind:this={textareaEl}
    bind:value={body}
    oninput={handleInput}
    onkeydown={handleKeydown}
    placeholder="Paste or type anything to stash…"
    rows="2"
    disabled={saving}
  ></textarea>
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
      <span class="hint">{body.length} character{body.length !== 1 ? 's' : ''}</span>
    </div>
    <button type="submit" class="save-btn" disabled={saving || !body.trim()}>
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

  .capture textarea {
    width: 100%;
    min-height: 72px;
    padding: var(--space-4);
    box-sizing: border-box;
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