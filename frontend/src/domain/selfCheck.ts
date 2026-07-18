export interface BrowserEnvironmentSnapshot {
  capturedAt: string
  userAgent: string
  platform: string
  language: string
  languages: string[]
  timezone: string
  screen: { width: number; height: number; availableWidth: number; availableHeight: number; dpr: number }
  hardwareConcurrency?: number
  deviceMemoryGB?: number
  cookieEnabled: boolean
  webdriver: boolean
}

export interface BrowserEnvironmentSource {
  now: () => Date
  userAgent: string
  platform: string
  language: string
  languages: readonly string[]
  timezone: string
  screen: { width: number; height: number; availWidth: number; availHeight: number }
  dpr: number
  hardwareConcurrency?: number
  deviceMemoryGB?: number
  cookieEnabled: boolean
  webdriver: boolean
}

export function snapshotFrom(source: BrowserEnvironmentSource): BrowserEnvironmentSnapshot {
  return {
    capturedAt: source.now().toISOString(),
    userAgent: source.userAgent,
    platform: source.platform,
    language: source.language,
    languages: [...source.languages],
    timezone: source.timezone,
    screen: {
      width: source.screen.width,
      height: source.screen.height,
      availableWidth: source.screen.availWidth,
      availableHeight: source.screen.availHeight,
      dpr: source.dpr,
    },
    hardwareConcurrency: source.hardwareConcurrency,
    deviceMemoryGB: source.deviceMemoryGB,
    cookieEnabled: source.cookieEnabled,
    webdriver: source.webdriver,
  }
}

export function collectBrowserEnvironment(): BrowserEnvironmentSnapshot {
  const extendedNavigator = navigator as Navigator & { deviceMemory?: number }
  return snapshotFrom({
    now: () => new Date(),
    userAgent: navigator.userAgent,
    platform: navigator.platform,
    language: navigator.language,
    languages: navigator.languages,
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    screen,
    dpr: window.devicePixelRatio,
    hardwareConcurrency: navigator.hardwareConcurrency || undefined,
    deviceMemoryGB: extendedNavigator.deviceMemory,
    cookieEnabled: navigator.cookieEnabled,
    webdriver: navigator.webdriver,
  })
}
