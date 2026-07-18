import { describe, expect, it, vi } from 'vitest'
import { ApiError, createApi } from './client'
import { defaultDraft, type DoctorReport } from '@/domain/profile'

function response(body: unknown, status = 200): Response {
  return new Response(body === undefined ? undefined : JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('profile API adapter', () => {
  it('reads the public health endpoint for lifecycle confirmation', async () => {
    const payload = { status: 'ok', product: 'ProfileWeave', version: 'test', commit: 'test', date: 'test' } as const
    const fetcher = vi.fn<typeof fetch>().mockResolvedValue(response(payload))

    await expect(createApi(fetcher).getHealth()).resolves.toEqual(payload)
    expect(fetcher).toHaveBeenCalledWith('/api/v1/health', expect.objectContaining({
      headers: expect.objectContaining({ Accept: 'application/json' }),
    }))
  })

  it('unwraps list envelopes from the HTTP contract', async () => {
    const fetcher = vi.fn<typeof fetch>().mockResolvedValue(response({ items: [{ id: 'p_one', name: 'QA' }] }))
    const profiles = await createApi(fetcher).listProfiles()

    expect(profiles).toHaveLength(1)
    expect(profiles[0]?.name).toBe('QA')
    expect(fetcher).toHaveBeenCalledWith('/api/v1/profiles', expect.objectContaining({ headers: expect.objectContaining({ Accept: 'application/json' }) }))
  })

  it('serializes a new profile through the adapter boundary', async () => {
    const draft = defaultDraft()
    draft.name = 'Shanghai QA'
    const fetcher = vi.fn<typeof fetch>()
      .mockResolvedValueOnce(response({ controlToken: 'test-token' }))
      .mockResolvedValueOnce(response({ ...draft, id: 'p_one', revision: 1 }))

    await createApi(fetcher).createProfile(draft)

    const init = fetcher.mock.calls[1]?.[1]
    expect(init?.method).toBe('POST')
    expect(init?.headers).toMatchObject({ 'X-ProfileWeave-Token': 'test-token' })
    expect(JSON.parse(String(init?.body))).toMatchObject({ name: 'Shanghai QA', browser: { kind: 'auto' } })
  })

  it('maps structured errors without leaking transport details', async () => {
    const fetcher = vi.fn<typeof fetch>()
      .mockResolvedValueOnce(response({ controlToken: 'test-token' }))
      .mockResolvedValueOnce(response({
        error: { code: 'profile_invalid', message: '配置无效', details: [{ field: 'name', message: '名称不能为空' }] },
      }, 422))

    await expect(createApi(fetcher).createProfile(defaultDraft())).rejects.toMatchObject<ApiError>({
      code: 'profile_invalid', status: 422, message: '配置无效', details: [{ field: 'name', message: '名称不能为空' }],
    })
  })

  it('supports listing, restoring, and permanently deleting recycle entries', async () => {
    const fetcher = vi.fn<typeof fetch>()
      .mockResolvedValueOnce(response({ items: [{ profile: { id: 'p_one', name: 'QA' }, deletedAt: '2026-07-18T00:00:00Z', hasBrowserData: true }] }))
      .mockResolvedValueOnce(response({ controlToken: 'test-token' }))
      .mockResolvedValueOnce(response({ id: 'p_one', name: 'QA' }))
      .mockResolvedValueOnce(response(undefined, 204))
    const api = createApi(fetcher)

    expect((await api.listTrash())[0]?.hasBrowserData).toBe(true)
    expect((await api.restoreTrash('p_one')).name).toBe('QA')
    await api.purgeTrash('p_one')

    expect(fetcher.mock.calls[2]).toEqual([
      '/api/v1/trash/p_one/restore',
      expect.objectContaining({ method: 'POST', headers: expect.objectContaining({ 'X-ProfileWeave-Token': 'test-token' }) }),
    ])
    expect(fetcher.mock.calls[3]?.[0]).toBe('/api/v1/trash/p_one')
    expect(fetcher.mock.calls[3]?.[1]?.method).toBe('DELETE')
  })

  it('reads the runtime doctor report from the public local endpoint', async () => {
    const payload = {
      provider: {
        id: 'system-chromium', name: 'System Chromium', description: 'Fixture',
        source: 'host-installed browser', license: 'browser-specific', versionManagement: 'host-managed', capabilities: [],
      }, healthy: true,
      inspectedBrowsers: 1, availableBrowsers: 1, activeSessions: 0, browsers: [], issues: [],
    } satisfies DoctorReport
    const fetcher = vi.fn<typeof fetch>().mockResolvedValue(response(payload))

    expect(await createApi(fetcher).getDoctor()).toEqual(payload)
    expect(fetcher).toHaveBeenCalledWith('/api/v1/doctor', expect.objectContaining({
      headers: expect.objectContaining({ Accept: 'application/json' }),
    }))
  })

  it('uses the local control token to request an application shutdown', async () => {
    const fetcher = vi.fn<typeof fetch>()
      .mockResolvedValueOnce(response({ controlToken: 'shutdown-token' }))
      .mockResolvedValueOnce(response({ status: 'shutting_down' }, 202))

    await expect(createApi(fetcher).shutdown()).resolves.toEqual({ status: 'shutting_down' })

    expect(fetcher.mock.calls[0]?.[0]).toBe('/api/v1/bootstrap')
    expect(fetcher.mock.calls[1]).toEqual([
      '/api/v1/shutdown',
      expect.objectContaining({
        method: 'POST',
        headers: expect.objectContaining({ 'X-ProfileWeave-Token': 'shutdown-token' }),
      }),
    ])
  })
})
