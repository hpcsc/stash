<script lang="ts">
  interface Props {
    placeholder?: string
    onsearch?: (query: string) => void
  }

  let { placeholder = 'Search…', onsearch }: Props = $props()

  let value = $state('')
  let timer: ReturnType<typeof setTimeout> | undefined

  function handleInput() {
    clearTimeout(timer)
    timer = setTimeout(() => {
      onsearch?.(value.trim())
    }, 250)
  }

  function handleClear() {
    value = ''
    clearTimeout(timer)
    onsearch?.('')
  }
</script>

<div class="search">
  <svg class="search-icon" width="16" height="16" viewBox="0 0 16 16" fill="none">
    <circle cx="7" cy="7" r="5" stroke="currentColor" stroke-width="1.5" />
    <path d="M11 11l3 3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
  </svg>
  <input
    type="text"
    bind:value
    oninput={handleInput}
    {placeholder}
    class="search-input"
  />
  {#if value}
    <button type="button" class="clear-btn" onclick={handleClear} aria-label="Clear search">
      <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
        <path d="M3 3l8 8M11 3l-8 8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
      </svg>
    </button>
  {/if}
</div>

<style>
  .search {
    position: relative;
    display: flex;
    align-items: center;
    margin-bottom: var(--space-6);
  }

  .search-icon {
    position: absolute;
    left: var(--space-3);
    color: var(--color-text-tertiary);
    pointer-events: none;
  }

  .search-input {
    width: 100%;
    padding: var(--space-2) var(--space-3) var(--space-2) var(--space-8);
    font-size: var(--text-sm);
    border-radius: var(--radius-md);
    color: var(--color-text);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    box-shadow: var(--shadow-xs);
    transition: border-color var(--duration-fast) var(--ease-out),
                box-shadow var(--duration-fast) var(--ease-out);
  }

  .search-input:focus {
    outline: none;
    border-color: var(--color-accent);
    box-shadow: 0 0 0 3px var(--color-accent-subtle);
  }

  .search-input::placeholder {
    color: var(--color-text-tertiary);
  }

  .clear-btn {
    position: absolute;
    right: var(--space-2);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    padding: 0;
    color: var(--color-text-tertiary);
    background: transparent;
    border-radius: var(--radius-sm);
  }

  .clear-btn:hover {
    color: var(--color-text);
    background: var(--color-surface-hover);
  }
</style>