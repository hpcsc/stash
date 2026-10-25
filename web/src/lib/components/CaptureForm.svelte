<script lang="ts">
  interface Props {
    saving: boolean
    onsave: (body: string) => void
  }

  let { saving, onsave }: Props = $props()

  let body = $state('')
  let textareaEl: HTMLTextAreaElement | undefined = $state()

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
    <span class="hint">{body.length} character{body.length !== 1 ? 's' : ''}</span>
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