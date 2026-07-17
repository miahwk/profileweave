import type { ConsistencyReport, ProfileDraft, ReportIssue } from './profile'

function issue(severity: ReportIssue['severity'], code: string, message: string, field?: string): ReportIssue {
  return { severity, code, message, field }
}

function uaOSConflict(userAgent: string, os: ProfileDraft['fingerprint']['os']): boolean {
  const ua = userAgent.toLowerCase()
  if (!ua) return false
  if (os === 'windows') return /macintosh|x11; linux/.test(ua)
  if (os === 'macos') return /windows nt|x11; linux/.test(ua)
  return /windows nt|macintosh/.test(ua)
}

function hasValidTimezone(timezone: string): boolean {
  try {
    new Intl.DateTimeFormat('en-US', { timeZone: timezone }).format()
    return true
  } catch {
    return false
  }
}

export function evaluateDraft(draft: ProfileDraft): ConsistencyReport {
  const issues: ReportIssue[] = []
  const fp = draft.fingerprint
  if (!draft.name.trim()) issues.push(issue('error', 'name_required', 'Profile 名称不能为空', 'name'))
  if (fp.os !== 'native') {
    issues.push(issue('warning', 'os_diagnostic_only', '目标操作系统仅用于一致性诊断，运行时仍保留本机操作系统', 'fingerprint.os'))
  }
  try {
    const url = new URL(draft.startURL)
    if (!['http:', 'https:'].includes(url.protocol)) throw new Error()
  } catch {
    issues.push(issue('error', 'start_url_invalid', '启动页必须是有效的 HTTP(S) 地址', 'startURL'))
  }
  if (fp.uaMode === 'custom' && !fp.userAgent.trim()) {
    issues.push(issue('error', 'ua_required', '选择自定义 UA 后必须填写 User-Agent', 'fingerprint.userAgent'))
  } else if (fp.uaMode === 'custom' && uaOSConflict(fp.userAgent, fp.os)) {
    issues.push(issue('error', 'ua_os_conflict', 'User-Agent 与目标操作系统不一致', 'fingerprint.userAgent'))
  }
  if (!hasValidTimezone(fp.timezone)) {
    issues.push(issue('error', 'timezone_invalid', '请输入有效的 IANA 时区，例如 Asia/Shanghai', 'fingerprint.timezone'))
  }
  if (fp.languages.length) {
    issues.push(issue('info', 'languages_diagnostic_only', '语言列表会保存用于诊断；当前运行时只通过 --lang 应用 Locale', 'fingerprint.languages'))
  }
  if (fp.screen.width < 800 || fp.screen.height < 600 || fp.screen.width > 7680 || fp.screen.height > 4320) {
    issues.push(issue('warning', 'screen_unusual', '屏幕尺寸超出常见桌面范围，请确认这是预期配置', 'fingerprint.screen'))
  }
  if (fp.screen.dpr < 0.75 || fp.screen.dpr > 4) {
    issues.push(issue('warning', 'dpr_unusual', 'DPR 超出常见范围，窗口管理器可能无法完整应用', 'fingerprint.screen.dpr'))
  }
  if (draft.proxy.mode !== 'direct') {
    if (!draft.proxy.host.trim() || draft.proxy.port < 1 || draft.proxy.port > 65535) {
      issues.push(issue('error', 'proxy_invalid', '代理需要有效的主机与 1–65535 端口', 'proxy'))
    }
    if (fp.webrtcPolicy === 'native') {
      issues.push(issue('warning', 'webrtc_proxy_mismatch', '原生 WebRTC 可能产生不经过代理的 UDP 流量', 'fingerprint.webrtcPolicy'))
    }
  }
  if (fp.uaMode === 'custom') {
    issues.push(issue('warning', 'ua_partial', '自定义 UA 无法同步全部 Client Hints 与 TLS 特征', 'fingerprint.uaMode'))
  }
  issues.push(issue('info', 'server_authoritative', '保存和启动时仍会由本地服务执行权威校验'))

  const deductions = issues.reduce((score, item) => score + (item.severity === 'error' ? 30 : item.severity === 'warning' ? 10 : 0), 0)
  return { score: Math.max(0, 100 - deductions), issues }
}
