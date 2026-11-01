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
let onUnauthorized: (() => void) | null = null

const REQUEST_TIMEOUT_MS = 15_000
const UPLOAD_TIMEOUT_MS = 30_000

export function setToken(value: string | null): void {
  token = value
}

export function setUnauthorizedHandler(handler: (() => void) | null): void {
  onUnauthorized = handler
}

function toApiError(e: unknown): ApiError {
  if (e instanceof ApiError) return e
  if (e instanceof DOMException && (e.name === 'AbortError' || e.name === 'TimeoutError')) {
    return new ApiError('The server took too long to respond. Try again.')
  }
  if (e instanceof TypeError) {
    return new ApiError("Can't reach the server. Check your connection and try again.")
  }
  if (e instanceof Error) return new ApiError(e.message)
  return new ApiError('Something went wrong. Try again.')
}

async function exchange(
  method: string,
  path: string,
  body: BodyInit | undefined,
  headers: Record<string, string>,
  timeoutMs: number,
): Promise<Envelope> {
  const authed = token !== null
  let response: Response
  try {
    response = await fetch(`${apiBaseUrl}${path}`, {
      method,
      headers,
      body,
      signal: AbortSignal.timeout(timeoutMs),
    })
  } catch (e) {
    throw toApiError(e)
  }

  const text = await response.text()
  let envelope: Envelope
  try {
    envelope = JSON.parse(text) as Envelope
  } catch {
    if (authed && response.status === 401) {
      onUnauthorized?.()
      throw new ApiError('Your session has expired. Please sign in again.', 401)
    }
    throw new ApiError('The server returned an unexpected response. Try again.', response.status)
  }

  if (authed && response.status === 401) {
    onUnauthorized?.()
    throw new ApiError('Your session has expired. Please sign in again.', 401)
  }
  if (envelope.successful !== true) {
    throw new ApiError((envelope.messages ?? []).join('\n') || 'The request failed. Try again.', response.status)
  }
  return envelope
}

async function request(method: string, path: string, body?: object): Promise<Envelope> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (token) headers.Authorization = `Bearer ${token}`
  return exchange(method, path, body === undefined ? undefined : JSON.stringify(body), headers, REQUEST_TIMEOUT_MS)
}

export async function signIn(idToken: string): Promise<string> {
  const envelope = await request('POST', '/auth/signin', { provider: 'google', idToken })
  return (envelope.data as { token: string }).token
}

export async function signOut(): Promise<void> {
  await request('POST', '/auth/signout')
}

export interface ItemImage {
  id: string
  filePath: string
  thumbnailPath?: string
  fileName?: string
  fileSize?: number
  mimeType?: string
}

export interface ItemBlock {
  type: string
  text?: string
  image?: ItemImage
}

export interface Item {
  id: string
  type: string
  body: string
  state?: string
  sourceApp?: string
  sourceUrl?: string
  title?: string
  description?: string
  filePath?: string
  thumbnailPath?: string
  fileName?: string
  fileSize?: number
  mimeType?: string
  capturedAt: string
  images?: ItemImage[]
  blocks?: ItemBlock[]
}

export interface NoteDocumentBlock {
  type: 'text' | 'image'
  text?: string
  index?: number
}

export interface NoteDocument {
  blocks: NoteDocumentBlock[]
}

export async function listItems(search?: string, states?: string[]): Promise<Item[]> {
  const params = new URLSearchParams()
  if (search) params.set('q', search)
  states?.forEach((s) => params.append('state', s))
  const qs = params.toString()
  const envelope = await request('GET', `/items${qs ? `?${qs}` : ''}`)
  return envelope.data as Item[]
}

export interface Captured {
  id: string
  duplicate: boolean
  duplicateState?: string
}

export async function captureText(body: string): Promise<Captured> {
  const envelope = await request('POST', '/items', { body })
  const data = envelope.data as { id: string; duplicate?: boolean; duplicateState?: string }
  return { id: data.id, duplicate: data.duplicate ?? false, duplicateState: data.duplicateState }
}

export async function captureLink(url: string): Promise<Captured> {
  const envelope = await request('POST', '/items', { body: url, type: 'link' })
  const data = envelope.data as { id: string; duplicate?: boolean; duplicateState?: string }
  return { id: data.id, duplicate: data.duplicate ?? false, duplicateState: data.duplicateState }
}

export interface CapturedAttachment {
  id: string
  type: string
  filePath: string
  fileName?: string
}

async function postForm(path: string, form: FormData): Promise<Envelope> {
  const headers: Record<string, string> = {}
  if (token) headers.Authorization = `Bearer ${token}`
  return exchange('POST', path, form, headers, UPLOAD_TIMEOUT_MS)
}

export async function captureAttachment(file: File): Promise<CapturedAttachment> {
  const form = new FormData()
  form.append('file', file)
  const envelope = await postForm('/items', form)
  return envelope.data as CapturedAttachment
}

export async function captureNote(document: NoteDocument, images: File[]): Promise<Item> {
  const form = new FormData()
  form.append('document', JSON.stringify(document))
  for (const image of images) form.append('images', image, image.name)
  const envelope = await postForm('/items', form)
  return envelope.data as Item
}

export function fileUrl(path: string): string {
  const url = `${apiBaseUrl}/files/${path}`
  if (!token) return url
  return `${url}${url.includes('?') ? '&' : '?'}token=${encodeURIComponent(token)}`
}

export async function keepItem(id: string, tags?: string[]): Promise<void> {
  await request('POST', `/items/${id}/keep`, { tags })
}

export async function dropItem(id: string): Promise<void> {
  await request('POST', `/items/${id}/drop`)
}

export async function restoreItem(id: string): Promise<void> {
  await request('POST', `/items/${id}/restore`)
}

export async function deleteItem(id: string): Promise<void> {
  await request('POST', `/items/${id}/delete`)
}

export async function emptyTrash(): Promise<void> {
  await request('POST', '/trash/empty')
}

export interface Tag {
  id: string
  name: string
}

export async function listTags(): Promise<Tag[]> {
  const envelope = await request('GET', '/tags')
  return envelope.data as Tag[]
}

export async function createTag(name: string): Promise<string> {
  const envelope = await request('POST', '/tags', { name })
  return (envelope.data as { id: string }).id
}
