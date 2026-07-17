import { computed, ref } from 'vue'
import { ApiError, api as defaultApi, type ProfileApi } from '@/api/client'
import { evaluateDraft } from '@/domain/evaluate'
import { defaultDraft, toDraft, type Capabilities, type ConsistencyReport, type Profile, type ProfileDraft, type Session } from '@/domain/profile'

function messageFrom(error: unknown): string {
  if (error instanceof ApiError) {
    const detail = error.details?.[0]
    return detail ? `${error.message}：${detail.message}` : error.message
  }
  return error instanceof Error ? error.message : '发生未知错误，请稍后重试'
}

export function useProfileManager(service: ProfileApi = defaultApi) {
  const profiles = ref<Profile[]>([])
  const sessions = ref<Session[]>([])
  const capabilities = ref<Capabilities>({ browsers: [], features: [] })
  const reports = ref<Record<string, ConsistencyReport>>({})
  const search = ref('')
  const loading = ref(true)
  const loadError = ref('')
  const notice = ref('')
  const editorOpen = ref(false)
  const editingProfile = ref<Profile | null>(null)
  const draft = ref<ProfileDraft>(defaultDraft())
  const actionIds = ref(new Set<string>())

  const filteredProfiles = computed(() => {
    const query = search.value.trim().toLocaleLowerCase()
    if (!query) return profiles.value
    return profiles.value.filter((profile) =>
      [profile.name, profile.notes, profile.browser.kind, profile.fingerprint.locale, ...profile.tags]
        .join(' ').toLocaleLowerCase().includes(query),
    )
  })
  const sessionByProfile = computed<Record<string, Session>>(() => Object.fromEntries(
    sessions.value.map((session) => [session.profileId, session]),
  ))
  const runningCount = computed(() => sessions.value.filter((session) => session.status === 'running' || session.status === 'starting').length)
  const featureCoverage = computed(() => {
    const all = capabilities.value.features
    return all.length ? all.filter((feature) => feature.status === 'applied').length / all.length : 0
  })
  const draftReport = computed(() => evaluateDraft(draft.value))
  const draftHasErrors = computed(() => draftReport.value.issues.some((item) => item.severity === 'error'))

  function setBusy(id: string, busy: boolean) {
    const next = new Set(actionIds.value)
    busy ? next.add(id) : next.delete(id)
    actionIds.value = next
  }
  function showNotice(message: string) {
    notice.value = message
    globalThis.setTimeout(() => { if (notice.value === message) notice.value = '' }, 4000)
  }
  async function loadReports(items: Profile[]) {
    const settled = await Promise.allSettled(items.map(async (profile) => ({ id: profile.id, report: await service.validateProfile(profile.id) })))
    const next = { ...reports.value }
    for (const result of settled) if (result.status === 'fulfilled') next[result.value.id] = result.value.report
    reports.value = next
  }
  async function load() {
    loading.value = true
    loadError.value = ''
    const [profileResult, sessionResult, capabilityResult] = await Promise.allSettled([
      service.listProfiles(), service.listSessions(), service.getCapabilities(),
    ])
    if (profileResult.status === 'fulfilled') profiles.value = profileResult.value
    else loadError.value = messageFrom(profileResult.reason)
    if (sessionResult.status === 'fulfilled') sessions.value = sessionResult.value
    else loadError.value ||= messageFrom(sessionResult.reason)
    if (capabilityResult.status === 'fulfilled') capabilities.value = capabilityResult.value
    else loadError.value ||= messageFrom(capabilityResult.reason)
    loading.value = false
    if (profileResult.status === 'fulfilled') void loadReports(profileResult.value)
  }
  async function refreshSessions() {
    try {
      sessions.value = await service.listSessions()
    } catch {
      // Background refresh failures stay quiet; the next user action still surfaces its own error.
    }
  }
  function create() {
    editingProfile.value = null
    draft.value = defaultDraft()
    editorOpen.value = true
  }
  function edit(profile: Profile) {
    editingProfile.value = profile
    draft.value = toDraft(profile)
    editorOpen.value = true
  }
  function closeEditor() {
    editorOpen.value = false
  }
  async function save(): Promise<boolean> {
    if (draftHasErrors.value) return false
    const id = editingProfile.value?.id ?? 'create'
    setBusy(id, true)
    try {
      const saved = editingProfile.value
        ? await service.updateProfile(editingProfile.value.id, draft.value, editingProfile.value.revision)
        : await service.createProfile(draft.value)
      const index = profiles.value.findIndex((item) => item.id === saved.id)
      if (index >= 0) profiles.value.splice(index, 1, saved)
      else profiles.value.unshift(saved)
      reports.value[saved.id] = await service.validateProfile(saved.id)
      closeEditor()
      showNotice(editingProfile.value ? 'Profile 已更新' : '隔离 Profile 已创建')
      return true
    } catch (error) {
      loadError.value = messageFrom(error)
      return false
    } finally {
      setBusy(id, false)
    }
  }
  async function duplicate(profile: Profile) {
    setBusy(profile.id, true)
    try {
      const copy = await service.duplicateProfile(profile.id)
      profiles.value.unshift(copy)
      reports.value[copy.id] = await service.validateProfile(copy.id)
      showNotice(`已复制“${profile.name}”，运行状态未复制`)
    } catch (error) { loadError.value = messageFrom(error) }
    finally { setBusy(profile.id, false) }
  }
  async function remove(profile: Profile) {
    setBusy(profile.id, true)
    try {
      await service.deleteProfile(profile.id)
      profiles.value = profiles.value.filter((item) => item.id !== profile.id)
      showNotice(`已删除“${profile.name}”`)
    } catch (error) { loadError.value = messageFrom(error) }
    finally { setBusy(profile.id, false) }
  }
  async function launch(profile: Profile) {
    setBusy(profile.id, true)
    try {
      const session = await service.launchProfile(profile.id)
      sessions.value = sessions.value.filter((item) => item.profileId !== profile.id).concat(session)
      showNotice(`正在启动“${profile.name}”`)
    } catch (error) { loadError.value = messageFrom(error) }
    finally { setBusy(profile.id, false) }
  }
  async function stop(profile: Profile) {
    setBusy(profile.id, true)
    try {
      const session = await service.stopProfile(profile.id)
      sessions.value = sessions.value.filter((item) => item.profileId !== profile.id).concat(session)
      showNotice(`已停止“${profile.name}”`)
    } catch (error) { loadError.value = messageFrom(error) }
    finally { setBusy(profile.id, false) }
  }

  return {
    profiles, sessions, capabilities, reports, search, loading, loadError, notice, editorOpen, editingProfile,
    draft, actionIds, filteredProfiles, sessionByProfile, runningCount, featureCoverage, draftReport, draftHasErrors,
    load, refreshSessions, create, edit, closeEditor, save, duplicate, remove, launch, stop,
  }
}
