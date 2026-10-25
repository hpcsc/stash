<script lang="ts">
  import SignIn from './lib/components/SignIn.svelte'
  import { token, signOut } from './lib/auth/session'
</script>

{#if $token}
  <main>
    <h1>Stash</h1>
    <p>Signed in. The Inbox / Library / Trash screens land with APP-002–APP-004.</p>
    <button type="button" onclick={() => signOut()}>Sign out</button>
  </main>
{:else}
  <SignIn />
{/if}
