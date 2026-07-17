import type { Fingerprint, ProfileDraft } from './profile'

type DesktopTemplateFingerprint = Omit<Fingerprint, 'webrtcPolicy'>

export interface DesktopTemplate {
  id: 'zh-cn-desktop' | 'en-us-desktop' | 'de-de-desktop'
  label: string
  summary: string
  fingerprint: DesktopTemplateFingerprint
}

export const desktopTemplates: readonly DesktopTemplate[] = [
  {
    id: 'zh-cn-desktop',
    label: '中文桌面',
    summary: '简体中文、上海时区、常见全高清窗口。',
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
    },
  },
  {
    id: 'en-us-desktop',
    label: '英文桌面',
    summary: '美式英语、纽约时区、常见全高清窗口。',
    fingerprint: {
      os: 'native',
      uaMode: 'native',
      userAgent: '',
      locale: 'en-US',
      languages: ['en-US', 'en'],
      timezone: 'America/New_York',
      screen: { width: 1920, height: 1080, dpr: 1 },
      cpuCores: 8,
      memoryGB: 8,
    },
  },
  {
    id: 'de-de-desktop',
    label: '欧洲桌面',
    summary: '德语、柏林时区、常见全高清窗口。',
    fingerprint: {
      os: 'native',
      uaMode: 'native',
      userAgent: '',
      locale: 'de-DE',
      languages: ['de-DE', 'de', 'en'],
      timezone: 'Europe/Berlin',
      screen: { width: 1920, height: 1080, dpr: 1 },
      cpuCores: 8,
      memoryGB: 8,
    },
  },
]

export function applyDesktopTemplate(draft: ProfileDraft, templateID: string): ProfileDraft {
  const template = desktopTemplates.find((item) => item.id === templateID)
  if (!template) throw new Error(`unknown desktop template: ${templateID}`)

  return {
    ...draft,
    browser: { ...draft.browser },
    proxy: { ...draft.proxy },
    fingerprint: {
      ...template.fingerprint,
      languages: [...template.fingerprint.languages],
      screen: { ...template.fingerprint.screen },
      webrtcPolicy: draft.fingerprint.webrtcPolicy,
    },
  }
}
