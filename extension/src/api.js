import { apiUrl } from './settings.js'
import { getToken } from './storage.js'

export class ApiError extends Error {
  constructor(message, status) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

export class ConflictError extends ApiError {
  constructor(message) {
    super(message, 409)
    this.name = 'ConflictError'
  }
}

async function parseEnvelope(response) {
  let envelope
  try {
    envelope = await response.json()
  } catch {
    throw new ApiError('request failed', response.status)
  }
  if (envelope.successful !== true) {
    const messages = envelope.messages ?? []
    const err = new ApiError(messages.length > 0 ? messages.join('\n') : 'request failed', response.status)
    if (response.status === 409) {
      const conflict = new ConflictError(err.message)
      throw conflict
    }
    throw err
  }
  return envelope
}

async function bearer() {
  const token = await getToken()
  return token ? { Authorization: `Bearer ${token}` } : {}
}

async function postJson(path, body) {
  const response = await fetch(await apiUrl(path), {
    method: 'POST',
    headers: { ...(await bearer()), 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  return parseEnvelope(response)
}

export async function saveText(text) {
  await postJson('/items', { body: text })
}

export async function saveLink(url) {
  await postJson('/items', { body: url, type: 'link' })
}

export async function saveImage(blob, filename) {
  const form = new FormData()
  form.append('file', blob, filename)
  const response = await fetch(await apiUrl('/items'), {
    method: 'POST',
    headers: await bearer(),
    body: form,
  })
  await parseEnvelope(response)
}

export async function saveNote(markdown, images = []) {
  if (images.length === 0) {
    await saveText(markdown)
    return
  }

  const form = new FormData()
  form.append('markdown', markdown)
  for (const image of images) {
    form.append('images', image.blob, image.name)
  }
  const response = await fetch(await apiUrl('/items'), {
    method: 'POST',
    headers: await bearer(),
    body: form,
  })
  await parseEnvelope(response)
}

export async function listItems(states = []) {
  const params = new URLSearchParams()
  for (const s of states) params.append('state', s)
  const qs = params.toString()
  const response = await fetch(await apiUrl(`/items${qs ? `?${qs}` : ''}`), {
    headers: await bearer(),
  })
  const envelope = await parseEnvelope(response)
  return envelope.data ?? []
}

export async function getItem(id) {
  const response = await fetch(await apiUrl(`/items/${id}`), {
    headers: await bearer(),
  })
  const envelope = await parseEnvelope(response)
  return envelope.data
}

export async function editNote(id, markdown, version, images = []) {
  if (images.length === 0) {
    const response = await fetch(await apiUrl(`/items/${id}`), {
      method: 'PATCH',
      headers: { ...(await bearer()), 'Content-Type': 'application/json' },
      body: JSON.stringify({ body: markdown, version }),
    })
    await parseEnvelope(response)
    return
  }

  const form = new FormData()
  form.append('markdown', markdown)
  form.append('version', String(version))
  for (const image of images) {
    form.append('images', image.blob, image.name)
  }
  const response = await fetch(await apiUrl(`/items/${id}`), {
    method: 'PATCH',
    headers: await bearer(),
    body: form,
  })
  await parseEnvelope(response)
}
