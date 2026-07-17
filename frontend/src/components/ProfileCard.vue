<script setup lang="ts">
import type { ConsistencyReport, Profile, Session } from '@/domain/profile'

const props = defineProps<{
  profile: Profile
  session?: Session
  report?: ConsistencyReport
  busy: boolean
}>()
defineEmits<{ edit: []; duplicate: []; remove: []; launch: []; stop: [] }>()

const browserName = () => props.profile.browser.kind === 'auto' ? '自动选择' : props.profile.browser.kind
const running = () => ['starting', 'running', 'stopping'].includes(props.session?.status ?? '')
const hasErrors = () => props.report?.issues.some((item) => item.severity === 'error') ?? false
const proxyLabel = () => props.profile.proxy.mode === 'direct'
  ? '直连'
  : `${props.profile.proxy.mode.toUpperCase()} · ${props.profile.proxy.host}:${props.profile.proxy.port}`

function formatTime(value?: string) {
  if (!value) return '—'
  return new Intl.DateTimeFormat('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(value))
}
</script>

<template>
  <article class="profile-card" :class="{ 'profile-card--running': running() }">
    <header>
      <div class="profile-symbol" aria-hidden="true">{{ profile.name.slice(0, 1).toUpperCase() }}</div>
      <div class="profile-title">
        <div class="title-row">
          <h3>{{ profile.name }}</h3>
          <span class="run-status" :class="{ 'run-status--active': running() }">
            <i aria-hidden="true"></i>{{ running() ? (session?.status === 'starting' ? '启动中' : '运行中') : '已停止' }}
          </span>
        </div>
        <p>{{ profile.notes || '暂无备注' }}</p>
      </div>
    </header>

    <div v-if="profile.tags.length" class="tag-row" aria-label="标签">
      <span v-for="tag in profile.tags.slice(0, 3)" :key="tag"># {{ tag }}</span>
      <span v-if="profile.tags.length > 3">+{{ profile.tags.length - 3 }}</span>
    </div>

    <dl class="profile-facts">
      <div><dt>浏览器</dt><dd>{{ browserName() }}</dd></div>
      <div><dt>地区意图</dt><dd>{{ profile.fingerprint.locale }} · {{ profile.fingerprint.timezone }}</dd></div>
      <div><dt>网络</dt><dd>{{ proxyLabel() }}</dd></div>
      <div><dt>窗口</dt><dd>{{ profile.fingerprint.screen.width }}×{{ profile.fingerprint.screen.height }} @{{ profile.fingerprint.screen.dpr }}</dd></div>
    </dl>

    <div class="health-row">
      <div class="health-score" :class="(report?.score ?? 0) >= 80 ? 'health-score--good' : ''">
        <span>{{ report?.score ?? '—' }}</span><small>一致性分</small>
      </div>
      <div class="health-details">
        <span v-if="!report">正在分析配置…</span>
        <template v-else>
          <span>{{ report.issues.filter((item) => item.severity === 'error').length }} 错误</span>
          <span>{{ report.issues.filter((item) => item.severity === 'warning').length }} 警告</span>
        </template>
      </div>
      <div v-if="session?.pid" class="pid"><span>PID</span>{{ session.pid }}</div>
    </div>

    <p v-if="running()" class="session-note">本机进程 · 启动于 {{ formatTime(session?.startedAt) }}</p>
    <p v-else-if="hasErrors()" class="blocked-note">配置存在错误，修复后才能启动</p>

    <footer>
      <div class="secondary-actions">
        <button class="icon-button" type="button" :disabled="busy || running()" aria-label="编辑 Profile" title="编辑" @click="$emit('edit')">编辑</button>
        <button class="icon-button" type="button" :disabled="busy" aria-label="复制 Profile" title="复制" @click="$emit('duplicate')">复制</button>
        <button class="icon-button icon-button--danger" type="button" :disabled="busy || running()" aria-label="删除 Profile" title="删除" @click="$emit('remove')">删除</button>
      </div>
      <button v-if="running()" class="action-button action-button--stop" type="button" :disabled="busy" @click="$emit('stop')">
        <span aria-hidden="true">■</span>{{ busy ? '处理中…' : '停止' }}
      </button>
      <button v-else class="action-button" type="button" :disabled="busy || hasErrors()" @click="$emit('launch')">
        <span aria-hidden="true">▶</span>{{ busy ? '处理中…' : '启动' }}
      </button>
    </footer>
  </article>
</template>

<style scoped>
.profile-card { position: relative; display: flex; flex-direction: column; min-height: 360px; padding: 20px; overflow: hidden; border: 1px solid var(--line); border-radius: var(--radius-lg); background: linear-gradient(145deg, rgba(255,255,255,.035), rgba(255,255,255,.012)); box-shadow: var(--shadow-sm); transition: border-color .2s, transform .2s, box-shadow .2s; }
.profile-card::before { content: ''; position: absolute; inset: 0 0 auto; height: 2px; opacity: 0; background: linear-gradient(90deg, var(--mint), transparent 70%); transition: opacity .2s; }.profile-card:hover { transform: translateY(-2px); border-color: var(--line-strong); box-shadow: var(--shadow-md); }.profile-card--running::before { opacity: 1; }
header { display: flex; gap: 12px; }.profile-symbol { display: grid; place-items: center; flex: 0 0 40px; height: 40px; border: 1px solid var(--line-strong); border-radius: 12px; color: var(--text); background: linear-gradient(145deg, var(--surface-3), var(--surface-2)); font: 700 14px var(--display); }
.profile-title { min-width: 0; flex: 1; }.title-row { display: flex; align-items: center; gap: 8px; }.title-row h3 { min-width: 0; margin: 0; overflow: hidden; color: var(--text); font: 650 15px/1.3 var(--display); text-overflow: ellipsis; white-space: nowrap; }.profile-title p { margin: 5px 0 0; overflow: hidden; color: var(--text-soft); font-size: 10px; text-overflow: ellipsis; white-space: nowrap; }
.run-status { display: inline-flex; align-items: center; gap: 5px; flex: 0 0 auto; color: var(--text-soft); font-size: 9px; font-weight: 700; }.run-status i { width: 5px; height: 5px; border-radius: 50%; background: currentColor; }.run-status--active { color: var(--mint); }
.tag-row { display: flex; gap: 5px; margin-top: 14px; overflow: hidden; }.tag-row span { flex: 0 0 auto; padding: 4px 7px; border: 1px solid var(--line); border-radius: 6px; color: var(--text-soft); background: rgba(255,255,255,.02); font-size: 9px; }
.profile-facts { display: grid; grid-template-columns: 1fr 1fr; gap: 13px 18px; margin: 20px 0 0; padding: 17px 0; border-block: 1px solid var(--line); }.profile-facts div { min-width: 0; }.profile-facts dt { margin-bottom: 4px; color: var(--text-soft); font-size: 9px; text-transform: uppercase; letter-spacing: .06em; }.profile-facts dd { margin: 0; overflow: hidden; color: var(--muted); font-size: 10px; text-overflow: ellipsis; white-space: nowrap; }
.health-row { display: flex; align-items: center; gap: 14px; margin-top: 15px; }.health-score { display: flex; align-items: baseline; gap: 5px; color: var(--amber); }.health-score--good { color: var(--mint); }.health-score span { font: 700 22px/1 var(--display); }.health-score small { color: var(--text-soft); font-size: 9px; }.health-details { display: flex; gap: 8px; color: var(--text-soft); font-size: 9px; }.pid { margin-left: auto; color: var(--muted); font: 650 11px var(--mono); }.pid span { margin-right: 5px; color: var(--text-soft); font: 8px var(--sans); }
.session-note,.blocked-note { margin: 9px 0 0; color: var(--text-soft); font-size: 9px; }.blocked-note { color: #ffaaa5; }
footer { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-top: auto; padding-top: 18px; }.secondary-actions { display: flex; gap: 5px; }.icon-button { padding: 7px; border: 0; border-radius: 7px; color: var(--text-soft); background: transparent; font-size: 9px; cursor: pointer; }.icon-button:hover:not(:disabled) { color: var(--text); background: var(--surface-3); }.icon-button--danger:hover:not(:disabled) { color: #ffaaa5; background: rgba(255,105,97,.1); }
.action-button { display: inline-flex; align-items: center; gap: 7px; padding: 9px 14px; border: 1px solid rgba(80,217,169,.25); border-radius: 9px; color: #0a1612; background: var(--mint); font-size: 10px; font-weight: 800; cursor: pointer; box-shadow: 0 6px 18px rgba(80,217,169,.12); }.action-button--stop { border-color: rgba(255,105,97,.2); color: #ffd2cf; background: rgba(255,105,97,.12); box-shadow: none; }.action-button:disabled,.icon-button:disabled { cursor: not-allowed; opacity: .42; }
@media (max-width: 560px) { .profile-card { min-height: 340px; }.profile-facts { grid-template-columns: 1fr; gap: 10px; }.profile-facts div:nth-child(4) { display: none; } }
</style>
