import { describe, expect, it } from 'vitest'
import { evaluateDraft } from './evaluate'
import { defaultDraft } from './profile'

describe('draft consistency evaluation', () => {
  it('accepts the safe native desktop template', () => {
    const draft = defaultDraft()
    draft.name = 'Local QA'

    const report = evaluateDraft(draft)

    expect(report.issues.filter((item) => item.severity === 'error')).toHaveLength(0)
    expect(report.score).toBe(100)
  })

  it('blocks conflicting UA and malformed proxy values', () => {
    const draft = defaultDraft()
    draft.name = 'Conflict'
    draft.fingerprint.os = 'windows'
    draft.fingerprint.uaMode = 'custom'
    draft.fingerprint.userAgent = 'Mozilla/5.0 (Macintosh; Intel Mac OS X)'
    draft.proxy = { mode: 'http', host: '', port: 70000 }

    const report = evaluateDraft(draft)

    expect(report.issues.map((item) => item.code)).toEqual(expect.arrayContaining(['ua_os_conflict', 'proxy_invalid']))
    expect(report.score).toBeLessThan(60)
  })
})
