import { describe, expect, it } from 'vitest'
import { applyDesktopTemplate, desktopTemplates } from './desktopTemplates'
import { defaultDraft } from './profile'

describe('desktopTemplates', () => {
  it('provides deterministic Chinese, English, and European starting points', () => {
    expect(desktopTemplates.map((template) => template.id)).toEqual([
      'zh-cn-desktop',
      'en-us-desktop',
      'de-de-desktop',
    ])
    expect(desktopTemplates.map((template) => template.fingerprint.timezone)).toEqual([
      'Asia/Shanghai',
      'America/New_York',
      'Europe/Berlin',
    ])
    for (const template of desktopTemplates) {
      expect(template.fingerprint).toMatchObject({ os: 'native', uaMode: 'native', userAgent: '' })
    }
  })

  it('applies fingerprint values without changing profile, browser, or network choices', () => {
    const draft = defaultDraft()
    draft.name = 'Regression profile'
    draft.browser = { kind: 'edge' }
    draft.proxy = { mode: 'http', host: '127.0.0.1', port: 8080 }
    draft.fingerprint.webrtcPolicy = 'native'

    const applied = applyDesktopTemplate(draft, 'de-de-desktop')

    expect(applied).toMatchObject({
      name: 'Regression profile',
      browser: { kind: 'edge' },
      proxy: { mode: 'http', host: '127.0.0.1', port: 8080 },
      fingerprint: {
        locale: 'de-DE',
        languages: ['de-DE', 'de', 'en'],
        timezone: 'Europe/Berlin',
        webrtcPolicy: 'native',
      },
    })
  })

  it('returns independent arrays and rejects unknown template IDs', () => {
    const first = applyDesktopTemplate(defaultDraft(), 'zh-cn-desktop')
    const second = applyDesktopTemplate(defaultDraft(), 'zh-cn-desktop')

    first.fingerprint.languages.push('changed')
    expect(second.fingerprint.languages).not.toContain('changed')
    expect(() => applyDesktopTemplate(defaultDraft(), 'missing')).toThrow('unknown desktop template')
  })
})
