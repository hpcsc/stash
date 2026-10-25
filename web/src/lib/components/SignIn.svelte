<script lang="ts">
  import { onMount } from 'svelte'
  import { initGoogleSignIn, renderGoogleButton } from '../auth/google'
  import { signIn } from '../auth/session'

  let error = $state<string | null>(null)
  let signingIn = $state(false)
  let ready = $state(false)

  onMount(async () => {
    try {
      await initGoogleSignIn(async (idToken) => {
        signingIn = true
        error = null
        try {
          await signIn(idToken)
        } catch (e) {
          error = e instanceof Error ? e.message : 'sign-in failed'
          signingIn = false
        }
      })
      ready = true
    } catch (e) {
      error = e instanceof Error ? e.message : 'sign-in failed'
    }
  })

  function renderButton(node: HTMLDivElement) {
    renderGoogleButton(node)
  }
</script>

<main>
  <h1>Stash</h1>
  <p>Save text, links, images, and files to triage later.</p>

  {#if error}
    <p class="error">{error}</p>
  {/if}

  {#if signingIn}
    <p>Signing in…</p>
  {:else if ready}
    <div use:renderButton></div>
  {:else}
    <p>Loading…</p>
  {/if}
</main>