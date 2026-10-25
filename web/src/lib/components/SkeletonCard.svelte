<script lang="ts">
</script>

<div class="skeleton">
  <div class="skel-header">
    <div class="skel-badge pulse"></div>
    <div class="skel-time pulse"></div>
  </div>
  <div class="skel-lines">
    <div class="skel-line pulse" style="width: 90%"></div>
    <div class="skel-line pulse" style="width: 65%"></div>
  </div>
  <div class="skel-footer">
    <div class="skel-source pulse"></div>
  </div>
</div>

<style>
  .skeleton {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    padding: var(--space-4);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
  }

  .skel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .skel-badge {
    width: 28px;
    height: 28px;
    border-radius: var(--radius-sm);
    background: var(--color-border);
  }

  .skel-time {
    width: 60px;
    height: 12px;
    border-radius: var(--radius-sm);
    background: var(--color-border-light);
  }

  .skel-lines {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .skel-line {
    height: 14px;
    border-radius: var(--radius-sm);
    background: var(--color-border-light);
  }

  .skel-footer {
    display: flex;
  }

  .skel-source {
    width: 80px;
    height: 20px;
    border-radius: var(--radius-sm);
    background: var(--color-border);
  }

  .pulse {
    animation: skeleton-pulse 1.8s ease-in-out infinite;
  }

  @keyframes skeleton-pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.4; }
  }
</style>