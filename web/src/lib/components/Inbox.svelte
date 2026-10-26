<script lang="ts">
  import { onMount } from 'svelte'
  import { listItems, captureText, captureLink, captureImage, keepItem, dropItem, type Item } from '../api/client'
  import { connect, onMessage } from '../api/ws'
  import { token } from '../auth/session'
  import CaptureForm from './CaptureForm.svelte'
  import ItemCard from './ItemCard.svelte'
  import ItemDetail from './ItemDetail.svelte'
  import EmptyState from './EmptyState.svelte'
  import SkeletonCard from './SkeletonCard.svelte'
  import SearchBar from './SearchBar.svelte'

  const urlRe = /^https?:\/\/\S+$/i

  let items = $state<Item[]>([])
  let loading = $state(true)
  let saving = $state(false)
  let error = $state<string | null>(null)
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

  async function save(body: string) {
    saving = true
    error = null
    try {
      if (urlRe.test(body.trim())) {
        const id = await captureLink(body.trim())
        items = [
          { id, type: 'link', body: body.trim(), state: 'inbox', capturedAt: new Date().toISOString() },
          ...items,
        ]
      } else {
        const id = await captureText(body)
        items = [
          { id, type: 'text', body, state: 'inbox', capturedAt: new Date().toISOString() },
          ...items,
        ]
      }
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to save'
    } finally {
      saving = false
    }
  }

  async function saveImage(file: File) {
    saving = true
    error = null
    try {
      const { id, filePath } = await captureImage(file)
      items = [
        { id, type: 'image', body: '', filePath, state: 'inbox', capturedAt: new Date().toISOString() },
        ...items,
      ]
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to save image'
    } finally {
      saving = false
    }
  }

  async function handleKeep(id: string) {
    error = null
    try {
      await keepItem(id)
      items = items.filter((i) => i.id !== id)
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to keep item'
    }
  }

  async function handleDrop(id: string) {
    error = null
    try {
      await dropItem(id)
      items = items.filter((i) => i.id !== id)
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to drop item'
    }
  }
</script>

{#if selected}
  <ItemDetail item={selected} onback={() => (selected = null)} />
{:else}
  <CaptureForm {saving} onsave={save} onimage={saveImage} />

  <SearchBar placeholder="Search inbox…" onsearch={handleSearch} />

  {#if error}
    <div class="error-banner">
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
        <circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.5" />
        <path d="M8 5v3M8 10.5v.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
      </svg>
      {error}
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

  .error-banner {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-3) var(--space-4);
    margin-bottom: var(--space-6);
    font-size: var(--text-sm);
    color: var(--color-danger);
    background: var(--color-danger-subtle);
    border: 1px solid var(--color-danger);
    border-radius: var(--radius-md);
  }
</style>