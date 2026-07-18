import type {
  Capabilities,
  ConsistencyReport,
  DoctorReport,
  Profile,
  ProfileDraft,
  Session,
  TrashItem,
} from '@/domain/profile'

export interface ApiErrorDetail {
  field?: string
  message: string
}

export class ApiError extends Error {
  constructor(
    message: string,
    readonly code = 'request_failed',
    readonly status = 0,
    readonly details: ApiErrorDetail[] = [],
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

type Fetcher = typeof fetch

async function request<T>(fetcher: Fetcher, path: string, init?: RequestInit): Promise<T> {
  const response = await fetcher(path, {
    ...init,
    headers: { Accept: 'application/json', ...(init?.body ? { 'Content-Type': 'application/json' } : {}), ...init?.headers },
  })
  const data = response.status === 204 ? undefined : await response.json().catch(() => undefined)
  if (!response.ok) {
    const payload = data?.error
    throw new ApiError(payload?.message ?? `请求失败（${response.status}）`, payload?.code, response.status, payload?.details)
  }
  return data as T
}

function arrayFrom<T>(data: T[] | { items?: T[]; profiles?: T[]; sessions?: T[] }): T[] {
  if (Array.isArray(data)) return data
  return data.items ?? data.profiles ?? data.sessions ?? []
}

export function createApi(fetcher: Fetcher = fetch) {
  const base = '/api/v1'
  let tokenRequest: Promise<string> | undefined

  function getControlToken(): Promise<string> {
    if (!tokenRequest) {
      tokenRequest = request<{ controlToken: string }>(fetcher, `${base}/bootstrap`)
        .then((result) => result.controlToken)
        .catch((error) => {
          tokenRequest = undefined
          throw error
        })
    }
    return tokenRequest
  }

  async function mutate<T>(path: string, init: RequestInit): Promise<T> {
    for (let attempt = 0; attempt < 2; attempt += 1) {
      const token = await getControlToken()
      try {
        return await request<T>(fetcher, path, {
          ...init,
          headers: { ...init.headers, 'X-ProfileWeave-Token': token },
        })
      } catch (error) {
        if (!(error instanceof ApiError) || error.code !== 'control_token_invalid' || attempt > 0) throw error
        tokenRequest = undefined
      }
    }
    throw new ApiError('无法获取本地控制令牌', 'control_token_invalid', 403)
  }

  return {
    async listProfiles(): Promise<Profile[]> {
      return arrayFrom(await request<Profile[] | { items: Profile[] }>(fetcher, `${base}/profiles`))
    },
    createProfile(draft: ProfileDraft): Promise<Profile> {
      return mutate(`${base}/profiles`, { method: 'POST', body: JSON.stringify(draft) })
    },
    updateProfile(id: string, draft: ProfileDraft, revision?: number): Promise<Profile> {
      return mutate(`${base}/profiles/${encodeURIComponent(id)}`, {
        method: 'PUT', body: JSON.stringify({ ...draft, id, ...(revision === undefined ? {} : { revision }) }),
      })
    },
    deleteProfile(id: string): Promise<void> {
      return mutate(`${base}/profiles/${encodeURIComponent(id)}`, { method: 'DELETE' })
    },
    async listTrash(): Promise<TrashItem[]> {
      return arrayFrom(await request<TrashItem[] | { items: TrashItem[] }>(fetcher, `${base}/trash`))
    },
    restoreTrash(id: string): Promise<Profile> {
      return mutate(`${base}/trash/${encodeURIComponent(id)}/restore`, { method: 'POST' })
    },
    purgeTrash(id: string): Promise<void> {
      return mutate(`${base}/trash/${encodeURIComponent(id)}`, { method: 'DELETE' })
    },
    duplicateProfile(id: string): Promise<Profile> {
      return mutate(`${base}/profiles/${encodeURIComponent(id)}/duplicate`, { method: 'POST' })
    },
    validateProfile(id: string): Promise<ConsistencyReport> {
      return mutate(`${base}/profiles/${encodeURIComponent(id)}/validate`, { method: 'POST' })
    },
    async listSessions(): Promise<Session[]> {
      return arrayFrom(await request<Session[] | { items: Session[] }>(fetcher, `${base}/sessions`))
    },
    launchProfile(id: string): Promise<Session> {
      return mutate(`${base}/profiles/${encodeURIComponent(id)}/launch`, { method: 'POST' })
    },
    stopProfile(id: string): Promise<Session> {
      return mutate(`${base}/profiles/${encodeURIComponent(id)}/stop`, { method: 'POST' })
    },
    getCapabilities(): Promise<Capabilities> {
      return request(fetcher, `${base}/capabilities`)
    },
    getDoctor(): Promise<DoctorReport> {
      return request(fetcher, `${base}/doctor`)
    },
  }
}

export type ProfileApi = ReturnType<typeof createApi>
export const api = createApi()
