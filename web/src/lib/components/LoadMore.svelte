<script lang="ts">
  let {
    remaining,
    loading = false,
    onclick,
  }: { remaining: number; loading?: boolean; onclick: () => void } = $props()
</script>

{#if remaining > 0}
  <div class="load-more">
    <button type="button" onclick={onclick} disabled={loading}>
      {loading ? 'Loading…' : 'Load more'}
    </button>
    <span>{remaining} more</span>
  </div>
{/if}

<style>
  .load-more {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-1);
    padding: var(--space-4) 0;
  }

  .load-more button {
    padding: var(--space-2) var(--space-4);
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    color: var(--color-text);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    transition: background var(--duration-fast) var(--ease-out),
                border-color var(--duration-fast) var(--ease-out);
  }

  .load-more button:hover:not(:disabled) {
    background: var(--color-surface-hover);
  }

  .load-more button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .load-more span {
    font-size: var(--text-xs);
    color: var(--color-text-tertiary);
  }
</style>
