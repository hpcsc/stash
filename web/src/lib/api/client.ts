import { apiBaseUrl } from '../config'

interface Envelope {
  successful: boolean
  messages: string[]
  data?: unknown
}

export class ApiError extends Error {
  readonly status: number | undefined

  constructor(message: string, status?: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

let token: string | null = null

export function setToken(value: string | null): void {
  token = value
}

async function request(method: string, path: string, body?: object): Promise<Envelope> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (token) headers.Authorization = `Bearer ${token}`

  const response = await fetch(`${apiBaseUrl}${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  })

  const envelope = (await response.json()) as Envelope
  if (envelope.successful !== true) {
    throw new ApiError(envelope.messages.join('\n') || 'request failed', response.status)
  }
  return envelope
}

export async function signIn(idToken: string): Promise<string> {
  const envelope = await request('POST', '/auth/signin', { provider: 'google', idToken })
  return (envelope.data as { token: string }).token
}

export async function signOut(): Promise<void> {
  await request('POST', '/auth/signout')
}

export interface Item {
  id: string
  type: string
  body: string
  sourceApp?: string
  sourceUrl?: string
  capturedAt: string
}

export async function listItems(): Promise<Item[]> {
  const envelope = await request('GET', '/items')
  return envelope.data as Item[]
}

export async function captureText(body: string): Promise<string> {
  const envelope = await request('POST', '/items', { body })
  return (envelope.data as { id: string }).id
}

export async function keepItem(id: string, tags?: string[]): Promise<void> {
  await request('POST', `/items/${id}/keep`, { tags })
}

export async function dropItem(id: string): Promise<void> {
  await request('POST', `/items/${id}/drop`)
}
