<script lang="ts">
  import { onMount } from 'svelte'
  import { listItems, type Item } from '../api/client'
  import ItemCard from './ItemCard.svelte'
  import ItemDetail from './ItemDetail.svelte'
  import EmptyState from './EmptyState.svelte'
  import SkeletonCard from './SkeletonCard.svelte'

  let items = $state<Item[]>([])
  let loading = $state(true)
  let error = $state<string | null>(null)
  let selected = $state<Item | null>(null)

  onMount(async () => {
    try {
      items = await listItems(['dropped'])
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load trash'
    } finally {
      loading = false
    }
  })
</script>

{#if selected}
  <ItemDetail item={selected} onback={() => (selected = null)} />
{:else}
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
    <h2>Trash</h2>
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
      title="Trash is empty"
      description="Dropped items will appear here. You can restore them or empty the trash."
    />
  {:else}
    <div class="list">
      {#each items as item (item.id)}
        <ItemCard {item} onclick={() => (selected = item)} />
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