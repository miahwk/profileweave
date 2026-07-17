export type OperatingSystem = 'native' | 'windows' | 'macos' | 'linux'
export type UAMode = 'native' | 'custom'
export type ProxyMode = 'direct' | 'http' | 'socks5'
export type WebRTCPolicy = 'native' | 'proxy_only'
export type CapabilityStatus = 'applied' | 'partial' | 'unsupported'
export type Severity = 'error' | 'warning' | 'info'
export type SessionStatus = 'starting' | 'running' | 'stopping' | 'stopped' | 'failed'

export interface Fingerprint {
  os: OperatingSystem
  uaMode: UAMode
  userAgent: string
  locale: string
  languages: string[]
  timezone: string
  screen: { width: number; height: number; dpr: number }
  cpuCores: number
  memoryGB: number
  webrtcPolicy: WebRTCPolicy
}

export interface ProxyConfig {
  mode: ProxyMode
  host: string
  port: number
}

export interface BrowserSelection {
  kind: string
  customPath?: string
}

export interface ProfileDraft {
  name: string
  notes: string
  tags: string[]
  startURL: string
  browser: BrowserSelection
  fingerprint: Fingerprint
  proxy: ProxyConfig
}

export interface Profile extends ProfileDraft {
  id: string
  revision: number
  createdAt: string
  updatedAt: string
}

export interface Session {
  profileId: string
  status: SessionStatus
  pid?: number
  startedAt?: string
  stoppedAt?: string
  lastError?: string
}

export interface ReportIssue {
  severity: Severity
  code: string
  field?: string
  message: string
}

export interface ConsistencyReport {
  score: number
  issues: ReportIssue[]
}

export interface BrowserCapability {
  id: string
  name: string
  path?: string
  available: boolean
}

export interface RuntimeCapability {
  key: string
  label: string
  status: CapabilityStatus
  detail?: string
}

export interface Capabilities {
  browsers: BrowserCapability[]
  features: RuntimeCapability[]
}

export const defaultDraft = (): ProfileDraft => ({
  name: '',
  notes: '',
  tags: [],
  startURL: 'https://example.com',
  browser: { kind: 'auto' },
  fingerprint: {
    os: 'native',
    uaMode: 'native',
    userAgent: '',
    locale: 'zh-CN',
    languages: ['zh-CN', 'zh', 'en-US'],
    timezone: 'Asia/Shanghai',
    screen: { width: 1920, height: 1080, dpr: 1 },
    cpuCores: 8,
    memoryGB: 8,
    webrtcPolicy: 'proxy_only',
  },
  proxy: { mode: 'direct', host: '', port: 0 },
})

export function toDraft(profile: Profile): ProfileDraft {
  const { id: _id, revision: _revision, createdAt: _createdAt, updatedAt: _updatedAt, ...draft } = profile
  return structuredClone(draft)
}

export function emptyReport(): ConsistencyReport {
  return { score: 100, issues: [] }
}
