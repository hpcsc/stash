<script lang="ts">
  import { onMount } from 'svelte'
  import { listItems, listTags, dropItem, type Item, type Tag } from '../api/client'
  import ErrorBanner from './ErrorBanner.svelte'
  import EmptyState from './EmptyState.svelte'
  import ItemCard from './ItemCard.svelte'
  import ItemDetail from './ItemDetail.svelte'
  import LoadMore from './LoadMore.svelte'
  import SearchBar from './SearchBar.svelte'
  import SkeletonCard from './SkeletonCard.svelte'

  let items = $state<Item[]>([])
  let tags = $state<Tag[]>([])
  let total = $state(0)
  let loading = $state(true)
  let loadingMore = $state(false)
  let error = $state<string | null>(null)
  let query = $state('')
  let activeTags = $state<string[]>([])
  let selected = $state<Item | null>(null)

  onMount(() => {
    load()
    loadTags()

    function onTabChange(e: Event) {
      if ((e as CustomEvent).detail === 'library') {
        load(query)
        loadTags()
      }
    }
    window.addEventListener('tab-change', onTabChange)
    return () => window.removeEventListener('tab-change', onTabChange)
  })

  async function load(search = '', append = false) {
    if (append) {
      loadingMore = true
    } else {
      loading = true
      items = []
    }
    error = null
    try {
      const page = await listItems({
        search,
        states: ['kept'],
        tags: activeTags.length > 0 ? activeTags : undefined,
        offset: append ? items.length : 0,
      })
      items = append ? [...items, ...page.items] : page.items
      total = page.total
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load the library'
    } finally {
      loading = false
      loadingMore = false
    }
  }

  async function loadTags() {
    try {
      tags = await listTags()
    } catch {
      // tags are optional
    }
  }

  function handleSearch(q: string) {
    query = q
    load(q)
  }

  function toggleTag(name: string) {
    activeTags = activeTags.includes(name)
      ? activeTags.filter((t) => t !== name)
      : [...activeTags, name]
    load(query)
  }

  function selectAllItems() {
    activeTags = []
    load(query)
  }

  async function handleDrop(id: string) {
    error = null
    try {
      await dropItem(id)
      items = items.filter((item) => item.id !== id)
      total = Math.max(0, total - 1)
      if (selected?.id === id) selected = null
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to drop item'
    }
  }
</script>

<div class="library-layout">
  <aside class="tag-sidebar">
    <button
      type="button"
      class="tag-nav-item"
      class:active={activeTags.length === 0}
      onclick={selectAllItems}
    >
      All items
    </button>

    <div class="tag-list">
      {#each tags as tag (tag.id)}
        <button
          type="button"
          class="tag-nav-item"
          class:active={activeTags.includes(tag.name)}
          onclick={() => toggleTag(tag.name)}
        >
          {tag.name}
        </button>
      {/each}
    </div>
  </aside>

  <section class="content">
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
        <h2>{activeTags.length > 0 ? activeTags.join(' + ') : 'Library'}</h2>
        <span class="count">{total} item{total !== 1 ? 's' : ''}</span>
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
          description={activeTags.length > 0
            ? 'No items have all of these tags.'
            : 'Items you keep from the Inbox will appear here, organized by tag.'}
        />
      {:else}
        <div class="list">
          {#each items as item (item.id)}
            <ItemCard {item} onclick={() => (selected = item)} ondrop={handleDrop} />
          {/each}
        </div>
        <LoadMore
          remaining={total - items.length}
          loading={loadingMore}
          onclick={() => load(query, true)}
        />
      {/if}
    {/if}
  </section>
</div>

<style>
  .library-layout {
    display: flex;
    gap: var(--space-8);
    align-items: flex-start;
  }

  .tag-sidebar {
    position: sticky;
    top: calc(56px + var(--space-10));
    width: 180px;
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .tag-nav-item {
    width: 100%;
    text-align: left;
    padding: var(--space-2) var(--space-3);
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    color: var(--color-text-secondary);
    background: transparent;
    border-radius: var(--radius-sm);
    transition: all var(--duration-fast) var(--ease-out);
  }

  .tag-nav-item:first-child {
    font-weight: var(--weight-semibold);
    color: var(--color-text);
    margin-bottom: var(--space-1);
  }

  .tag-nav-item:hover {
    color: var(--color-text);
    background: var(--color-surface-hover);
  }

  .tag-nav-item.active {
    color: var(--color-accent);
    background: var(--color-accent-subtle);
  }

  .tag-list {
    display: flex;
    flex-direction: column;
    gap: 1px;
    max-height: calc(100vh - 200px);
    overflow-y: auto;
  }

  .content {
    flex: 1;
    min-width: 0;
  }

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
