<script lang="ts">
  import { onMount } from 'svelte'
  import { init, renderButton, start } from '../auth/google'
  import { signIn, type SessionState } from '../api/session'

  interface Props {
    googleClientId: string
    onsignedin: (session: SessionState) => void
  }

  let { googleClientId, onsignedin }: Props = $props()

  let button = $state<HTMLDivElement>()
  let error = $state<string | null>(null)
  let signingIn = $state(false)

  onMount(() => {
    let active = true

    init(googleClientId, async (idToken) => {
      signingIn = true
      error = null
      try {
        const session = await signIn(idToken)
        if (active) onsignedin(session)
      } catch (e) {
        if (active) error = e instanceof Error ? e.message : 'Sign-in failed.'
      } finally {
        if (active) signingIn = false
      }
    })

    start()
      .then((ready) => {
        if (!active) return
        if (ready && button) renderButton(button)
        else error = 'Google sign-in is not available.'
      })
      .catch((e) => {
        if (active) error = e instanceof Error ? e.message : 'Google sign-in is not available.'
      })

    return () => {
      active = false
    }
  })
</script>

<div class="signin">
  <div class="card">
    <svg width="28" height="28" viewBox="0 0 28 28" fill="none" aria-hidden="true">
      <rect width="28" height="28" rx="6" fill="var(--color-accent)" />
      <path d="M7 10h14M7 14h14M7 18h10" stroke="var(--color-surface)" stroke-width="2" stroke-linecap="round" />
    </svg>
    <h1>Event browser</h1>
    <p class="intro">Sign in to read the event store.</p>

    <div bind:this={button} class="google-btn"></div>

    {#if signingIn}
      <p class="hint">Signing in…</p>
    {/if}
    {#if error}
      <p class="error">{error}</p>
    {/if}
  </div>
</div>

<style>
  .signin {
    min-height: 100dvh;
    display: grid;
    place-items: center;
    padding: var(--space-6);
    background: var(--color-bg);
  }

  .card {
    width: 100%;
    max-width: 22rem;
    padding: var(--space-8);
    text-align: center;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-sm);
  }

  h1 {
    margin: var(--space-4) 0 0;
    font-size: var(--text-xl);
    font-weight: var(--weight-semibold);
  }

  .intro {
    margin: var(--space-2) 0 var(--space-6);
    font-size: var(--text-sm);
    color: var(--color-text-secondary);
  }

  .google-btn {
    display: flex;
    justify-content: center;
    min-height: 2.5rem;
  }

  .hint {
    margin: var(--space-4) 0 0;
    font-size: var(--text-sm);
    color: var(--color-text-secondary);
  }

  .error {
    margin: var(--space-4) 0 0;
    font-size: var(--text-sm);
    color: var(--color-danger);
  }
</style>
