import { describe, expect, it, vi } from 'vitest'
import { ApiError } from '@/api/client'
import { useAppShutdown } from './useAppShutdown'

describe('useAppShutdown', () => {
  const health = { status: 'ok', product: 'ProfileWeave', version: 'test', commit: 'test', date: 'test' } as const

  it('shows a terminal page state only after the service becomes unreachable', async () => {
    let finish!: () => void
    const shutdown = vi.fn(() => new Promise<{ status: 'shutting_down' }>((resolve) => {
      finish = () => resolve({ status: 'shutting_down' })
    }))
    const getHealth = vi.fn()
      .mockResolvedValueOnce(health)
      .mockRejectedValueOnce(new TypeError('fetch failed'))
    const state = useAppShutdown(
      { shutdown, getHealth },
      { pollIntervalMs: 1, sleep: async () => undefined },
    )

    state.request()
    const result = state.confirm()

    expect(state.confirmOpen.value).toBe(true)
    expect(state.busy.value).toBe(true)
    finish()
    await expect(result).resolves.toBe(true)
    expect(shutdown).toHaveBeenCalledOnce()
    expect(getHealth).toHaveBeenCalledTimes(2)
    expect(state.confirmOpen.value).toBe(false)
    expect(state.stopped.value).toBe(true)
    expect(state.busy.value).toBe(false)
  })

  it('times out while the service remains online and restores the retry controls', async () => {
    let elapsed = 0
    const shutdown = vi.fn().mockResolvedValue({ status: 'shutting_down' as const })
    const getHealth = vi.fn().mockResolvedValue(health)
    const state = useAppShutdown(
      { shutdown, getHealth },
      {
        pollIntervalMs: 5,
        timeoutMs: 10,
        now: () => elapsed,
        sleep: async (milliseconds) => { elapsed += milliseconds },
      },
    )

    state.request()
    await expect(state.confirm()).resolves.toBe(false)

    expect(state.error.value).toContain('本地服务在等待时间内仍可访问')
    expect(state.confirmOpen.value).toBe(true)
    expect(state.stopped.value).toBe(false)
    expect(state.busy.value).toBe(false)
    expect(getHealth).toHaveBeenCalledTimes(3)
  })

  it('can retry after a timeout and completes once the service is offline', async () => {
    let elapsed = 0
    const shutdown = vi.fn()
      .mockResolvedValue({ status: 'shutting_down' as const })
    const getHealth = vi.fn()
      .mockResolvedValueOnce(health)
      .mockResolvedValueOnce(health)
      .mockRejectedValueOnce(new TypeError('fetch failed'))
    const state = useAppShutdown(
      { shutdown, getHealth },
      {
        pollIntervalMs: 1,
        timeoutMs: 1,
        now: () => elapsed,
        sleep: async (milliseconds) => { elapsed += milliseconds },
      },
    )

    state.request()
    await expect(state.confirm()).resolves.toBe(false)
    expect(state.error.value).toContain('本地服务在等待时间内仍可访问')
    expect(state.confirmOpen.value).toBe(true)
    expect(state.stopped.value).toBe(false)
    expect(state.busy.value).toBe(false)

    elapsed = 0
    await expect(state.confirm()).resolves.toBe(true)
    expect(shutdown).toHaveBeenCalledTimes(2)
    expect(state.error.value).toBe('')
    expect(state.stopped.value).toBe(true)
    expect(state.confirmOpen.value).toBe(false)
  })

  it('keeps the confirmation recoverable when shutdown is rejected', async () => {
    const shutdown = vi.fn().mockRejectedValue(new ApiError('仍有请求正在结束', 'shutdown_failed', 500))
    const getHealth = vi.fn()
    const state = useAppShutdown({ shutdown, getHealth })

    state.request()
    await expect(state.confirm()).resolves.toBe(false)

    expect(state.error.value).toBe('仍有请求正在结束')
    expect(state.confirmOpen.value).toBe(true)
    expect(state.stopped.value).toBe(false)
    expect(state.busy.value).toBe(false)
    expect(getHealth).not.toHaveBeenCalled()
  })
})
