<script lang="ts">
  import { onMount } from 'svelte'
  import { listItems, restoreItem, deleteItem, emptyTrash as emptyTrashRequest, type Item } from '../api/client'
  import ItemCard from './ItemCard.svelte'
  import ItemDetail from './ItemDetail.svelte'
  import EmptyState from './EmptyState.svelte'
  import ErrorBanner from './ErrorBanner.svelte'
  import SkeletonCard from './SkeletonCard.svelte'
  import SearchBar from './SearchBar.svelte'

  let items = $state<Item[]>([])
  let loading = $state(true)
  let error = $state<string | null>(null)
  let query = $state('')
  let selected = $state<Item | null>(null)
  let emptying = $state(false)

  onMount(() => {
    load()
  })

  async function load(search = '') {
    loading = true
    error = null
    try {
      items = await listItems(search, ['dropped'])
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load trash'
    } finally {
      loading = false
    }
  }

  function handleSearch(q: string) {
    query = q
    load(q)
  }

  async function handleRestore(id: string) {
    error = null
    try {
      await restoreItem(id)
      items = items.filter((item) => item.id !== id)
      if (selected?.id === id) selected = null
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to restore the item'
    }
  }

  async function handleDelete(id: string) {
    if (!confirm('Delete this item forever?')) return
    error = null
    try {
      await deleteItem(id)
      items = items.filter((item) => item.id !== id)
      if (selected?.id === id) selected = null
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to delete the item'
    }
  }

  async function emptyTrash() {
    if (!confirm('Empty the trash? This cannot be undone.')) return

    emptying = true
    error = null
    try {
      await emptyTrashRequest()
      items = []
      selected = null
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to empty the trash'
    } finally {
      emptying = false
    }
  }
</script>

{#if selected}
  {@const current = selected}
  <ItemDetail
    item={current}
    onback={() => (selected = null)}
    onrestore={() => handleRestore(current.id)}
    ondelete={() => handleDelete(current.id)}
  />
{:else}
  <SearchBar placeholder="Search trash…" onsearch={handleSearch} />
  {#if error}
    <ErrorBanner message={error} onretry={() => load(query)} />
  {/if}

  <div class="section-header">
    <div class="heading">
      <h2>Trash</h2>
      <span class="count">{items.length} item{items.length !== 1 ? 's' : ''}</span>
    </div>
    {#if items.length > 0 && !loading}
      <button type="button" class="empty-btn" onclick={emptyTrash} disabled={emptying}>
        {emptying ? 'Emptying…' : 'Empty trash'}
      </button>
    {/if}
  </div>

  {#if loading}
    <div class="list">
      {#each Array(3) as _}
        <SkeletonCard />
      {/each}
    </div>
  {:else if items.length === 0}
    <EmptyState
      title="Trash is empty"
      description="Dropped items will appear here. You can restore them or empty the trash."
    />
  {:else}
    <div class="list">
      {#each items as item (item.id)}
        <ItemCard
          {item}
          onclick={() => (selected = item)}
          onrestore={handleRestore}
          ondelete={handleDelete}
        />
      {/each}
    </div>
  {/if}
{/if}

<style>
  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    margin-bottom: var(--space-4);
  }

  .heading {
    display: flex;
    align-items: baseline;
    gap: var(--space-3);
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

  .empty-btn {
    padding: var(--space-2) var(--space-3);
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    color: var(--color-danger);
    background: transparent;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    transition: background var(--duration-fast) var(--ease-out),
                border-color var(--duration-fast) var(--ease-out);
  }

  .empty-btn:hover:not(:disabled) {
    background: var(--color-danger-subtle);
    border-color: var(--color-danger);
  }

  .empty-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .list {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }
</style>