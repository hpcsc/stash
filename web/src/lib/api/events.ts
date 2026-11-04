export interface Position {
  transactionId: number
  sequence: number
}

export interface EventRow {
  id: string
  streamId: string
  version: number
  position: Position
  type: string
  schemaVersion: number
  createdAt: string
  data: unknown
  metadata?: unknown
  decoded?: unknown
  parseError?: string
  correlationId?: string
  causationId?: string
}

export interface Page {
  events: EventRow[]
  next?: Position
}

export interface StreamSummary {
  streamId: string
  eventCount: number
  lastVersion: number
  lastCreatedAt: string
}

export interface Checkpoint {
  name: string
  position: Position
  updatedAt: string
  lag: number
}

export interface Checkpoints {
  lastPosition?: Position
  checkpoints: Checkpoint[]
}

interface Envelope {
  successful: boolean
  messages: string[]
  data?: unknown
}

let onUnauthorized: (() => void) | null = null

export function setUnauthorizedHandler(handler: (() => void) | null): void {
  onUnauthorized = handler
}

async function getJson(path: string): Promise<Envelope> {
  const res = await fetch(`/api${path}`)
  if (res.status === 401) {
    onUnauthorized?.()
    throw new Error('Your session was rejected. Sign in again.')
  }

  const envelope = (await res.json()) as Envelope
  if (!envelope.successful) throw new Error(envelope.messages.join('\n') || 'Request failed.')
  return envelope
}

export interface ListFilter {
  after?: Position
  types?: string[]
  streamPrefix?: string
  correlationId?: string
  since?: string
  until?: string
  limit?: number
}

export async function listEvents(filter: ListFilter = {}): Promise<Page> {
  const params = new URLSearchParams()
  if (filter.after) {
    params.set('afterTxid', String(filter.after.transactionId))
    params.set('afterSeq', String(filter.after.sequence))
  }
  filter.types?.forEach((t) => params.append('type', t))
  if (filter.streamPrefix) params.set('streamPrefix', filter.streamPrefix)
  if (filter.correlationId) params.set('correlationId', filter.correlationId)
  if (filter.since) params.set('since', filter.since)
  if (filter.until) params.set('until', filter.until)
  if (filter.limit) params.set('limit', String(filter.limit))
  const qs = params.toString()
  const envelope = await getJson(`/events${qs ? `?${qs}` : ''}`)
  return envelope.data as Page
}

export async function listTypes(): Promise<string[]> {
  const envelope = await getJson('/events/types')
  return envelope.data as string[]
}

export async function listStreams(): Promise<StreamSummary[]> {
  const envelope = await getJson('/events/streams')
  return envelope.data as StreamSummary[]
}

export async function streamEvents(streamId: string): Promise<EventRow[]> {
  const envelope = await getJson(`/events/stream?streamId=${encodeURIComponent(streamId)}`)
  return envelope.data as EventRow[]
}

export async function listCheckpoints(): Promise<Checkpoints> {
  const envelope = await getJson('/events/checkpoints')
  return envelope.data as Checkpoints
}

export function openTail(onEvent: (event: EventRow) => void): () => void {
  const source = new EventSource('/api/events/tail')
  source.onmessage = (event) => {
    try {
      onEvent(JSON.parse(event.data) as EventRow)
    } catch {
      // ignore malformed lines
    }
  }
  return () => source.close()
}
