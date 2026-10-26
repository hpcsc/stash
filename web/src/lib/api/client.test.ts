import { afterEach, describe, expect, it, vi } from 'vitest'
import { setToken, signIn, signOut, listItems, captureText, captureNote, restoreItem, deleteItem, emptyTrash } from './client'

function jsonResponse(body: object, status = 200): Response {
  return new Response(JSON.stringify(body), { status })
}

describe('client', () => {
  afterEach(() => {
    setToken(null)
    vi.unstubAllGlobals()
  })

  it('signIn returns the session token', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ successful: true, messages: [], data: { token: 'session-token' } }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const token = await signIn('google-token')

    expect(token).toBe('session-token')
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://127.0.0.1:3333/auth/signin')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body)).toEqual({ provider: 'google', idToken: 'google-token' })
  })

  it('signIn throws the backend message on failure', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse({ successful: false, messages: ['sign-in failed'] }, 401)),
    )

    await expect(signIn('bad-token')).rejects.toThrow('sign-in failed')
  })

  it('signOut sends the bearer token', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ successful: true, messages: [] }))
    vi.stubGlobal('fetch', fetchMock)
    setToken('session-token')

    await signOut()

    const [, init] = fetchMock.mock.calls[0]
    expect(init.headers.Authorization).toBe('Bearer session-token')
  })

  it('listItems returns the items', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({
        successful: true,
        messages: [],
        data: [{ id: '1', type: 'text', body: 'hello', state: 'inbox', capturedAt: '2026-01-01T00:00:00Z' }],
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const items = await listItems()

    expect(items).toHaveLength(1)
    expect(items[0].body).toBe('hello')
  })

  it('listItems passes search and state query params', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ successful: true, messages: [], data: [] }),
    )
    vi.stubGlobal('fetch', fetchMock)

    await listItems('hello', ['inbox'])

    const [url] = fetchMock.mock.calls[0]
    expect(url).toContain('q=hello')
    expect(url).toContain('state=inbox')
  })

  it('captureText posts the body and returns the id', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ successful: true, messages: [], data: { id: 'item-1' } }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const id = await captureText('remember this')

    expect(id).toBe('item-1')
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://127.0.0.1:3333/items')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body)).toEqual({ body: 'remember this' })
  })

  it('captureNote posts the document and images as multipart', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({
        successful: true,
        messages: [],
        data: { id: 'item-1', type: 'note', body: 'before', state: 'inbox', capturedAt: '2026-01-01T00:00:00Z' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)
    const image = new File(['png'], 'photo.png', { type: 'image/png' })
    const document = {
      blocks: [
        { type: 'text' as const, text: 'before' },
        { type: 'image' as const, index: 0 },
      ],
    }

    const item = await captureNote(document, [image])

    expect(item.type).toBe('note')
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://127.0.0.1:3333/items')
    expect(init.body).toBeInstanceOf(FormData)
    const form = init.body as FormData
    expect(form.get('document')).toBe(JSON.stringify(document))
    expect(form.get('images')).toBeInstanceOf(File)
  })

  it('restoreItem posts to the restore endpoint', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ successful: true, messages: [] }))
    vi.stubGlobal('fetch', fetchMock)

    await restoreItem('item-1')

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://127.0.0.1:3333/items/item-1/restore')
    expect(init.method).toBe('POST')
  })

  it('deleteItem posts to the delete endpoint', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ successful: true, messages: [] }))
    vi.stubGlobal('fetch', fetchMock)

    await deleteItem('item-1')

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://127.0.0.1:3333/items/item-1/delete')
    expect(init.method).toBe('POST')
  })

  it('emptyTrash posts to the trash endpoint', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ successful: true, messages: [] }))
    vi.stubGlobal('fetch', fetchMock)

    await emptyTrash()

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://127.0.0.1:3333/trash/empty')
    expect(init.method).toBe('POST')
  })
})
