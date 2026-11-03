import { afterEach, describe, expect, it, vi } from 'vitest'
import { listEvents, setUnauthorizedHandler } from './events'

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
})
