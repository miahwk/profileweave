import { describe, expect, it, vi } from 'vitest'
import { ApiError, type ProfileApi } from '@/api/client'
import {
  defaultDraft,
  emptyRuntimeProvider,
  type Capabilities,
  type ConsistencyReport,
  type DoctorReport,
  type Profile,
  type Session,
  type TrashItem,
} from '@/domain/profile'
import { useProfileManager } from './useProfileManager'

const report: ConsistencyReport = { score: 92, issues: [] }
const provider = {
  id: 'test-runtime', name: 'Test runtime', description: 'Fixture', source: 'test',
  license: 'test-only', versionManagement: 'fixed', capabilities: [],
}
const capabilities: Capabilities = {
  provider,
  browsers: [{ id: 'chromium', name: 'Chromium', available: true }],
  features: [
    { key: 'profile', label: 'Profile', status: 'applied' },
    { key: 'timezone', label: 'Timezone', status: 'partial' },
  ],
}
const doctor: DoctorReport = {
  provider,
  healthy: true, inspectedBrowsers: 1, availableBrowsers: 1, activeSessions: 0,
  browsers: capabilities.browsers, issues: [],
}

function profile(id: string, name = id): Profile {
  return {
    ...defaultDraft(),
    id,
    name,
    revision: 1,
    createdAt: '2026-07-18T00:00:00Z',
    updatedAt: '2026-07-18T00:00:00Z',
  }
}

function trashItem(item: Profile): TrashItem {
  return { profile: item, deletedAt: '2026-07-18T01:00:00Z', hasBrowserData: true }
}

function fakeApi(overrides: Partial<ProfileApi> = {}): ProfileApi {
  return {
    listProfiles: vi.fn(async () => []),
    createProfile: vi.fn(async (draft) => ({ ...profile('created'), ...draft })),
    updateProfile: vi.fn(async (id, draft) => ({ ...profile(id), ...draft, revision: 2 })),
    deleteProfile: vi.fn(async () => undefined),
    listTrash: vi.fn(async () => []),
    restoreTrash: vi.fn(async (id) => profile(id)),
    purgeTrash: vi.fn(async () => undefined),
    duplicateProfile: vi.fn(async (id) => profile(`${id}_copy`)),
    validateProfile: vi.fn(async () => report),
    listSessions: vi.fn(async () => []),
    launchProfile: vi.fn(async (id) => ({ profileId: id, status: 'running' as const })),
    stopProfile: vi.fn(async (id) => ({ profileId: id, status: 'stopped' as const })),
    getCapabilities: vi.fn(async () => ({ provider: emptyRuntimeProvider(), browsers: [], features: [] })),
    getDoctor: vi.fn(async () => doctor),
    shutdown: vi.fn(async () => undefined),
    ...overrides,
  }
}

