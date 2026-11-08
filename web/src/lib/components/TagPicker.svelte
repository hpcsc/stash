<script lang="ts">
  import { onMount } from 'svelte'
  import { listTags, createTag, type Tag } from '../api/client'

  interface Props {
    onconfirm: (tagIDs: string[]) => void
    oncancel: () => void
  }

  let { onconfirm, oncancel }: Props = $props()

  let tags = $state<Tag[]>([])
  let selected = $state<Set<string>>(new Set())
  let query = $state('')
  let creating = $state(false)
  let confirming = $state(false)
  let error = $state('')
  let filterInput = $state<HTMLInputElement>()

  const trimmed = $derived(query.trim())
  const normalized = $derived(trimmed.toLowerCase())
  const visibleTags = $derived(
    normalized === '' ? tags : tags.filter((tag) => tag.name.toLowerCase().includes(normalized)),
  )
  const exactMatch = $derived(tags.find((tag) => tag.name.toLowerCase() === normalized))
  const canCreate = $derived(normalized !== '' && !exactMatch)

  onMount(() => {
    loadTags()
    filterInput?.focus()
  })

  async function loadTags() {
    try {
      tags = await listTags()
    } catch {
      error = 'Failed to load tags'
    }
  }

  function toggle(id: string) {
    const next = new Set(selected)
    if (next.has(id)) {
      next.delete(id)
    } else {
      next.add(id)
    }
    selected = next
  }

  function isSelected(id: string): boolean {
    return selected.has(id)
  }

  async function handleCreate() {
    const name = trimmed
    if (!name || creating) return

    if (exactMatch) {
      toggle(exactMatch.id)
      query = ''
      return
    }

    error = ''
    creating = true
    try {
      const id = await createTag(name)
      tags = [...tags, { id, name }]
      selected = new Set(selected).add(id)
      query = ''
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to create tag'
    } finally {
      creating = false
    }
  }

  function confirm() {
    if (confirming) return
    confirming = true
    onconfirm([...selected])
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      oncancel()
      return
    }
    if (e.key === 'Enter' && e.target === filterInput) {
      e.preventDefault()
      if (exactMatch) {
        toggle(exactMatch.id)
        query = ''
      } else if (canCreate) {
        handleCreate()
      }
    }
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="backdrop" onclick={oncancel} onkeydown={handleKeydown}>
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="picker" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true" aria-label="Pick tags">
    <h3>Pick tags</h3>

    {#if error}
      <p class="error">{error}</p>
    {/if}

    <div class="filter">
      <input
        bind:this={filterInput}
        bind:value={query}
        type="text"
        placeholder="Find or create a tag…"
        disabled={creating}
      />
      {#if canCreate}
        <button type="button" class="btn-create" onclick={handleCreate} disabled={creating}>
          {creating ? '…' : 'Create'}
        </button>
      {/if}
    </div>

    <div class="chips">
      {#if tags.length === 0}
        <p class="hint">No tags yet. Type a name above to create one.</p>
      {:else if visibleTags.length === 0}
        <p class="hint">No tags match “{trimmed}”. Press Enter to create it.</p>
      {:else}
        {#each visibleTags as tag (tag.id)}
          <button
            type="button"
            class="chip"
            class:active={isSelected(tag.id)}
            onclick={() => toggle(tag.id)}
          >
            {tag.name}
          </button>
        {/each}
      {/if}
    </div>

    <div class="actions">
      <button type="button" class="btn btn-cancel" onclick={oncancel}>Cancel</button>
      <button type="button" class="btn btn-keep" onclick={confirm} disabled={confirming}>
        Keep{selected.size > 0 ? ` with ${selected.size} tag${selected.size > 1 ? 's' : ''}` : ''}
      </button>
    </div>
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 100;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.3);
    backdrop-filter: blur(4px);
  }

  .picker {
    display: flex;
    flex-direction: column;
    width: min(400px, 90vw);
    max-height: 80vh;
    padding: var(--space-6);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-xl);
    box-shadow: var(--shadow-lg);
    animation: fade-in var(--duration-normal) var(--ease-out);
  }

  @keyframes fade-in {
    from { opacity: 0; transform: scale(0.97) translateY(4px); }
    to { opacity: 1; transform: scale(1) translateY(0); }
  }

  h3 {
    margin: 0 0 var(--space-4);
    font-size: var(--text-lg);
  }

  .error {
    margin: 0 0 var(--space-3);
    padding: var(--space-2) var(--space-3);
    font-size: var(--text-sm);
    color: var(--color-danger);
    background: var(--color-danger-subtle);
    border-radius: var(--radius-sm);
  }

  .hint {
    margin: 0;
    font-size: var(--text-sm);
    color: var(--color-text-tertiary);
  }

  .filter {
    display: flex;
    gap: var(--space-2);
    margin-bottom: var(--space-4);
  }

  .filter input {
    flex: 1;
    min-width: 0;
  }

  .chips {
    display: flex;
    flex-wrap: wrap;
    align-content: flex-start;
    gap: var(--space-2);
    min-height: 44px;
    max-height: 240px;
    overflow-y: auto;
    padding: var(--space-2);
    margin-bottom: var(--space-5);
    background: var(--color-bg);
    border: 1px solid var(--color-border-light);
    border-radius: var(--radius-md);
  }

  .chip {
    padding: var(--space-1) var(--space-3);
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    color: var(--color-text-secondary);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-xl);
    transition: all var(--duration-fast) var(--ease-out);
  }

  .chip:hover {
    border-color: var(--color-accent);
    background: var(--color-accent-subtle);
  }

  .chip.active {
    color: white;
    background: var(--color-accent);
    border-color: var(--color-accent);
  }

  .btn-create {
    padding: var(--space-2) var(--space-4);
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    color: var(--color-accent);
    background: var(--color-accent-subtle);
    border: 1px solid var(--color-accent);
    border-radius: var(--radius-md);
    white-space: nowrap;
  }

  .btn-create:hover:not(:disabled) {
    color: white;
    background: var(--color-accent);
  }

  .btn-create:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .actions {
    display: flex;
    gap: var(--space-3);
    justify-content: flex-end;
  }

  .btn {
    padding: var(--space-2) var(--space-5);
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    border-radius: var(--radius-md);
  }

  .btn-cancel {
    color: var(--color-text-secondary);
    background: var(--color-bg);
    border: 1px solid var(--color-border);
  }

  .btn-cancel:hover {
    color: var(--color-text);
    border-color: var(--color-text);
  }

  .btn-keep {
    color: white;
    background: var(--color-success);
    border: 1px solid transparent;
  }

  .btn-keep:hover {
    background: var(--color-success);
    filter: brightness(1.1);
  }
</style>
