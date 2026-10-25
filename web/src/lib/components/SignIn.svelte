<script lang="ts">
  import { renderButton } from '../auth/google'
  import { googleClientId } from '../config'

  function mountButton(node: HTMLDivElement) {
    renderButton(node)
  }
</script>

<main>
  <h1>Stash</h1>
  <p>Save text, links, images, and files to triage later.</p>

  {#if googleClientId}
    <div use:mountButton></div>
  {:else}
    <p class="error">Google sign-in is not configured (set VITE_GOOGLE_CLIENT_ID)</p>
  {/if}
</main>