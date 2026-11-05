import { afterEach, describe, expect, it, vi } from 'vitest'
import { listEvents, listProjection, setUnauthorizedHandler } from './events'

describe('events', () => {
  afterEach(() => {
    setUnauthorizedHandler(null)
    vi.unstubAllGlobals()
  })

  it('notifies the page when the session is rejected', async () => {
    const onUnauthorized = vi.fn()
    setUnauthorizedHandler(onUnauthorized)
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('', { status: 401 })))

    await expect(listEvents()).rejects.toThrow('Your session was rejected. Sign in again.')

    expect(onUnauthorized).toHaveBeenCalledOnce()
  })

  it('treats a projection with null rows as empty', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({ successful: true, messages: [], data: { columns: ['id'], rows: null } }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        ),
      ),
    )

    const table = await listProjection('items')

    expect(table.columns).toEqual(['id'])
    expect(table.rows).toEqual([])
  })

  it('sends the projection filter as query parameters', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({ successful: true, messages: [], data: { columns: ['id'], rows: [], total: 0 } }),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      ),
    )
    vi.stubGlobal('fetch', fetchMock)

    await listProjection('items', { column: 'body', operator: 'contains', value: 'hello world' })

    const url = new URL(String(fetchMock.mock.calls[0][0]), 'http://localhost')
    expect(url.searchParams.get('name')).toBe('items')
    expect(url.searchParams.get('filterColumn')).toBe('body')
    expect(url.searchParams.get('filterOp')).toBe('contains')
    expect(url.searchParams.get('filterValue')).toBe('hello world')
  })

  it('omits the value when filtering on an empty column', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({ successful: true, messages: [], data: { columns: ['id'], rows: [], total: 0 } }),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      ),
    )
    vi.stubGlobal('fetch', fetchMock)

    await listProjection('items', { column: 'title', operator: 'empty', value: 'ignored' })

    const url = new URL(String(fetchMock.mock.calls[0][0]), 'http://localhost')
    expect(url.searchParams.get('filterOp')).toBe('empty')
    expect(url.searchParams.has('filterValue')).toBe(false)
  })

  it('omits the filter parameters when there is no column', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({ successful: true, messages: [], data: { columns: ['id'], rows: [], total: 0 } }),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      ),
    )
    vi.stubGlobal('fetch', fetchMock)

    await listProjection('items')

    const url = new URL(String(fetchMock.mock.calls[0][0]), 'http://localhost')
    expect(url.searchParams.has('filterColumn')).toBe(false)
    expect(url.searchParams.has('filterOp')).toBe(false)
    expect(url.searchParams.has('filterValue')).toBe(false)
  })
})
