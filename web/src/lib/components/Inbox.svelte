<script lang="ts">
  import { onMount } from 'svelte'
  import { listItems, captureLink, captureAttachment, captureNote, keepItem, dropItem, type Item } from '../api/client'
  import { connect, onMessage } from '../api/ws'
  import { token } from '../auth/session'
  import type { NoteUpload } from '../editor'
  import CaptureForm from './CaptureForm.svelte'
  import ItemCard from './ItemCard.svelte'
  import ItemDetail from './ItemDetail.svelte'
  import EmptyState from './EmptyState.svelte'
  import ErrorBanner from './ErrorBanner.svelte'
  import SkeletonCard from './SkeletonCard.svelte'
  import SearchBar from './SearchBar.svelte'

  const urlRe = /^https?:\/\/\S+$/i

  const locationLabels: Record<string, string> = { inbox: 'Inbox', kept: 'Library', dropped: 'Trash' }

  function duplicateNotice(state?: string): string {
    return `You already saved this — it's in your ${locationLabels[state ?? ''] ?? 'Inbox'}.`
  }

  let items = $state<Item[]>([])
  let loading = $state(true)
  let saving = $state(false)
  let error = $state<string | null>(null)
  let warning = $state<string | null>(null)
  let selected = $state<Item | null>(null)
  let query = $state('')

  onMount(() => {
    load()

    const unsubToken = token.subscribe((t) => {
      if (t) connect(t)
    })

    const unsubMessages = onMessage((msg) => {
      if (msg.type === 'link-metadata-fetched') {
        const data = msg.data as { id: string; title: string; description: string }
        items = items.map((i) =>
          i.id === data.id ? { ...i, title: data.title, description: data.description } : i,
        )
      }
      if (msg.type === 'thumbnail-generated') {
        const data = msg.data as { id: string; thumbnailPath: string }
        items = items.map((i) =>
          i.id === data.id ? { ...i, thumbnailPath: data.thumbnailPath } : i,
        )
      }
    })

    return () => {
      unsubToken()
      unsubMessages()
    }
  })

  async function load(search = '') {
    loading = true
    error = null
    try {
      items = await listItems(search, ['inbox'])
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load the inbox'
    } finally {
      loading = false
    }
  }

  function handleSearch(q: string) {
    query = q
    load(q)
  }

  async function save(upload: NoteUpload) {
    saving = true
    error = null
    warning = null
    try {
      const markdown = upload.markdown.trim()
      if (!markdown && upload.images.length === 0) return

      if (upload.images.length === 0 && urlRe.test(markdown)) {
        const captured = await captureLink(markdown)
        items = [
          { id: captured.id, type: 'link', body: markdown, state: 'inbox', capturedAt: new Date().toISOString() },
          ...items,
        ]
        if (captured.duplicate) warning = duplicateNotice(captured.duplicateState)
      } else {
        const item = await captureNote(upload.markdown, upload.images)
        items = [item, ...items]
      }
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to save'
    } finally {
      saving = false
    }
  }

  async function saveAttachment(file: File) {
    saving = true
    error = null
    try {
      const { id, type, filePath, fileName } = await captureAttachment(file)
      const item: Item = {
        id,
        type,
        body: '',
        filePath,
        state: 'inbox',
        capturedAt: new Date().toISOString(),
      }
      if (type === 'file') {
        if (fileName) item.fileName = fileName
        item.fileSize = file.size
      }
      items = [item, ...items]
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to save attachment'
    } finally {
      saving = false
    }
  }

  async function handleKeep(id: string) {
    error = null
    try {
      await keepItem(id)
      items = items.filter((i) => i.id !== id)
      if (selected?.id === id) selected = null
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to keep item'
    }
  }

  async function handleDrop(id: string) {
    error = null
    try {
      await dropItem(id)
      items = items.filter((i) => i.id !== id)
      if (selected?.id === id) selected = null
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to drop item'
    }
  }
</script>

{#if selected}
  {@const current = selected}
  <ItemDetail
    item={current}
    onback={() => (selected = null)}
    onkeep={() => handleKeep(current.id)}
    ondrop={() => handleDrop(current.id)}
  />
{:else}
  <CaptureForm {saving} onsave={save} onattach={saveAttachment} />

  <SearchBar placeholder="Search inbox…" onsearch={handleSearch} />

  {#if error}
    <ErrorBanner message={error} onretry={() => load(query)} />
  {/if}

  {#if warning}
    <div class="warning-banner">
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
        <path d="M8 2.5 14.5 13H1.5L8 2.5z" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round" />
        <path d="M8 6.5v3M8 11.5v.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
      </svg>
      {warning}
    </div>
  {/if}

  <div class="section-header">
    <h2>All items</h2>
    <span class="count">{items.length} item{items.length !== 1 ? 's' : ''}</span>
  </div>

  {#if loading}
    <div class="list">
      {#each Array(3) as _}
        <SkeletonCard />
      {/each}
    </div>
  {:else if items.length === 0}
    <EmptyState
      title="Your inbox is empty"
      description="Save text, links, images, or files to triage them later."
    />
  {:else}
    <div class="list">
      {#each items as item (item.id)}
        <ItemCard {item} onclick={() => (selected = item)} onkeep={handleKeep} ondrop={handleDrop} />
      {/each}
    </div>
  {/if}
{/if}

<style>
  .section-header {
    display: flex;
    align-items: baseline;
    gap: var(--space-3);
    margin-bottom: var(--space-4);
  }

  .section-header h2 {
    margin: 0;
    font-size: var(--text-sm);
    font-weight: var(--weight-semibold);
    color: var(--color-text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .count {
    font-size: var(--text-sm);
    color: var(--color-text-tertiary);
  }

  .list {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .warning-banner {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-3) var(--space-4);
    margin-bottom: var(--space-6);
    font-size: var(--text-sm);
    color: var(--color-warning);
    background: var(--color-warning-subtle);
    border: 1px solid var(--color-warning);
    border-radius: var(--radius-md);
  }
</style>