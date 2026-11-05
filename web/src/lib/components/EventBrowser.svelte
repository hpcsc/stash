<script lang="ts">
  import { onMount } from 'svelte'
  import {
    listEvents,
    listTypes,
    listCheckpoints,
    listProjection,
    listStreams,
    streamEvents,
    openTail,
    type EventRow,
    type Checkpoint,
    type Checkpoints,
    type ProjectionFilter,
    type ProjectionOperator,
    type ProjectionTable,
    type StreamSummary,
    type Position,
  } from '../api/events'
  import { formatDate, relativeTime } from '../time'
  import ErrorBanner from './ErrorBanner.svelte'
  import EmptyState from './EmptyState.svelte'

  interface Props {
    onsignout?: () => void
  }

  let { onsignout }: Props = $props()

  type View = 'events' | 'streams' | 'checkpoints'
  type CheckpointTab = 'projections' | 'reactions' | 'other'

  const checkpointTabs: { id: CheckpointTab; label: string }[] = [
    { id: 'projections', label: 'Projections' },
    { id: 'reactions', label: 'Reactions' },
    { id: 'other', label: 'Other' },
  ]

  interface StreamNode {
    name: string
    path: string
    stream?: StreamSummary
    children: StreamNode[]
  }

  let view = $state<View>('events')
  let checkpointTab = $state<CheckpointTab>('projections')

  let events = $state<EventRow[]>([])
  let next = $state<Position | null>(null)
  let loading = $state(true)
  let loadingMore = $state(false)

  let types = $state<string[]>([])
  let typeFilter = $state('')
  let streamPrefix = $state('')
  let correlationId = $state('')
  let since = $state('')
  let until = $state('')

  let streams = $state<StreamSummary[]>([])
  let loadingStreams = $state(false)
  let expandedDirs = $state(new Set<string>())
  let expandedStream = $state<string | null>(null)
  let streamEventsList = $state<EventRow[]>([])
  let loadingStream = $state(false)

  let checkpoints = $state<Checkpoints | null>(null)
  let loadingCheckpoints = $state(false)
  let checkpointGroups = $derived.by(() => {
    const all = checkpoints?.checkpoints ?? []
    return {
      projections: all.filter((cp) => cp.kind === 'projection'),
      reactions: all.filter((cp) => cp.kind === 'reaction'),
      other: all.filter((cp) => cp.kind !== 'projection' && cp.kind !== 'reaction'),
    }
  })

  let expandedProjection = $state<string | null>(null)
  let projectionTable = $state<ProjectionTable | null>(null)
  let loadingProjection = $state(false)
  let projectionFilter = $state<ProjectionFilter>({ column: '', operator: 'contains', value: '' })
  let projectionRequest = 0
  let projectionTimer: ReturnType<typeof setTimeout> | null = null

  let error = $state<string | null>(null)
  let selected = $state<EventRow | null>(null)

  let live = $state(false)
  let closeTail = $state<(() => void) | null>(null)

  let streamTree = $derived(buildStreamTree(streams))
  let visibleStreamRows = $derived.by(() => {
    const rows: { node: StreamNode; depth: number }[] = []
    const walk = (nodes: StreamNode[], depth: number) => {
      for (const node of nodes) {
        rows.push({ node, depth })
        if (node.children.length > 0 && expandedDirs.has(node.path)) walk(node.children, depth + 1)
      }
    }
    walk(streamTree, 0)
    return rows
  })

  let activeList = $derived(view === 'streams' ? streamEventsList : events)
  let selectedIndex = $derived.by(() => {
    const current = selected
    if (!current) return -1
    return activeList.findIndex((e) => e.id === current.id)
  })

  onMount(() => {
    loadEvents()
    return () => {
      closeTail?.()
      clearProjectionTimer()
    }
  })

  async function switchView(next: View) {
    view = next
    error = null
    selected = null
    expandedStream = null
    closeProjection()
    if (next === 'streams') await loadStreams()
    if (next === 'checkpoints') await loadCheckpoints()
  }

  function switchCheckpointTab(next: CheckpointTab) {
    if (checkpointTab === next) return
    checkpointTab = next
    closeProjection()
  }

  function tabLabel(tab: CheckpointTab): string {
    if (tab === 'projections') return 'projections'
    if (tab === 'reactions') return 'reactions'
    return 'other consumers'
  }

  async function loadEvents() {
    loading = true
    error = null
    try {
      const [page, availableTypes] = await Promise.all([
        listEvents({ types: typeFilter ? [typeFilter] : [], streamPrefix, correlationId, since: iso(since), until: iso(until) }),
        listTypes(),
      ])
      events = page.events
      next = page.next ?? null
      types = availableTypes
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load events'
    } finally {
      loading = false
    }
  }

  async function loadMore() {
    if (!next || loadingMore) return
    loadingMore = true
    error = null
    try {
      const page = await listEvents({
        after: next,
        types: typeFilter ? [typeFilter] : [],
        streamPrefix,
        correlationId,
        since: iso(since),
        until: iso(until),
      })
      events = [...events, ...page.events]
      next = page.next ?? null
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load more events'
    } finally {
      loadingMore = false
    }
  }

  async function loadStreams() {
    loadingStreams = true
    error = null
    try {
      streams = await listStreams()
      expandedDirs = autoExpand(streams)
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load streams'
    } finally {
      loadingStreams = false
    }
  }

  function toggleDir(path: string) {
    const next = new Set(expandedDirs)
    if (next.has(path)) next.delete(path)
    else next.add(path)
    expandedDirs = next
  }

  async function toggleStream(streamId: string) {
    if (expandedStream === streamId) {
      expandedStream = null
      streamEventsList = []
      return
    }
    expandedStream = streamId
    loadingStream = true
    error = null
    try {
      streamEventsList = await streamEvents(streamId)
    } catch (e) {
      streamEventsList = []
      error = e instanceof Error ? e.message : 'Failed to load stream'
    } finally {
      loadingStream = false
    }
  }

  async function loadCheckpoints() {
    loadingCheckpoints = true
    error = null
    try {
      checkpoints = await listCheckpoints()
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load checkpoints'
    } finally {
      loadingCheckpoints = false
    }
  }

  async function toggleProjection(name: string) {
    if (expandedProjection === name) {
      closeProjection()
      return
    }
    clearProjectionTimer()
    expandedProjection = name
    projectionFilter = { column: '', operator: 'contains', value: '' }
    projectionTable = null
    await loadProjection(name)
  }

  function closeProjection() {
    clearProjectionTimer()
    projectionRequest++
    expandedProjection = null
    projectionTable = null
    loadingProjection = false
  }

  async function loadProjection(name: string) {
    const request = ++projectionRequest
    loadingProjection = true
    error = null
    try {
      const filter = projectionFilter.column ? { ...projectionFilter } : undefined
      const table = await listProjection(name, filter)
      if (request !== projectionRequest) return
      projectionTable = table
    } catch (e) {
      if (request !== projectionRequest) return
      error = e instanceof Error ? e.message : 'Failed to load the projection'
    } finally {
      if (request === projectionRequest) loadingProjection = false
    }
  }

  function setProjectionColumn(column: string) {
    projectionFilter.column = column
    applyProjectionFilter()
  }

  function setProjectionOperator(operator: ProjectionOperator) {
    projectionFilter.operator = operator
    applyProjectionFilter()
  }

  function setProjectionValue(value: string) {
    projectionFilter.value = value
    clearProjectionTimer()
    projectionTimer = setTimeout(applyProjectionFilter, 300)
  }

  function applyProjectionFilter() {
    clearProjectionTimer()
    if (expandedProjection) loadProjection(expandedProjection)
  }

  function clearProjectionTimer() {
    if (projectionTimer !== null) {
      clearTimeout(projectionTimer)
      projectionTimer = null
    }
  }

  function projectionSummary(table: ProjectionTable): string {
    const noun = table.total === 1 ? 'row' : 'rows'
    return table.rows.length < table.total
      ? `showing first ${table.rows.length} of ${table.total} ${noun}`
      : `${table.total} ${noun}`
  }

  function retry() {
    if (view === 'events') loadEvents()
    else if (view === 'streams') (expandedStream ? toggleStream(expandedStream) : loadStreams())
    else loadCheckpoints()
  }

  function toggleLive() {
    if (live) {
      closeTail?.()
      closeTail = null
      live = false
      return
    }
    live = true
    closeTail = openTail((event) => {
      events = [...events, event]
    })
  }

  function onTypeChange() {
    loadEvents()
  }

  function stepSelected(delta: number) {
    const target = activeList[selectedIndex + delta]
    if (target) selected = target
  }

  function pretty(value: unknown): string {
    return JSON.stringify(value, null, 2)
  }

  function iso(datetimeLocal: string): string | undefined {
    return datetimeLocal ? new Date(datetimeLocal).toISOString() : undefined
  }

  const STALE_AFTER_MS = 5 * 60 * 1000

  function isStale(checkpoint: Checkpoint): boolean {
    return checkpoint.lag > 0 && Date.now() - new Date(checkpoint.updatedAt).getTime() > STALE_AFTER_MS
  }

  function positionTitle(position: Position): string {
    return `transaction ${position.transactionId}, sequence ${position.sequence}`
  }

  function buildStreamTree(streams: StreamSummary[]): StreamNode[] {
    const roots: StreamNode[] = []
    const byPath = new Map<string, StreamNode>()
    for (const stream of streams) {
      const parts = stream.streamId.split('/')
      let siblings = roots
      let parentPath = ''
      for (let i = 0; i < parts.length; i++) {
        const name = parts[i]
        const path = parentPath ? `${parentPath}/${name}` : name
        parentPath = path
        let node = byPath.get(path)
        if (!node) {
          node = { name, path, children: [] }
          byPath.set(path, node)
          siblings.push(node)
        }
        if (i === parts.length - 1) node.stream = stream
        siblings = node.children
      }
    }
    sortNodes(roots)
    return roots
  }

  function sortNodes(nodes: StreamNode[]) {
    nodes.sort((a, b) => a.name.localeCompare(b.name))
    for (const node of nodes) sortNodes(node.children)
  }

  function autoExpand(streams: StreamSummary[]): Set<string> {
    const expanded = new Set<string>()
    const walk = (nodes: StreamNode[]) => {
      for (const node of nodes) {
        if (node.children.length === 0) continue
        expanded.add(node.path)
        if (node.children.length === 1) walk(node.children)
      }
    }
    walk(buildStreamTree(streams))
    return expanded
  }
