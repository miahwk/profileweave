<script setup lang="ts">
import { ref } from 'vue'
import { useDialogFocusTrap } from '@/composables/useDialogFocusTrap'
import type { DoctorReport } from '@/domain/profile'

const props = defineProps<{ open: boolean; report: DoctorReport | null; loading: boolean; error: string }>()
const emit = defineEmits<{ close: []; run: [] }>()
const panel = ref<HTMLElement | null>(null)
const closeButton = ref<HTMLElement | null>(null)

useDialogFocusTrap(() => props.open, panel, closeButton, () => emit('close'))

const statusLabel = { applied: '已生效', partial: '部分生效', unsupported: '未支持' }
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="doctor-layer">
      <button class="doctor-backdrop" type="button" aria-label="关闭运行诊断" @click="emit('close')"></button>
      <section ref="panel" class="doctor-panel" role="dialog" aria-modal="true" aria-labelledby="doctor-title" tabindex="-1">
        <header>
          <div><span class="eyebrow">Runtime doctor</span><h2 id="doctor-title">浏览器运行诊断</h2></div>
          <button ref="closeButton" class="close-button" type="button" aria-label="关闭运行诊断" @click="emit('close')">×</button>
        </header>
        <p class="intro">检查本机浏览器发现、运行 Provider 与实际支持边界。结果只描述可观测能力，不代表可绕过站点检测。</p>

        <div v-if="error" class="doctor-error" role="alert"><b>诊断请求失败</b><span>{{ error }}</span></div>
        <div v-if="loading && !report" class="doctor-loading" aria-live="polite">正在检查本地浏览器环境…</div>

        <template v-if="report">
          <section class="health-card" :class="{ 'health-card--bad': !report.healthy }">
            <span class="health-mark" aria-hidden="true">{{ report.healthy ? '✓' : '!' }}</span>
            <div><b>{{ report.healthy ? '已发现可用浏览器候选' : '未发现可用浏览器候选' }}</b><small>{{ report.provider.name }} · {{ report.provider.id }}</small></div>
            <button class="button button--quiet" type="button" :disabled="loading" @click="$emit('run')">{{ loading ? '检查中…' : '重新检查' }}</button>
          </section>

          <dl class="metrics">
            <div><dt>检查浏览器类型</dt><dd>{{ report.inspectedBrowsers }}</dd></div>
            <div><dt>可用浏览器</dt><dd>{{ report.availableBrowsers }}</dd></div>
            <div><dt>活动会话</dt><dd>{{ report.activeSessions }}</dd></div>
          </dl>

          <section class="section">
            <div class="section-title"><h3>Runtime Provider</h3><span>{{ report.provider.versionManagement }}</span></div>
            <p>{{ report.provider.description }}</p>
            <dl class="provider-facts">
              <div><dt>来源</dt><dd>{{ report.provider.source }}</dd></div>
              <div><dt>许可</dt><dd>{{ report.provider.license }}</dd></div>
            </dl>
          </section>

          <section v-if="report.issues.length" class="section">
            <div class="section-title"><h3>需要处理</h3><span>{{ report.issues.length }} 项</span></div>
            <ul class="issue-list">
              <li v-for="issue in report.issues" :key="issue.code" :class="`issue--${issue.severity}`">
                <b>{{ issue.message }}</b><span v-if="issue.suggestion">{{ issue.suggestion }}</span>
              </li>
            </ul>
          </section>

          <section class="section">
            <div class="section-title"><h3>能力边界</h3><span>{{ report.provider.capabilities.length }} 项</span></div>
            <ul class="capability-list">
              <li v-for="item in report.provider.capabilities" :key="item.id">
                <div><b>{{ item.name }}</b><small>{{ item.detail }}</small></div>
                <span :class="`status status--${item.status}`">{{ statusLabel[item.status] }}</span>
              </li>
            </ul>
          </section>

          <section class="section">
            <div class="section-title"><h3>本地浏览器</h3><span>{{ report.availableBrowsers }}/{{ report.browsers.length }} 可用</span></div>
            <ul class="browser-list">
              <li v-for="browser in report.browsers" :key="browser.id">
                <i :class="{ available: browser.available }" aria-hidden="true"></i>
                <div><b>{{ browser.name }}</b><small>{{ browser.available ? '已发现本机可执行文件（路径已隐藏）' : '未发现可执行文件' }}</small></div>
              </li>
            </ul>
          </section>

          <a class="self-check" href="/self-check" target="_blank" rel="noopener">打开浏览器实际环境自检页 ↗</a>
        </template>

        <button v-else-if="!loading" class="button button--primary first-run" type="button" @click="$emit('run')">开始诊断</button>
      </section>
    </div>
  </Teleport>
</template>

