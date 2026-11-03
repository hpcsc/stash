<script lang="ts">
  import { onMount } from 'svelte'
  import EventBrowser from './EventBrowser.svelte'
  import EventBrowserSignIn from './EventBrowserSignIn.svelte'
  import { readSession, signOut, type SessionState } from '../api/session'
  import { setUnauthorizedHandler } from '../api/events'

  let session = $state<SessionState | null>(null)
  let error = $state<string | null>(null)

  async function refresh() {
    error = null
    try {
      session = await readSession()
    } catch (e) {
      session = null
      error = e instanceof Error ? e.message : 'The event browser could not read the session.'
    }
  }

  async function signOutAndRefresh() {
    try {
      await signOut()
    } finally {
      await refresh()
    }
  }

  onMount(() => {
    setUnauthorizedHandler(() => {
      if (session) session = { ...session, signedIn: false }
    })
    refresh()
    return () => setUnauthorizedHandler(null)
  })
</script>

{#if error}
  <div class="gate">
    <p class="error">{error}</p>
    <button type="button" class="retry" onclick={refresh}>Retry</button>
  </div>
{:else if session === null}
  <div class="gate">
    <p class="hint">Loading…</p>
  </div>
{:else if !session.signedIn}
  <EventBrowserSignIn googleClientId={session.googleClientId} onsignedin={(next) => (session = next)} />
{:else}
  <EventBrowser onsignout={session.canSignOut ? signOutAndRefresh : undefined} />
{/if}

<style>
  .gate {
    min-height: 100dvh;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-4);
    padding: var(--space-6);
    text-align: center;
  }

  .hint {
    font-size: var(--text-sm);
    color: var(--color-text-secondary);
  }

  .error {
    max-width: 28rem;
    font-size: var(--text-sm);
    color: var(--color-danger);
  }

  .retry {
    padding: var(--space-2) var(--space-4);
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    color: var(--color-text);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
  }
</style>
