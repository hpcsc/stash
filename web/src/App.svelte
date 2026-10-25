<script lang="ts">
  import SignIn from './lib/components/SignIn.svelte'
  import Inbox from './lib/components/Inbox.svelte'
  import { token, signOut } from './lib/auth/session'
</script>

{#if $token}
  <header>
    <h1>Stash</h1>
    <button type="button" onclick={() => signOut()}>Sign out</button>
  </header>
  <main>
    <Inbox />
  </main>
{:else}
  <SignIn />
{/if}
