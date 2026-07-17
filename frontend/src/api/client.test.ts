import { describe, expect, it, vi } from 'vitest'
import { ApiError, createApi } from './client'
import { defaultDraft } from '@/domain/profile'

function response(body: unknown, status = 200): Response {
  return new Response(body === undefined ? undefined : JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('profile API adapter', () => {
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
})
