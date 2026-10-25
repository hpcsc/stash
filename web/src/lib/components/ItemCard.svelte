<script lang="ts">
  import type { Item } from '../api/client'
  import { relativeTime } from '../time'

  interface Props {
    item: Item
    onclick?: () => void
    onkeep?: (id: string) => void
    ondrop?: (id: string) => void
  }

  let { item, onclick, onkeep, ondrop }: Props = $props()

  const typeMeta: Record<string, { icon: string; label: string }> = {
    text: { icon: 'T', label: 'Text' },
    link: { icon: '🔗', label: 'Link' },
    image: { icon: '🖼', label: 'Image' },
    file: { icon: '📎', label: 'File' },
  }

  const meta = $derived(typeMeta[item.type] ?? typeMeta.text)
  const timeAgo = $derived(relativeTime(item.capturedAt))
  const preview = $derived(item.body.slice(0, 160))
  const hasMore = $derived(item.body.length > 160)

  function handleKeep(e: MouseEvent) {
    e.stopPropagation()
    onkeep?.(item.id)
  }

  function handleDrop(e: MouseEvent) {
    e.stopPropagation()
    ondrop?.(item.id)
  }
</script>

<button type="button" class="card" {onclick}>
  <div class="card-header">
    <span class="type-badge" title={meta.label}>{meta.icon}</span>
    <span class="time">{timeAgo}</span>
  </div>
  <p class="body">{preview}{hasMore ? '…' : ''}</p>
  <div class="card-footer">
    {#if item.sourceApp}
      <span class="source">{item.sourceApp}</span>
    {/if}
    <div class="spacer"></div>
    {#if !item.state || item.state === 'inbox'}
      <div class="actions">
        <span role="button" tabindex="0" class="action-btn keep" onclick={handleKeep} onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') handleKeep(e as unknown as MouseEvent) }} title="Keep">
          <svg width="20" height="20" viewBox="0 0 16 16" fill="none">
            <path d="M3 8.5l3 3 7-7" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
        </span>
        <span role="button" tabindex="0" class="action-btn drop" onclick={handleDrop} onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') handleDrop(e as unknown as MouseEvent) }} title="Drop">
          <svg width="20" height="20" viewBox="0 0 16 16" fill="none">
            <path d="M3 4h10M5.5 4v8a1 1 0 0 0 1 1h3a1 1 0 0 0 1-1V4M6.5 4V3a1 1 0 0 1 1-1h1a1 1 0 0 1 1 1v1" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
        </span>
      </div>
    {/if}
  </div>
</button>

<style>
  .card {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    width: 100%;
    text-align: left;
    padding: var(--space-4);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-xs);
    cursor: pointer;
    transition: border-color var(--duration-fast) var(--ease-out),
                box-shadow var(--duration-fast) var(--ease-out),
                transform var(--duration-fast) var(--ease-out);
  }

  .card:hover {
    border-color: var(--color-accent);
    box-shadow: var(--shadow-md);
    transform: translateY(-2px);
  }

  .card:active {
    transform: translateY(0);
  }

  .card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .type-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 30px;
    height: 30px;
    font-size: var(--text-sm);
    color: var(--color-accent);
    background: var(--color-accent-subtle);
    border: 1px solid color-mix(in srgb, var(--color-accent) 25%, transparent);
    border-radius: var(--radius-sm);
  }

  .time {
    font-size: var(--text-xs);
    color: var(--color-text-tertiary);
  }

  .body {
    margin: 0;
    font-size: var(--text-base);
    line-height: var(--leading);
    color: var(--color-text);
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .card-footer {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .spacer {
    flex: 1;
  }

  .source {
    font-size: var(--text-xs);
    color: var(--color-text-secondary);
    background: var(--color-bg);
    padding: var(--space-1) var(--space-2);
    border-radius: var(--radius-sm);
  }

  .actions {
    display: flex;
    gap: var(--space-1);
    opacity: 0;
    transform: translateX(4px);
    transition: opacity var(--duration-fast) var(--ease-out),
                transform var(--duration-fast) var(--ease-out);
  }

  .card:hover .actions {
    opacity: 1;
    transform: translateX(0);
  }

  .action-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 38px;
    height: 38px;
    border-radius: var(--radius-sm);
    background: transparent;
    cursor: pointer;
    transition: background var(--duration-fast) var(--ease-out),
                color var(--duration-fast) var(--ease-out);
  }

  .action-btn.keep {
    color: var(--color-text-tertiary);
  }

  .action-btn.keep:hover {
    color: var(--color-success);
    background: var(--color-success-subtle);
  }

  .action-btn.drop {
    color: var(--color-text-tertiary);
  }

  .action-btn.drop:hover {
    color: var(--color-danger);
    background: var(--color-danger-subtle);
  }
</style>