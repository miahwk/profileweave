import { describe, expect, it } from 'vitest'
import { snapshotFrom } from './selfCheck'

describe('browser environment self-check', () => {
  it('copies actual runtime values into a stable serializable snapshot', () => {
    const languages = ['zh-CN', 'en-US']
    const snapshot = snapshotFrom({
      now: () => new Date('2026-07-18T00:00:00Z'),
      userAgent: 'Runtime UA', platform: 'Win32', language: 'zh-CN', languages,
      timezone: 'Asia/Shanghai',
      screen: { width: 1920, height: 1080, availWidth: 1920, availHeight: 1040 },
      dpr: 1.25, hardwareConcurrency: 8, deviceMemoryGB: 16,
      cookieEnabled: true, webdriver: false,
    })

    languages.push('mutated')
    expect(snapshot).toMatchObject({
      capturedAt: '2026-07-18T00:00:00.000Z',
      userAgent: 'Runtime UA', timezone: 'Asia/Shanghai',
      languages: ['zh-CN', 'en-US'], hardwareConcurrency: 8, deviceMemoryGB: 16,
      screen: { width: 1920, availableHeight: 1040, dpr: 1.25 },
      webdriver: false,
    })
  })
})
