<script lang="ts">
  import { renderButton } from '../auth/google'
  import { googleClientId } from '../config'

  function mountButton(node: HTMLDivElement) {
    renderButton(node)
  }
</script>

<div class="signin">
  <div class="hero">
    <div class="logo">
      <svg width="56" height="56" viewBox="0 0 32 32" fill="none">
        <rect width="32" height="32" rx="8" fill="var(--color-accent)" />
        <circle cx="20" cy="9" r="2.5" fill="var(--color-surface)" />
        <circle cx="13" cy="14" r="2.5" fill="var(--color-surface)" />
        <circle cx="19" cy="18" r="2.5" fill="var(--color-surface)" />
        <circle cx="13" cy="23" r="2.5" fill="var(--color-surface)" />
        <path d="M19 11l-4 2M14 16l4 1M18 20l-4 2" stroke="var(--color-surface)" stroke-width="1.5" stroke-linecap="round" opacity="0.5" />
      </svg>
    </div>
    <h1 class="title">Stash</h1>
    <p class="subtitle">Capture anything. Triage later.</p>
    <div class="features">
      <div class="feature">
        <span class="feature-icon">💬</span>
        <span>Save text from any app</span>
      </div>
      <div class="feature">
        <span class="feature-icon">🔗</span>
        <span>Bookmark links as you browse</span>
      </div>
      <div class="feature">
        <span class="feature-icon">🖼</span>
        <span>Stash images and files</span>
      </div>
      <div class="feature">
        <span class="feature-icon">🏷</span>
        <span>Sort and search everything</span>
      </div>
    </div>
  </div>

  <div class="auth-card">
    {#if googleClientId}
      <p class="auth-heading">Sign in to continue</p>
      <div use:mountButton class="google-btn"></div>
    {:else}
      <div class="config-error">
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
          <circle cx="10" cy="10" r="8" stroke="currentColor" stroke-width="1.5" />
          <path d="M10 7v3M10 13v.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
        </svg>
        <p>Google sign-in is not configured.</p>
        <p class="config-hint">Set <code>VITE_GOOGLE_CLIENT_ID</code> in your environment.</p>
      </div>
    {/if}
  </div>
</div>

<style>
  .signin {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 100dvh;
    padding: var(--space-8) var(--space-6);
    background: var(--color-bg);
  }

  .hero {
    text-align: center;
    margin-bottom: var(--space-10);
  }

  .logo {
    margin-bottom: var(--space-4);
    display: inline-flex;
  }

  .title {
    margin: 0 0 var(--space-2);
    font-size: var(--text-3xl);
    font-weight: var(--weight-bold);
    color: var(--color-text);
    letter-spacing: -0.02em;
  }

  .subtitle {
    margin: 0 0 var(--space-8);
    font-size: var(--text-lg);
    color: var(--color-text-secondary);
  }

  .features {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-3);
    max-width: 340px;
    margin: 0 auto;
  }

  .feature {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--text-sm);
    color: var(--color-text-secondary);
    text-align: left;
  }

  .feature-icon {
    font-size: var(--text-lg);
    flex-shrink: 0;
  }

  .auth-card {
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-xl);
    padding: var(--space-8) var(--space-8) var(--space-8);
    width: 100%;
    max-width: 320px;
    text-align: center;
    box-shadow: var(--shadow-md);
  }

  .auth-heading {
    margin: 0 0 var(--space-4);
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    color: var(--color-text-secondary);
  }

  .google-btn {
    display: flex;
    justify-content: center;
  }

  .config-error {
    color: var(--color-danger);
    font-size: var(--text-sm);
  }

  .config-error p {
    margin: var(--space-2) 0 0;
  }

  .config-hint {
    font-size: var(--text-xs);
    color: var(--color-text-tertiary);
    margin-top: var(--space-2);
  }

  .config-hint code {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    background: var(--color-bg);
    padding: 1px var(--space-1);
    border-radius: var(--radius-sm);
  }
</style>