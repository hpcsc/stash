<script lang="ts">
  import { onMount } from 'svelte'
  import AppShell from './lib/components/AppShell.svelte'
  import SignIn from './lib/components/SignIn.svelte'
  import Inbox from './lib/components/Inbox.svelte'
  import Library from './lib/components/Library.svelte'
  import Trash from './lib/components/Trash.svelte'
  import { token, signIn, signOut } from './lib/auth/session'
  import { init, start } from './lib/auth/google'

  let waiting = $state($token === null)
  let showSignIn = $state(false)
  let activeTab = $state<string>('inbox')

  onMount(async () => {
    if ($token !== null) return
    try {
      init(async (idToken) => {
        try {
          await signIn(idToken)
        } catch {
          showSignIn = true
        }
      })
      await start()
    } catch {
      // Google client ID not set — SignIn will show the error.
    } finally {
      waiting = false
      if (!showSignIn) showSignIn = true
    }
  })

  function onTabChange(e: Event) {
    activeTab = (e as CustomEvent).detail
  }

  $effect(() => {
    window.addEventListener('tab-change', onTabChange)
    return () => window.removeEventListener('tab-change', onTabChange)
  })
</script>

{#snippet signOutButton()}
  <button type="button" class="signout-btn" onclick={() => signOut()}>
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
      <path d="M5 2H3a1 1 0 0 0-1 1v8a1 1 0 0 0 1 1h2M9 10l3-3-3-3M12 7H5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
    </svg>
    Sign out
  </button>
{/snippet}

{#if $token}
  <AppShell headerActions={signOutButton}>
    {#if activeTab === 'inbox'}
      <Inbox />
    {:else if activeTab === 'library'}
      <Library />
    {:else if activeTab === 'trash'}
      <Trash />
    {/if}
  </AppShell>
{:else if waiting}
  <div class="loading-screen">
    <svg class="loading-logo" width="32" height="32" viewBox="0 0 28 28" fill="none">
      <rect width="28" height="28" rx="6" fill="var(--color-accent)" />
      <path d="M7 10h14M7 14h14M7 18h10" stroke="var(--color-surface)" stroke-width="2" stroke-linecap="round" />
    </svg>
    <span>Loading…</span>
  </div>
{:else}
  <SignIn />
{/if}

<style>
  .signout-btn {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-3);
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    color: var(--color-text-secondary);
    background: transparent;
    border-radius: var(--radius-sm);
    line-height: 1.8;
  }

  .signout-btn:hover {
    color: var(--color-text);
    background: var(--color-surface-hover);
  }

  .loading-screen {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-4);
    min-height: 100dvh;
    font-size: var(--text-sm);
    color: var(--color-text-secondary);
  }

  .loading-logo {
    animation: logo-pulse 2s ease-in-out infinite;
  }

  @keyframes logo-pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.5; }
  }
</style>