</script>

<div class="page">
  <header class="page-header">
    <div class="brand">
      <svg width="24" height="24" viewBox="0 0 28 28" fill="none" aria-hidden="true">
        <rect width="28" height="28" rx="6" fill="var(--color-accent)" />
        <path d="M7 10h14M7 14h14M7 18h10" stroke="var(--color-surface)" stroke-width="2" stroke-linecap="round" />
      </svg>
      <span>Event browser</span>
    </div>
    <div class="header-actions">
      <span class="read-only">Read-only</span>
      {#if onsignout}
        <button type="button" class="ghost-btn" onclick={onsignout}>Sign out</button>
      {/if}
    </div>
  </header>

  <div class="events">
    <div class="top-panel">
    <div class="view-switcher">
      <button type="button" class="segment" class:active={view === 'events'} onclick={() => switchView('events')}>Events</button>
      <button type="button" class="segment" class:active={view === 'streams'} onclick={() => switchView('streams')}>Streams</button>
      <button type="button" class="segment" class:active={view === 'checkpoints'} onclick={() => switchView('checkpoints')}>Checkpoints</button>
    </div>

    <div class="toolbar">
      {#if view === 'events'}
        <button
          type="button"
          class="ghost-btn"
          class:active={live}
          onclick={toggleLive}
        >
          {live ? 'Pause live' : 'Live'}
        </button>
      {/if}
    </div>

    {#if view === 'events'}
      <div class="filters">
        <label class="field">
          <span>Type</span>
          <div class="select-wrap">
            <select bind:value={typeFilter} onchange={onTypeChange}>
              <option value="">All types</option>
              {#each types as type (type)}
                <option value={type}>{type}</option>
              {/each}
            </select>
            <svg class="chevron" width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true">
              <path d="M3 4.5 6 7.5l3-3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
            </svg>
          </div>
        </label>
        <label class="field grow">
          <span>Stream prefix</span>
          <input type="text" bind:value={streamPrefix} placeholder="user/…" />
        </label>
        <label class="field grow">
          <span>Correlation id</span>
          <input type="text" bind:value={correlationId} placeholder="uuid" />
        </label>
        <label class="field">
          <span>Since</span>
          <input type="datetime-local" bind:value={since} />
        </label>
        <label class="field">
          <span>Until</span>
          <input type="datetime-local" bind:value={until} />
        </label>
        <button type="button" class="apply-btn" onclick={loadEvents}>Apply</button>
      </div>
    {/if}
  </div>

  {#if error}
    <ErrorBanner message={error} onretry={retry} />
  {/if}

  {#if view === 'events'}
    <div class="content">
      <div class="list-pane">
        <div class="section-header">
          <h2>Events</h2>
          <span class="count">{events.length} shown</span>
        </div>

        {#if loading}
          <p class="hint">Loading…</p>
        {:else if events.length === 0}
          <EmptyState title="No events" description="The store has no events matching the current scope and filters." />
        {:else}
          <div class="list">
            {#each events as event (event.id)}
              <button type="button" class="event-row" class:selected={selected?.id === event.id} onclick={() => (selected = event)}>
                {@render EventLine({ event })}
              </button>
            {/each}
          </div>
          {#if next}
            <div class="more">
              <button type="button" class="ghost-btn" onclick={loadMore} disabled={loadingMore}>
                {loadingMore ? 'Loading…' : 'Load more'}
              </button>
            </div>
          {/if}
        {/if}
      </div>

      {#if selected}
        {@render Detail(selected)}
      {/if}
    </div>
  {:else if view === 'streams'}
    <div class="content">
      <div class="list-pane">
        <div class="section-header">
          <h2>Streams</h2>
          <span class="count">{streams.length} streams</span>
        </div>

        {#if loadingStreams}
          <p class="hint">Loading…</p>
        {:else if streams.length === 0}
          <EmptyState title="No streams" description="The store has no streams matching the current scope." />
        {:else}
          <div class="stream-tree">
            {#each visibleStreamRows as { node, depth } (node.path)}
              {#if node.stream}
                <div class="tree-row" style:padding-left={`${depth * 16}px`}>
                  <button type="button" class="tree-toggle" onclick={() => toggleStream(node.path)}>
                    <span class="caret" class:open={expandedStream === node.path}>›</span>
                    <span class="tree-name">{node.name}</span>
                    <span class="line-meta">{node.stream.eventCount} event{node.stream.eventCount !== 1 ? 's' : ''} · v{node.stream.lastVersion}</span>
                  </button>
                </div>
                {#if expandedStream === node.path}
                  {#if loadingStream}
                    <p class="hint" style:padding-left={`${(depth + 1) * 16}px`}>Loading…</p>
                  {:else if streamEventsList.length === 0}
                    <p class="hint" style:padding-left={`${(depth + 1) * 16}px`}>No events.</p>
                  {:else}
                    <div class="list stream-events" style:padding-left={`${(depth + 1) * 16}px`}>
                      {#each streamEventsList as event (event.id)}
                        <button type="button" class="event-row" class:selected={selected?.id === event.id} onclick={() => (selected = event)}>
                          {@render EventLine({ event })}
                        </button>
                      {/each}
                    </div>
                  {/if}
                {/if}
              {:else}
                <div class="tree-row" style:padding-left={`${depth * 16}px`}>
                  <button type="button" class="tree-toggle" onclick={() => toggleDir(node.path)}>
                    <span class="caret" class:open={expandedDirs.has(node.path)}>›</span>
                    <span class="tree-name dir">{node.name}</span>
                  </button>
                </div>
              {/if}
            {/each}
          </div>
        {/if}
      </div>

      {#if selected}
        {@render Detail(selected)}
      {/if}
    </div>
  {:else}
    <div class="section-header">
      <h2>Checkpoints</h2>
      {#if checkpoints?.lastPosition}
        <span class="count" title={positionTitle(checkpoints.lastPosition)}>head event #{checkpoints.lastPosition.sequence}</span>
      {/if}
    </div>

    {#if loadingCheckpoints}
      <p class="hint">Loading…</p>
    {:else if !checkpoints || checkpoints.checkpoints.length === 0}
      <EmptyState title="No checkpoints" description="Consumers write a checkpoint as they read the store." />
    {:else}
      <div class="checkpoint-tabs" role="tablist" aria-label="Checkpoint kind">
        {#each checkpointTabs as tab}
          {#if tab.id !== 'other' || checkpointGroups.other.length > 0}
            <button
              type="button"
              class="segment"
              class:active={checkpointTab === tab.id}
              role="tab"
              aria-selected={checkpointTab === tab.id}
              onclick={() => switchCheckpointTab(tab.id)}
            >
              {tab.label}
              <span class="tab-count">{checkpointGroups[tab.id].length}</span>
            </button>
          {/if}
        {/each}
      </div>

      {#if checkpointGroups[checkpointTab].length === 0}
        <p class="hint">No {tabLabel(checkpointTab)} have written a checkpoint yet.</p>
      {:else}
        <div class="checkpoint-list">
          {#each checkpointGroups[checkpointTab] as cp (cp.name)}
            <div class="checkpoint-row" class:stale={isStale(cp)}>
              <div class="cp-line">
                <span class="cp-name">{cp.name}</span>
                <span class="cp-meta">
                  <span class="cp-position" title={positionTitle(cp.position)}>event #{cp.position.sequence}</span>
                  <span class="cp-lag" class:caught={cp.lag === 0}>{cp.lag === 0 ? 'caught up' : `${cp.lag} behind`}</span>
                  <span class="cp-updated">progressed {relativeTime(cp.updatedAt)}</span>
                  {#if cp.kind === 'projection'}
                    <button type="button" class="ghost-btn" onclick={() => toggleProjection(cp.name)}>
                      {expandedProjection === cp.name ? 'Hide read model' : 'Read model'}
                    </button>
                  {/if}
                </span>
              </div>
              {#if cp.next}
                <div class="cp-line cp-detail">next <code>{cp.next.type}</code> on <code>{cp.next.streamId}</code></div>
              {/if}
              {#if cp.eventTypes && cp.eventTypes.length > 0}
                <div class="cp-line cp-detail cp-types">
                  <span>reads</span>
                  {#each cp.eventTypes as type}
                    <span class="cp-type">{type}</span>
                  {/each}
                </div>
              {/if}
              {#if expandedProjection === cp.name}
                {#if projectionTable === null}
                  {#if loadingProjection}
                    <p class="hint">Loading…</p>
                  {/if}
                {:else}
                  <div class="projection-controls">
                    <select value={projectionFilter.column} onchange={(e) => setProjectionColumn(e.currentTarget.value)}>
                      <option value="">All columns</option>
                      {#each projectionTable.columns as column}
                        <option value={column}>{column}</option>
                      {/each}
                    </select>
                    <select
                      value={projectionFilter.operator}
                      disabled={!projectionFilter.column}
                      onchange={(e) => setProjectionOperator(e.currentTarget.value as ProjectionOperator)}
                    >
                      <option value="contains">contains</option>
                      <option value="equals">equals</option>
                      <option value="prefix">starts with</option>
                      <option value="empty">is empty</option>
                    </select>
                    {#if projectionFilter.column && projectionFilter.operator !== 'empty'}
                      <input
                        type="search"
                        placeholder="value"
                        value={projectionFilter.value}
                        oninput={(e) => setProjectionValue(e.currentTarget.value)}
                      />
                    {/if}
                  </div>
                  <p class="cp-detail">
                    {#if loadingProjection}Loading… · {/if}
                    {projectionSummary(projectionTable)}
                  </p>
                  {#if projectionTable.rows.length === 0}
                    <p class="hint">{projectionFilter.column ? 'No rows match the filter.' : 'No rows.'}</p>
                  {:else}
                    <div class="projection-table" class:loading={loadingProjection}>
                      <table>
                        <thead>
                          <tr>
                            {#each projectionTable.columns as column}
                              <th>{column}</th>
                            {/each}
                          </tr>
                        </thead>
                        <tbody>
                          {#each projectionTable.rows as row}
                            <tr>
                              {#each row as cell}
                                <td>{cell}</td>
                              {/each}
                            </tr>
                          {/each}
                        </tbody>
                      </table>
                    </div>
                  {/if}
                {/if}
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    {/if}
  {/if}
  </div>
</div>

{#snippet EventLine({ event }: { event: EventRow })}
  <span class="line-type">{event.type}</span>
  <span class="line-stream">{event.streamId}</span>
  <span class="line-meta">
    v{event.schemaVersion} · #{event.version} · {formatDate(event.createdAt)}
  </span>
  <span class="line-pos">{event.position.transactionId}:{event.position.sequence}</span>
{/snippet}

{#snippet Detail(event: EventRow)}
  <aside class="detail" aria-label="Event detail">
    <div class="detail-header">
      <div class="detail-title">
        <span class="badge">{event.type}</span>
        <span class="meta">v{event.schemaVersion} · #{event.version} · {event.position.transactionId}:{event.position.sequence}</span>
      </div>
      <div class="detail-actions">
        <button type="button" class="ghost-btn nav-btn" onclick={() => stepSelected(-1)} disabled={selectedIndex <= 0} aria-label="Previous event">‹</button>
        <button type="button" class="ghost-btn nav-btn" onclick={() => stepSelected(1)} disabled={selectedIndex >= activeList.length - 1} aria-label="Next event">›</button>
        <button type="button" class="ghost-btn" onclick={() => (selected = null)}>Close</button>
      </div>
    </div>
    <div class="detail-grid">
      <div class="kv"><span>Stream</span><code>{event.streamId}</code></div>
      <div class="kv"><span>Created</span><span>{formatDate(event.createdAt)}</span></div>
      {#if event.correlationId}
        <div class="kv"><span>Correlation</span><code>{event.correlationId}</code></div>
      {/if}
      {#if event.causationId}
        <div class="kv"><span>Causation</span><code>{event.causationId}</code></div>
      {/if}
    </div>
    {#if event.parseError}
      <div class="parse-error">{event.parseError}</div>
    {/if}
    <div class="json-section">
      <h3>Data</h3>
      <pre>{pretty(event.data)}</pre>
    </div>
    {#if event.metadata}
      <div class="json-section">
        <h3>Metadata</h3>
        <pre>{pretty(event.metadata)}</pre>
      </div>
    {/if}
    {#if event.decoded}
      <div class="json-section">
        <h3>Decoded</h3>
        <pre>{pretty(event.decoded)}</pre>
      </div>
    {/if}
  </aside>
{/snippet}

<style>
  .page {
    max-width: var(--content-width);
    margin: 0 auto;
    padding: var(--space-6);
    display: flex;
    flex-direction: column;
    gap: var(--space-5);
  }

  .page-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
  }

  .brand {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--text-lg);
    font-weight: var(--weight-semibold);
    color: var(--color-text);
  }

  .read-only {
    padding: var(--space-1) var(--space-2);
    font-size: var(--text-xs);
    font-weight: var(--weight-medium);
    color: var(--color-text-secondary);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .events {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .top-panel {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
    padding: var(--space-4);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-xs);
  }

  .view-switcher {
    display: flex;
    gap: var(--space-1);
    padding: 3px;
    background: var(--color-bg);
    border: 1px solid var(--color-border-light);
    border-radius: var(--radius-md);
    align-self: flex-start;
  }

  .segment {
    padding: var(--space-2) var(--space-4);
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    color: var(--color-text-secondary);
    background: transparent;
    border-radius: var(--radius-sm);
  }

  .segment:hover {
    color: var(--color-text);
  }

  .segment.active {
    color: var(--color-text);
    background: var(--color-surface);
    box-shadow: var(--shadow-xs);
  }

  .toolbar {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: var(--space-3);
    flex-wrap: wrap;
  }

  .ghost-btn {
    padding: var(--space-2) var(--space-3);
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    color: var(--color-text-secondary);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
  }

  .ghost-btn:hover {
    color: var(--color-text);
    background: var(--color-surface-hover);
  }

  .ghost-btn.active {
    color: var(--color-accent);
    border-color: var(--color-accent);
    background: var(--color-accent-subtle);
  }

  .ghost-btn:disabled {
    opacity: 0.5;
    cursor: default;
  }

  .filters {
    display: flex;
    align-items: flex-end;
    gap: var(--space-3);
    flex-wrap: wrap;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    font-size: var(--text-xs);
    color: var(--color-text-tertiary);
  }

  .field.grow {
    flex: 1;
    min-width: 140px;
  }

  .field select,
  .field input {
    font-size: var(--text-sm);
  }

  .select-wrap {
    position: relative;
  }

  .select-wrap select {
    appearance: none;
    -webkit-appearance: none;
    width: 100%;
    min-width: 180px;
    padding: var(--space-2) var(--space-6) var(--space-2) var(--space-3);
    font-size: var(--text-sm);
    color: var(--color-text);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-xs);
    cursor: pointer;
    transition: border-color var(--duration-fast) var(--ease-out),
                box-shadow var(--duration-fast) var(--ease-out);
  }

  .select-wrap select:focus {
    outline: none;
    border-color: var(--color-accent);
    box-shadow: 0 0 0 3px var(--color-accent-subtle);
  }

  .select-wrap .chevron {
    position: absolute;
    right: var(--space-2);
    top: 50%;
    transform: translateY(-50%);
    color: var(--color-text-tertiary);
    pointer-events: none;
  }

  .apply-btn {
    padding: var(--space-2) var(--space-4);
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    color: var(--color-surface);
    background: var(--color-accent);
    border-radius: var(--radius-sm);
  }

  .apply-btn:hover {
    background: var(--color-accent-hover);
  }

  .content {
    display: flex;
    align-items: flex-start;
    gap: var(--space-6);
  }

  .list-pane {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .section-header {
    display: flex;
    align-items: baseline;
    gap: var(--space-3);
  }

  .section-header h2 {
    margin: 0;
    font-size: var(--text-sm);
    font-weight: var(--weight-semibold);
    color: var(--color-text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .count {
    font-size: var(--text-sm);
    color: var(--color-text-tertiary);
  }

  .hint {
    font-size: var(--text-sm);
    color: var(--color-text-tertiary);
  }

  .list {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .stream-tree {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .tree-row {
    width: 100%;
  }

  .tree-toggle {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    width: 100%;
    text-align: left;
    padding: var(--space-1) var(--space-2);
    background: transparent;
    border-radius: var(--radius-sm);
    cursor: pointer;
  }

  .tree-toggle:hover {
    background: var(--color-surface-hover);
  }

  .caret {
    display: inline-block;
    width: 12px;
    font-size: var(--text-sm);
    color: var(--color-text-tertiary);
    transition: transform var(--duration-fast) var(--ease-out);
  }

  .caret.open {
    transform: rotate(90deg);
  }

  .tree-name {
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    color: var(--color-text);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .tree-name.dir {
    color: var(--color-text-secondary);
    font-weight: var(--weight-medium);
  }

  .stream-events {
    padding-top: var(--space-2);
    padding-bottom: var(--space-2);
  }

  .event-row {
    display: grid;
    grid-template-columns: auto 1fr auto auto;
    gap: var(--space-3);
    align-items: center;
    width: 100%;
    text-align: left;
    padding: var(--space-3) var(--space-4);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    font-size: var(--text-sm);
    cursor: pointer;
  }

  .event-row:hover {
    border-color: var(--color-accent);
  }

  .event-row.selected {
    border-color: var(--color-accent);
    background: var(--color-accent-subtle);
    box-shadow: 0 0 0 1px var(--color-accent);
  }

  .line-type {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    color: var(--color-accent);
    background: var(--color-accent-subtle);
    padding: 2px var(--space-2);
    border-radius: var(--radius-sm);
    white-space: nowrap;
  }

  .line-stream {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    color: var(--color-text);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .line-meta {
    font-size: var(--text-xs);
    color: var(--color-text-tertiary);
    white-space: nowrap;
  }

  .line-pos {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    color: var(--color-text-secondary);
    white-space: nowrap;
  }

  .more {
    display: flex;
    justify-content: center;
    padding: var(--space-2) 0;
  }

  .checkpoint-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .checkpoint-row {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    padding: var(--space-3) var(--space-4);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    font-size: var(--text-sm);
  }

  .cp-line {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    flex-wrap: wrap;
  }

  .checkpoint-tabs {
    display: flex;
    gap: var(--space-1);
    padding: 3px;
    background: var(--color-bg);
    border: 1px solid var(--color-border-light);
    border-radius: var(--radius-md);
    align-self: flex-start;
  }

  .checkpoint-tabs .segment {
    padding: var(--space-1) var(--space-3);
    font-size: var(--text-xs);
  }

  .tab-count {
    margin-left: var(--space-1);
    color: var(--color-text-tertiary);
  }

  .cp-detail {
    font-size: var(--text-xs);
    color: var(--color-text-tertiary);
  }

  .cp-detail code {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    color: var(--color-text-secondary);
  }

  .cp-types {
    gap: var(--space-2);
  }

  .cp-type {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    color: var(--color-text-secondary);
    background: var(--color-bg);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    padding: 1px var(--space-2);
  }

  .cp-meta {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    flex-wrap: wrap;
    margin-left: auto;
  }

  .projection-controls {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    flex-wrap: wrap;
    margin-bottom: var(--space-2);
  }

  .projection-controls select,
  .projection-controls input {
    padding: var(--space-1) var(--space-2);
    font-size: var(--text-xs);
    color: var(--color-text);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-xs);
  }

  .projection-controls select {
    min-width: 120px;
    cursor: pointer;
  }

  .projection-controls input {
    min-width: 160px;
  }

  .projection-controls select:focus,
  .projection-controls input:focus {
    outline: none;
    border-color: var(--color-accent);
    box-shadow: 0 0 0 3px var(--color-accent-subtle);
  }

  .projection-table {
    max-height: 60vh;
    overflow: auto;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    transition: opacity var(--duration-fast) var(--ease-out);
  }

  .projection-table.loading {
    opacity: 0.6;
  }

  .projection-table table {
    width: 100%;
    border-collapse: collapse;
    font-size: var(--text-xs);
  }

  .projection-table th,
  .projection-table td {
    padding: var(--space-2) var(--space-3);
    text-align: left;
    white-space: nowrap;
    border-bottom: 1px solid var(--color-border);
  }

  .projection-table th {
    position: sticky;
    top: 0;
    font-weight: var(--weight-medium);
    color: var(--color-text-secondary);
    background: var(--color-bg);
  }

  .projection-table td {
    font-family: var(--font-mono);
    color: var(--color-text);
  }

  .projection-table tbody tr:last-child td {
    border-bottom: none;
  }

  .cp-name {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
  }

  .cp-position {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    color: var(--color-text-secondary);
  }

  .cp-lag {
    font-size: var(--text-xs);
    color: var(--color-warning);
  }

  .cp-lag.caught {
    color: var(--color-success);
  }

  .cp-updated {
    font-size: var(--text-xs);
    color: var(--color-text-tertiary);
    white-space: nowrap;
  }

  .checkpoint-row.stale {
    opacity: 0.5;
  }

  .detail {
    position: sticky;
    top: var(--space-6);
    flex-shrink: 0;
    width: 400px;
    max-height: calc(100dvh - 3 * var(--space-6));
    overflow-y: auto;
    padding: var(--space-5);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-md);
  }

  .detail-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--space-4);
    margin-bottom: var(--space-6);
  }

  .detail-title {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    gap: var(--space-2);
  }

  .detail-actions {
    display: flex;
    gap: var(--space-1);
    flex-shrink: 0;
  }

  .nav-btn {
    width: 32px;
    padding: var(--space-1) 0;
    font-size: var(--text-lg);
    line-height: 1;
    text-align: center;
  }

  .badge {
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    color: var(--color-accent);
    background: var(--color-accent-subtle);
    padding: var(--space-1) var(--space-2);
    border-radius: var(--radius-sm);
  }

  .meta {
    font-size: var(--text-sm);
    color: var(--color-text-secondary);
  }

  .detail-grid {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    margin-bottom: var(--space-6);
  }

  .kv {
    display: flex;
    gap: var(--space-3);
    font-size: var(--text-sm);
  }

  .kv span:first-child {
    min-width: 96px;
    color: var(--color-text-tertiary);
  }

  .kv code {
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    word-break: break-all;
  }

  .parse-error {
    margin-bottom: var(--space-6);
    padding: var(--space-3) var(--space-4);
    font-size: var(--text-sm);
    color: var(--color-danger);
    background: var(--color-danger-subtle);
    border: 1px solid var(--color-danger);
    border-radius: var(--radius-md);
  }

  .json-section {
    margin-bottom: var(--space-6);
  }

  .json-section h3 {
    margin: 0 0 var(--space-2);
    font-size: var(--text-sm);
    font-weight: var(--weight-semibold);
    color: var(--color-text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .json-section pre {
    margin: 0;
    padding: var(--space-4);
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    line-height: 1.6;
    color: var(--color-text);
    background: var(--color-bg);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    overflow-x: auto;
  }
</style>
