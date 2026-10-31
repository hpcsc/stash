<script lang="ts">
  interface Props {
    message: string
    onretry?: () => void
  }

  let { message, onretry }: Props = $props()
</script>

<div class="error-banner">
  <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
    <circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.5" />
    <path d="M8 5v3M8 10.5v.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
  </svg>
  <span class="message">{message}</span>
  {#if onretry}
    <button type="button" class="retry-btn" onclick={onretry}>Retry</button>
  {/if}
</div>

<style>
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

  .message {
    flex: 1;
  }

  .retry-btn {
    padding: var(--space-1) var(--space-3);
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    color: var(--color-danger);
    background: transparent;
    border: 1px solid var(--color-danger);
    border-radius: var(--radius-sm);
    white-space: nowrap;
  }

  .retry-btn:hover {
    background: var(--color-danger);
    color: white;
  }
</style>
