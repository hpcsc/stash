import { afterEach, describe, expect, it, vi } from 'vitest'
import { setToken, setUnauthorizedHandler, signIn, signOut, listItems, captureLink, captureNote, restoreItem, deleteItem, emptyTrash, fileUrl } from './client'

function jsonResponse(body: object, status = 200): Response {
  return new Response(JSON.stringify(body), { status })
}

describe('client', () => {
  afterEach(() => {
    setToken(null)
    setUnauthorizedHandler(null)
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

  it('captureNote posts the markdown and returns the item', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ successful: true, messages: [], data: { id: 'item-1', type: 'note', body: 'remember this', state: 'inbox', capturedAt: '2026-01-01T00:00:00Z' } }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const item = await captureNote('remember this')

    expect(item.id).toBe('item-1')
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://127.0.0.1:3333/items')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body)).toEqual({ body: 'remember this' })
  })

  it('captureLink returns the duplicate info when the URL is already saved', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({
        successful: true,
        messages: [],
        data: { id: 'item-1', duplicate: true, duplicateState: 'kept' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const captured = await captureLink('https://example.com')

    expect(captured).toEqual({ id: 'item-1', duplicate: true, duplicateState: 'kept' })
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://127.0.0.1:3333/items')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body)).toEqual({ body: 'https://example.com', type: 'link' })
  })

  it('captureNote posts markdown and images as multipart', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({
        successful: true,
        messages: [],
        data: { id: 'item-1', type: 'note', body: '![](@stash-image-0)', state: 'inbox', capturedAt: '2026-01-01T00:00:00Z' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)
    const image = new File(['png'], 'photo.png', { type: 'image/png' })

    const item = await captureNote('![](@stash-image-0)', [image])

    expect(item.type).toBe('note')
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://127.0.0.1:3333/items')
    expect(init.body).toBeInstanceOf(FormData)
    const form = init.body as FormData
    expect(form.get('markdown')).toBe('![](@stash-image-0)')
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

  it('fileUrl appends the session token', () => {
    setToken('session-token')
    expect(fileUrl('user/1/files/a.png')).toBe('http://127.0.0.1:3333/files/user/1/files/a.png?token=session-token')
  })

  it('fileUrl omits the token when signed out', () => {
    expect(fileUrl('user/1/files/a.png')).toBe('http://127.0.0.1:3333/files/user/1/files/a.png')
  })

  it('maps a network failure to a friendly message', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))

    await expect(listItems()).rejects.toThrow("Can't reach the server. Check your connection and try again.")
  })

  it('maps a timeout to a friendly message', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new DOMException('timed out', 'TimeoutError')))

    await expect(listItems()).rejects.toThrow('The server took too long to respond. Try again.')
  })

  it('maps a non-JSON response to a friendly message', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('<html>502 Bad Gateway</html>', { status: 502 })))

    await expect(listItems()).rejects.toThrow('The server returned an unexpected response. Try again.')
  })

  it('signs out on a 401 to an authenticated request', async () => {
    setToken('session-token')
    const onUnauthorized = vi.fn()
    setUnauthorizedHandler(onUnauthorized)
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ successful: false, messages: ['expired'] }, 401)))

    await expect(listItems()).rejects.toThrow('Your session has expired. Please sign in again.')
    expect(onUnauthorized).toHaveBeenCalledOnce()
  })
})
