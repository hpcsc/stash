<script lang="ts">
  import { onMount } from 'svelte'
  import { listItems, dropItem, type Item } from '../api/client'
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

  onMount(() => {
    load()
  })

  async function load(search = '') {
    loading = true
    error = null
    try {
      items = await listItems(search, ['kept'])
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load the library'
    } finally {
      loading = false
    }
  }

  function handleSearch(q: string) {
    query = q
    load(q)
  }

  async function handleDrop(id: string) {
    error = null
    try {
      await dropItem(id)
      items = items.filter((item) => item.id !== id)
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
    ondrop={() => handleDrop(current.id)}
  />
{:else}
  <SearchBar placeholder="Search library…" onsearch={handleSearch} />
  {#if error}
    <ErrorBanner message={error} onretry={() => load(query)} />
  {/if}

  <div class="section-header">
    <h2>Library</h2>
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
      title="Nothing here yet"
      description="Items you keep from the Inbox will appear here, organized by tag."
    />
  {:else}
    <div class="list">
      {#each items as item (item.id)}
        <ItemCard {item} onclick={() => (selected = item)} ondrop={handleDrop} />
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
</style>