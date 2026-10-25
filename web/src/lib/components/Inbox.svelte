<script lang="ts">
  import { onMount } from 'svelte'
  import { listItems, captureText, type Item } from '../api/client'

  let items = $state<Item[]>([])
  let body = $state('')
  let loading = $state(true)
  let saving = $state(false)
  let error = $state<string | null>(null)
  let selected = $state<Item | null>(null)

  onMount(async () => {
    try {
      items = await listItems()
    } catch (e) {
      error = e instanceof Error ? e.message : 'failed to load the inbox'
    } finally {
      loading = false
    }
  })

  async function save() {
    const text = body.trim()
    if (!text || saving) return
    saving = true
    error = null
    try {
      const id = await captureText(text)
      // The read model is eventually consistent, so add the item locally
      // rather than waiting for the projector to catch up.
      items = [
        { id, type: 'text', body: text, capturedAt: new Date().toISOString() },
        ...items,
      ]
      body = ''
    } catch (e) {
      error = e instanceof Error ? e.message : 'failed to save'
    } finally {
      saving = false
    }
  }
</script>

<h2>Inbox</h2>

{#if selected}
  <button type="button" onclick={() => (selected = null)}>← Back</button>
  <p class="detail-body">{selected.body}</p>
  {#if selected.sourceApp}
    <p class="muted">From {selected.sourceApp}</p>
  {/if}
{:else}
  <form onsubmit={(e) => { e.preventDefault(); save() }}>
    <textarea bind:value={body} placeholder="Save a piece of text…" rows="3"></textarea>
    <button type="submit" disabled={saving}>{saving ? 'Saving…' : 'Save'}</button>
  </form>

  {#if error}
    <p class="error">{error}</p>
  {/if}

  {#if loading}
    <p>Loading…</p>
  {:else if items.length === 0}
    <p class="empty">Your inbox is empty. Save something to triage later.</p>
  {:else}
    <ul class="items">
      {#each items as item (item.id)}
        <li>
          <button type="button" class="item" onclick={() => (selected = item)}>
            <span class="item-body">{item.body}</span>
            {#if item.sourceApp}<span class="muted">{item.sourceApp}</span>{/if}
          </button>
        </li>
      {/each}
    </ul>
  {/if}
{/if}
