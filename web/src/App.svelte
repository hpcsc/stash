<script lang="ts">
  import { onMount } from 'svelte'
  import SignIn from './lib/components/SignIn.svelte'
  import Inbox from './lib/components/Inbox.svelte'
  import { token, signIn, signOut } from './lib/auth/session'
  import { init, start } from './lib/auth/google'

  let waiting = $state($token === null)
  let showSignIn = $state(false)

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
</script>

{#if $token}
  <header>
    <h1>Stash</h1>
    <button type="button" onclick={() => signOut()}>Sign out</button>
  </header>
  <main>
    <Inbox />
  </main>
{:else if waiting}
  <main>
    <p>Loading…</p>
  </main>
{:else}
  <SignIn />
{/if}