describe('useProfileManager', () => {
  it('runs runtime diagnostics without blocking initial resource loading', async () => {
    const api = fakeApi({ getDoctor: vi.fn(async () => doctor) })
    const manager = useProfileManager(api)

    await manager.load()
    expect(api.getDoctor).not.toHaveBeenCalled()

    await manager.runDoctor()

    expect(manager.doctor.value).toEqual(doctor)
    expect(manager.doctorLoading.value).toBe(false)
    expect(manager.doctorError.value).toBe('')
  })

  it('does not present a stale healthy report when diagnostics fail', async () => {
    const api = fakeApi({ getDoctor: vi.fn(async () => { throw new Error('诊断不可用') }) })
    const manager = useProfileManager(api)
    manager.doctor.value = doctor

    await manager.runDoctor()

    expect(manager.doctor.value).toBeNull()
    expect(manager.doctorError.value).toBe('诊断不可用')
    expect(manager.doctorLoading.value).toBe(false)
  })

  it('loads all four resources and asynchronously validates every profile', async () => {
    const first = profile('one', 'One')
    const second = profile('two', 'Two')
    const session: Session = { profileId: first.id, status: 'running', pid: 42 }
    const deleted = trashItem(profile('deleted'))
    const api = fakeApi({
      listProfiles: vi.fn(async () => [first, second]),
      listSessions: vi.fn(async () => [session]),
      getCapabilities: vi.fn(async () => capabilities),
      listTrash: vi.fn(async () => [deleted]),
    })
    const manager = useProfileManager(api)

    await manager.load()
    await vi.waitFor(() => expect(Object.keys(manager.reports.value)).toHaveLength(2))

    expect(manager.loading.value).toBe(false)
    expect(manager.profiles.value).toEqual([first, second])
    expect(manager.sessions.value).toEqual([session])
    expect(manager.trash.value).toEqual([deleted])
    expect(manager.capabilities.value).toEqual(capabilities)
    expect(manager.featureCoverage.value).toBe(0.5)
    expect(api.validateProfile).toHaveBeenCalledTimes(2)
  })

  it('keeps fulfilled load results when another resource fails', async () => {
    const item = profile('one')
    const api = fakeApi({
      listProfiles: vi.fn(async () => [item]),
      listSessions: vi.fn(async () => { throw new Error('会话读取失败') }),
      getCapabilities: vi.fn(async () => capabilities),
    })
    const manager = useProfileManager(api)

    await manager.load()

    expect(manager.profiles.value).toEqual([item])
    expect(manager.capabilities.value).toEqual(capabilities)
    expect(manager.loadError.value).toBe('会话读取失败')
    expect(manager.loading.value).toBe(false)
  })

  it('clears create and update busy state on both success and structured failure', async () => {
    const created = profile('created', 'New Profile')
    let finishCreate!: (value: Profile) => void
    const createPending = new Promise<Profile>((resolve) => { finishCreate = resolve })
    const api = fakeApi({
      createProfile: vi.fn(() => createPending),
      updateProfile: vi.fn(async () => {
        throw new ApiError('配置无效', 'profile_invalid', 422, [{ field: 'name', message: '名称已存在' }])
      }),
    })
    const manager = useProfileManager(api)
    manager.create()
    manager.draft.value.name = 'New Profile'

    const creating = manager.save()
    expect(manager.actionIds.value.has('create')).toBe(true)
    finishCreate(created)
    await expect(creating).resolves.toBe(true)
    expect(manager.actionIds.value.has('create')).toBe(false)
    expect(manager.profiles.value[0]).toEqual(created)

    manager.edit(created)
    await expect(manager.save()).resolves.toBe(false)
    expect(manager.actionIds.value.has(created.id)).toBe(false)
    expect(manager.loadError.value).toBe('配置无效：名称已存在')
  })

  it('keeps a successful recoverable deletion when refreshing trash fails', async () => {
    const item = profile('one', 'One')
    const api = fakeApi({ listTrash: vi.fn(async () => { throw new Error('刷新失败') }) })
    const manager = useProfileManager(api)
    manager.profiles.value = [item]
    manager.reports.value = { [item.id]: report }

    await manager.remove(item)

    expect(manager.profiles.value).toEqual([])
    expect(manager.reports.value).not.toHaveProperty(item.id)
    expect(manager.notice.value).toContain('已移入回收站')
    expect(manager.loadError.value).toBe('')
    expect(manager.actionIds.value.has(item.id)).toBe(false)
  })

  it('moves restored items to profiles and purged items out of trash', async () => {
    const restored = profile('restore', 'Restore')
    const purge = profile('purge', 'Purge')
    const api = fakeApi({ restoreTrash: vi.fn(async () => restored) })
    const manager = useProfileManager(api)
    const restoreEntry = trashItem(restored)
    const purgeEntry = trashItem(purge)
    manager.trash.value = [restoreEntry, purgeEntry]

    await manager.restore(restoreEntry)
    expect(manager.profiles.value[0]).toEqual(restored)
    expect(manager.trash.value.map((item) => item.profile.id)).toEqual([purge.id])
    expect(manager.reports.value[restored.id]).toEqual(report)

    await manager.purge(purgeEntry)
    expect(manager.trash.value).toEqual([])
    expect(api.purgeTrash).toHaveBeenCalledWith(purge.id)
    expect(manager.actionIds.value.size).toBe(0)
  })

  it('replaces a profile session across launch and stop transitions', async () => {
    const item = profile('one', 'One')
    const other: Session = { profileId: 'other', status: 'running' }
    const launched: Session = { profileId: item.id, status: 'running', pid: 101 }
    const stopped: Session = { profileId: item.id, status: 'stopped', stoppedAt: '2026-07-18T02:00:00Z' }
    const api = fakeApi({
      launchProfile: vi.fn(async () => launched),
      stopProfile: vi.fn(async () => stopped),
    })
    const manager = useProfileManager(api)
    manager.sessions.value = [{ profileId: item.id, status: 'failed' }, other]

    await manager.launch(item)
    expect(manager.sessionByProfile.value[item.id]).toEqual(launched)
    expect(manager.runningCount.value).toBe(2)

    await manager.stop(item)
    expect(manager.sessionByProfile.value[item.id]).toEqual(stopped)
    expect(manager.sessions.value).toContainEqual(other)
    expect(manager.runningCount.value).toBe(1)
    expect(manager.actionIds.value.has(item.id)).toBe(false)
  })
})
