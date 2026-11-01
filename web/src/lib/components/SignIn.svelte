<script lang="ts">
  import { renderButton } from '../auth/google'
  import { googleClientId } from '../config'

  function mountButton(node: HTMLDivElement) {
    renderButton(node)
  }

  const features = [
    { icon: '💬', title: 'Save text from any app', text: 'Clip a thought, note, or snippet in a keystroke.' },
    { icon: '🔗', title: 'Bookmark links as you browse', text: 'Keep the pages you want to come back to.' },
    { icon: '🖼', title: 'Stash images and files', text: 'Drop in pictures and documents for later.' },
    { icon: '🏷', title: 'Sort and search everything', text: 'Triage your inbox when you have a moment.' },
  ]
</script>

<div class="signin">
  <section class="brand-panel">
    <div class="brand">
      <svg class="logo" width="36" height="36" viewBox="0 0 28 28" fill="none">
        <rect width="28" height="28" rx="6" fill="var(--color-accent)" />
        <path d="M7 10h14M7 14h14M7 18h10" stroke="var(--color-surface)" stroke-width="2" stroke-linecap="round" />
      </svg>
      <span class="brand-name">Stash</span>
    </div>

    <div class="intro">
      <h1 class="title">Capture anything.<br />Triage later.</h1>
      <p class="subtitle">Save text, links, images and files into one inbox, then sort them when you're ready.</p>
    </div>

    <ul class="features">
      {#each features as feature}
        <li class="feature">
          <span class="feature-icon" aria-hidden="true">{feature.icon}</span>
          <div class="feature-body">
            <span class="feature-title">{feature.title}</span>
            <span class="feature-text">{feature.text}</span>
          </div>
        </li>
      {/each}
    </ul>
  </section>

  <section class="auth-panel">
    <div class="auth-card">
      {#if googleClientId}
        <p class="auth-heading">Sign in to continue</p>
        <div use:mountButton class="google-btn"></div>
        <p class="auth-note">One account for the web, the browser extension and the app.</p>
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
  </section>
</div>

<style>
  .signin {
    min-height: 100dvh;
    display: grid;
    grid-template-columns: minmax(0, 1.05fr) minmax(0, 1fr);
    background: var(--color-bg);
  }

  .brand-panel {
    display: flex;
    flex-direction: column;
    justify-content: center;
    gap: var(--space-10);
    padding: clamp(var(--space-8), 6vw, var(--space-16));
    background: linear-gradient(160deg, var(--color-accent-subtle), var(--color-bg) 70%);
  }

  .brand {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .logo {
    flex-shrink: 0;
  }

  .brand-name {
    font-size: var(--text-xl);
    font-weight: var(--weight-bold);
    letter-spacing: -0.01em;
  }

  .intro {
    max-width: 30rem;
  }

  .title {
    margin: 0;
    font-size: clamp(var(--text-3xl), 4vw, 3.5rem);
    line-height: 1.1;
    font-weight: var(--weight-bold);
    letter-spacing: -0.03em;
  }

  .subtitle {
    margin: var(--space-5) 0 0;
    max-width: 28rem;
    font-size: var(--text-lg);
    line-height: var(--leading);
    color: var(--color-text-secondary);
  }

  .features {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--space-4) var(--space-6);
    max-width: 34rem;
  }

  .feature {
    display: flex;
    gap: var(--space-3);
    align-items: flex-start;
  }

  .feature-icon {
    display: grid;
    place-items: center;
    flex-shrink: 0;
    width: 2.5rem;
    height: 2.5rem;
    font-size: 1.25rem;
    background: var(--color-surface);
    border: 1px solid var(--color-border-light);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-xs);
  }

  .feature-body {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .feature-title {
    font-size: var(--text-sm);
    font-weight: var(--weight-semibold);
    color: var(--color-text);
  }

  .feature-text {
    font-size: var(--text-sm);
    line-height: 1.45;
    color: var(--color-text-secondary);
  }

  .auth-panel {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-10);
    background: var(--color-surface);
    border-left: 1px solid var(--color-border-light);
  }

  .auth-card {
    width: 100%;
    max-width: 22rem;
    text-align: center;
  }

  .auth-heading {
    margin: 0 0 var(--space-5);
    font-size: var(--text-lg);
    font-weight: var(--weight-semibold);
    color: var(--color-text);
  }

  .google-btn {
    display: flex;
    justify-content: center;
  }

  .auth-note {
    margin: var(--space-6) 0 0;
    font-size: var(--text-xs);
    line-height: 1.5;
    color: var(--color-text-tertiary);
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

  @media (max-width: 860px) {
    .signin {
      grid-template-columns: 1fr;
    }

    .brand-panel {
      gap: var(--space-8);
      padding: var(--space-10) var(--space-6) var(--space-12);
    }

    .features {
      grid-template-columns: 1fr;
    }

    .auth-panel {
      border-left: none;
      border-top: 1px solid var(--color-border-light);
      padding: var(--space-10) var(--space-6);
    }
  }
</style>
