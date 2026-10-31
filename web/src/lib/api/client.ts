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

  const response = await fetch(`${apiBaseUrl}${path}`, {
    method: 'POST',
    headers,
    body: form,
  })

  const envelope = (await response.json()) as Envelope
  if (envelope.successful !== true) {
    throw new ApiError(envelope.messages.join('\n') || 'request failed', response.status)
  }
  return envelope
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
  return `${apiBaseUrl}/files/${path}`
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