<style scoped>
.doctor-layer { position: fixed; z-index: 75; inset: 0; display: flex; justify-content: flex-end; }.doctor-backdrop { position: absolute; inset: 0; border: 0; background: rgba(3,5,8,.68); backdrop-filter: blur(5px); }.doctor-panel { position: relative; width: min(100%,600px); height: 100%; padding: 28px; overflow-y: auto; border-left: 1px solid var(--line-strong); background: var(--surface-1); box-shadow: var(--shadow-lg); }
header,.health-card,.section-title { display: flex; align-items: center; justify-content: space-between; gap: 16px; }h2 { margin: 3px 0 0; font: 650 25px var(--display); }.close-button { width: 36px; height: 36px; border: 1px solid var(--line); border-radius: 10px; color: var(--text-soft); background: var(--surface-2); font-size: 20px; cursor: pointer; }.intro { margin: 18px 0 22px; color: var(--text-soft); font-size: 11px; line-height: 1.7; }
.doctor-error { display: grid; gap: 4px; margin-bottom: 14px; padding: 12px; border: 1px solid rgba(255,105,97,.22); border-radius: 10px; color: #ffc5c2; background: rgba(255,105,97,.08); font-size: 10px; }.doctor-loading { padding: 70px 20px; color: var(--text-soft); text-align: center; }.health-card { padding: 15px; border: 1px solid rgba(80,217,169,.22); border-radius: 13px; background: rgba(80,217,169,.06); }.health-card--bad { border-color: rgba(255,105,97,.25); background: rgba(255,105,97,.06); }.health-mark { display: grid; place-items: center; width: 34px; height: 34px; border-radius: 50%; color: var(--mint); background: rgba(80,217,169,.12); font-weight: 800; }.health-card--bad .health-mark { color: #ffc5c2; background: rgba(255,105,97,.13); }.health-card > div { display: grid; gap: 3px; margin-right: auto; }.health-card b { font: 650 12px var(--display); }.health-card small { color: var(--text-soft); font-size: 9px; }
.metrics { display: grid; grid-template-columns: repeat(3,1fr); gap: 8px; margin: 12px 0; }.metrics div { padding: 12px; border: 1px solid var(--line); border-radius: 11px; background: var(--surface-2); }.metrics dt { color: var(--text-soft); font-size: 8px; }.metrics dd { margin: 5px 0 0; font: 650 20px var(--display); }.section { margin-top: 12px; padding: 16px; border: 1px solid var(--line); border-radius: 13px; background: var(--surface-2); }.section-title h3 { margin: 0; font: 650 12px var(--display); }.section-title span { color: var(--text-soft); font-size: 8px; }.section > p { margin: 10px 0 0; color: var(--text-soft); font-size: 10px; line-height: 1.6; }.provider-facts { display: grid; gap: 8px; margin: 12px 0 0; }.provider-facts div { display: grid; grid-template-columns: 55px 1fr; gap: 8px; }.provider-facts dt,.provider-facts dd { margin: 0; color: var(--text-soft); font-size: 9px; }.provider-facts dd { color: var(--muted); }
.issue-list,.capability-list,.browser-list { display: grid; gap: 8px; margin: 12px 0 0; padding: 0; list-style: none; }.issue-list li { display: grid; gap: 4px; padding: 10px; border-left: 2px solid var(--amber); border-radius: 7px; background: rgba(240,176,84,.06); }.issue-list b { font-size: 10px; }.issue-list span { color: var(--text-soft); font-size: 9px; line-height: 1.5; }.issue-list .issue--error { border-left-color: var(--danger); }.capability-list li,.browser-list li { display: flex; align-items: center; gap: 10px; padding-top: 8px; border-top: 1px solid var(--line); }.capability-list li:first-child,.browser-list li:first-child { padding-top: 0; border-top: 0; }.capability-list li > div,.browser-list li > div { display: grid; min-width: 0; gap: 3px; flex: 1; }.capability-list b,.browser-list b { font-size: 10px; }.capability-list small,.browser-list small { overflow: hidden; color: var(--text-soft); font-size: 8px; line-height: 1.45; text-overflow: ellipsis; }.status { flex: 0 0 auto; padding: 4px 7px; border-radius: 999px; color: var(--text-soft); background: rgba(255,255,255,.04); font-size: 8px; }.status--applied { color: var(--mint); background: rgba(80,217,169,.09); }.status--partial { color: var(--amber); background: rgba(240,176,84,.09); }.browser-list i { width: 7px; height: 7px; border-radius: 50%; background: var(--danger); }.browser-list i.available { background: var(--mint); }.self-check { display: block; margin-top: 14px; padding: 13px; border: 1px solid rgba(115,167,250,.2); border-radius: 10px; color: var(--blue); background: rgba(115,167,250,.06); font-size: 10px; text-align: center; text-decoration: none; }.first-run { width: 100%; }
@media (max-width: 600px) { .doctor-panel { padding: 22px 16px; }.metrics { grid-template-columns: 1fr; }.health-card { align-items: flex-start; flex-wrap: wrap; }.health-card .button { width: 100%; } }
</style>
