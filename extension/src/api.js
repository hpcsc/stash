import { apiBaseUrl, apiPrefix } from './config.js'
import { getToken } from './storage.js'

export class ApiError extends Error {
  constructor(message, status) {
    super(message)
    this.name = 'ApiError'
    this.status = status
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
    throw new ApiError(messages.length > 0 ? messages.join('\n') : 'request failed', response.status)
  }
  return envelope
}

async function bearer() {
  const token = await getToken()
  return token ? { Authorization: `Bearer ${token}` } : {}
}

async function postJson(path, body) {
  const response = await fetch(`${apiBaseUrl}${apiPrefix}${path}`, {
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
  const response = await fetch(`${apiBaseUrl}${apiPrefix}/items`, {
    method: 'POST',
    headers: await bearer(),
    body: form,
  })
  await parseEnvelope(response)
}
