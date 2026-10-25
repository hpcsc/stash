<script lang="ts">
  interface Props {
    title: string
    description: string
  }

  let { title, description }: Props = $props()
</script>

<div class="empty">
  <div class="icon">
    <svg width="40" height="40" viewBox="0 0 40 40" fill="none">
      <rect x="8" y="8" width="24" height="28" rx="3" stroke="var(--color-text-tertiary)" stroke-width="1.5" />
      <line x1="14" y1="16" x2="26" y2="16" stroke="var(--color-text-tertiary)" stroke-width="1.5" stroke-linecap="round" />
      <line x1="14" y1="20" x2="26" y2="20" stroke="var(--color-text-tertiary)" stroke-width="1.5" stroke-linecap="round" />
      <line x1="14" y1="24" x2="22" y2="24" stroke="var(--color-text-tertiary)" stroke-width="1.5" stroke-linecap="round" />
    </svg>
  </div>
  <h3 class="title">{title}</h3>
  <p class="desc">{description}</p>
</div>

<style>
  .empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: var(--space-12) var(--space-6);
    text-align: center;
  }

  .icon {
    margin-bottom: var(--space-4);
    opacity: 0.5;
  }

  .title {
    margin: 0 0 var(--space-2);
    font-size: var(--text-lg);
    font-weight: var(--weight-semibold);
    color: var(--color-text-secondary);
  }

  .desc {
    margin: 0;
    font-size: var(--text-sm);
    color: var(--color-text-tertiary);
    max-width: 300px;
  }
</style>