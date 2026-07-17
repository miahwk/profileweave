<script setup lang="ts">
import type { ConsistencyReport, Severity } from '@/domain/profile'

defineProps<{ report: ConsistencyReport; compact?: boolean }>()

const labels: Record<Severity, string> = { error: '错误', warning: '警告', info: '提示' }
</script>

<template>
  <div class="report" :class="{ 'report--compact': compact }">
    <div class="report__summary">
      <div class="score" :class="report.score >= 80 ? 'score--good' : report.score >= 60 ? 'score--fair' : 'score--low'">
        <strong>{{ report.score }}</strong><span>/ 100</span>
      </div>
      <div>
        <b>配置一致性</b>
        <p>{{ report.issues.filter((item) => item.severity === 'error').length }} 个错误 · {{ report.issues.filter((item) => item.severity === 'warning').length }} 个警告</p>
      </div>
    </div>
    <ul v-if="report.issues.length" class="issue-list">
      <li v-for="item in report.issues" :key="`${item.code}-${item.field}`" :class="`issue issue--${item.severity}`">
        <span class="issue__mark" aria-hidden="true">{{ item.severity === 'error' ? '!' : item.severity === 'warning' ? '△' : 'i' }}</span>
        <div><b>{{ labels[item.severity] }}</b><p>{{ item.message }}</p></div>
      </li>
    </ul>
    <div v-else class="report__clear">✓ 未发现一致性问题</div>
  </div>
</template>

<style scoped>
.report { display: grid; gap: 14px; }
.report__summary { display: flex; align-items: center; gap: 14px; padding: 14px; border-radius: var(--radius-md); background: var(--surface-2); border: 1px solid var(--line); }
.score { display: flex; align-items: baseline; gap: 3px; min-width: 78px; color: var(--amber); }
.score--good { color: var(--mint); }.score--low { color: var(--danger); }
.score strong { font: 700 28px/1 var(--display); letter-spacing: -.04em; }.score span { color: var(--text-soft); font-size: 10px; }
.report__summary b { color: var(--text); font-size: 13px; }.report__summary p { margin: 4px 0 0; color: var(--muted); font-size: 11px; }
.issue-list { display: grid; gap: 8px; margin: 0; padding: 0; list-style: none; }
.issue { display: flex; gap: 10px; padding: 10px 12px; border-radius: var(--radius-sm); border: 1px solid var(--line); background: rgba(255,255,255,.018); }
.issue__mark { display: grid; place-items: center; flex: 0 0 20px; height: 20px; border-radius: 50%; color: var(--text); background: var(--surface-3); font-size: 11px; font-weight: 800; }
.issue--error .issue__mark { color: #ffc6c3; background: rgba(255,105,97,.18); }.issue--warning .issue__mark { color: #ffdca9; background: rgba(240,176,84,.16); }.issue--info .issue__mark { color: #b8d4ff; background: rgba(102,161,255,.15); }
.issue b { color: var(--text); font-size: 11px; }.issue p { margin: 2px 0 0; color: var(--text-soft); font-size: 11px; line-height: 1.45; }
.report__clear { padding: 14px; color: var(--mint); border: 1px solid rgba(80,217,169,.2); border-radius: var(--radius-md); background: rgba(80,217,169,.06); font-size: 12px; }
.report--compact .issue-list { max-height: 132px; overflow: auto; }.report--compact .issue { padding: 8px 10px; }
</style>
