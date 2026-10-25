<script lang="ts">
  import type { Snippet } from 'svelte'

  interface Props {
    children: Snippet
    headerActions?: Snippet
  }

  let { children, headerActions }: Props = $props()

  const tabs = [
    { id: 'inbox', label: 'Inbox' },
    { id: 'library', label: 'Library' },
    { id: 'trash', label: 'Trash' },
  ] as const

  type Tab = (typeof tabs)[number]['id']

  let active: Tab = $state('inbox')

  function onTabClick(id: Tab) {
    active = id
    window.dispatchEvent(new CustomEvent('tab-change', { detail: id }))
  }
</script>

<div class="shell">
  <header class="header">
    <div class="header-left">
      <svg class="logo" width="32" height="32" viewBox="0 0 28 28" fill="none">
        <rect width="28" height="28" rx="6" fill="var(--color-accent)" />
        <path d="M7 10h14M7 14h14M7 18h10" stroke="var(--color-surface)" stroke-width="2" stroke-linecap="round" />
      </svg>
      <span class="brand">Stash</span>
    </div>
    <nav class="tabs">
      {#each tabs as { id, label }}
        <button
          type="button"
          class="tab"
          class:active={active === id}
          onclick={() => onTabClick(id)}
        >
          {label}
        </button>
      {/each}
    </nav>
    <div class="header-right">
      {#if headerActions}
        {@render headerActions()}
      {/if}
    </div>
  </header>
  <main class="main">
    {@render children()}
  </main>
</div>

<style>
  .shell {
    min-height: 100dvh;
    display: flex;
    flex-direction: column;
  }

  .header {
    position: sticky;
    top: 0;
    z-index: 10;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 var(--space-6);
    height: 56px;
    background: var(--color-surface);
    border-bottom: 1px solid var(--color-border);
    box-shadow: var(--shadow-xs);
    backdrop-filter: blur(12px);
  }

  @supports (backdrop-filter: blur(12px)) {
    .header {
      background: color-mix(in srgb, var(--color-surface) 85%, transparent);
    }
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    min-width: 120px;
  }

  .logo {
    flex-shrink: 0;
  }

  .brand {
    font-size: var(--text-lg);
    font-weight: var(--weight-bold);
    color: var(--color-text);
    letter-spacing: -0.01em;
  }

  .tabs {
    display: flex;
    gap: var(--space-1);
    background: var(--color-bg);
    border: 1px solid var(--color-border-light);
    border-radius: var(--radius-md);
    padding: 3px;
  }

  .tab {
    padding: var(--space-2) var(--space-5);
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    color: var(--color-text-secondary);
    background: transparent;
    border-radius: var(--radius-sm);
    line-height: 1.4;
    transition: all var(--duration-fast) var(--ease-out);
  }

  .tab:hover {
    color: var(--color-text);
    background: var(--color-surface);
  }

  .tab.active {
    color: var(--color-text);
    background: var(--color-surface);
    box-shadow: var(--shadow-xs);
  }

  .header-right {
    display: flex;
    align-items: center;
    min-width: 120px;
    justify-content: flex-end;
  }

  .main {
    flex: 1;
    max-width: 44rem;
    width: 100%;
    margin: 0 auto;
    padding: var(--space-10) var(--space-6);
  }
</style>