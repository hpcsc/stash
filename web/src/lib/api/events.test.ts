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
})
