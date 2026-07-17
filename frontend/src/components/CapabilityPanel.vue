<script setup lang="ts">
import type { Capabilities, CapabilityStatus } from '@/domain/profile'

defineProps<{ capabilities: Capabilities; loading: boolean }>()
const statusText: Record<CapabilityStatus, string> = { applied: '已应用', partial: '部分应用', unsupported: '未支持' }
</script>

<template>
  <aside class="capability-panel" aria-labelledby="capability-title">
    <header>
      <div>
        <span class="eyebrow">Runtime truth</span>
        <h2 id="capability-title">运行能力</h2>
      </div>
      <span class="local-badge"><i aria-hidden="true"></i>本机</span>
    </header>
    <div v-if="loading" class="panel-skeleton" aria-label="正在读取运行能力">
      <span v-for="index in 5" :key="index"></span>
    </div>
    <template v-else>
      <section>
        <h3>发现的浏览器</h3>
        <ul v-if="capabilities.browsers.length" class="browser-list">
          <li v-for="browser in capabilities.browsers" :key="browser.id">
            <span class="browser-mark" aria-hidden="true">{{ browser.name.slice(0, 1).toUpperCase() }}</span>
            <div><b>{{ browser.name }}</b><small>{{ browser.available ? '可启动' : '未找到可执行文件' }}</small></div>
            <span class="availability" :class="{ 'availability--ready': browser.available }">{{ browser.available ? '就绪' : '缺失' }}</span>
          </li>
        </ul>
        <p v-else class="subtle-empty">暂未发现 Chromium 浏览器，可在 Profile 中选择自定义路径。</p>
      </section>
      <section>
        <div class="section-heading">
          <h3>配置生效矩阵</h3>
          <span>{{ capabilities.features.length }} 项</span>
        </div>
        <ul v-if="capabilities.features.length" class="feature-list">
          <li v-for="feature in capabilities.features" :key="feature.key">
            <div><b>{{ feature.label }}</b><small v-if="feature.detail">{{ feature.detail }}</small></div>
            <span class="status-pill" :class="`status-pill--${feature.status}`">
              <i aria-hidden="true"></i>{{ statusText[feature.status] }}
            </span>
          </li>
        </ul>
        <p v-else class="subtle-empty">服务尚未返回能力矩阵。</p>
      </section>
      <p class="honesty-note"><b>边界说明</b> “已保存”不等于“已在内核生效”。未支持项会保持原生行为，不进行易检测的脚本伪装。</p>
    </template>
  </aside>
</template>

<style scoped>
.capability-panel { position: sticky; top: 24px; align-self: start; overflow: hidden; border: 1px solid var(--line); border-radius: var(--radius-xl); background: var(--surface-1); box-shadow: var(--shadow-lg); }
header { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; padding: 22px; border-bottom: 1px solid var(--line); background: linear-gradient(135deg, rgba(102,161,255,.07), transparent 48%); }
.eyebrow { color: var(--blue); font-size: 9px; font-weight: 800; letter-spacing: .17em; text-transform: uppercase; }
h2 { margin: 5px 0 0; color: var(--text); font: 650 20px/1.1 var(--display); letter-spacing: -.02em; }
.local-badge { display: inline-flex; align-items: center; gap: 7px; padding: 7px 9px; border: 1px solid var(--line); border-radius: 999px; color: var(--text-soft); background: rgba(255,255,255,.025); font-size: 10px; font-weight: 700; }
.local-badge i { width: 6px; height: 6px; border-radius: 50%; background: var(--mint); box-shadow: 0 0 0 3px rgba(80,217,169,.1); }
section { padding: 18px 20px; border-bottom: 1px solid var(--line); }
h3 { margin: 0 0 12px; color: var(--muted); font-size: 10px; font-weight: 800; letter-spacing: .1em; text-transform: uppercase; }
.section-heading { display: flex; align-items: center; justify-content: space-between; }.section-heading span { color: var(--text-soft); font-size: 10px; }
ul { margin: 0; padding: 0; list-style: none; }
.browser-list { display: grid; gap: 8px; }.browser-list li { display: grid; grid-template-columns: auto 1fr auto; align-items: center; gap: 10px; }
.browser-mark { display: grid; place-items: center; width: 31px; height: 31px; border: 1px solid var(--line); border-radius: 9px; color: var(--text); background: var(--surface-2); font: 700 11px var(--display); }
.browser-list b,.feature-list b { display: block; color: var(--text); font-size: 11px; }.browser-list small,.feature-list small { display: block; margin-top: 2px; color: var(--text-soft); font-size: 9px; line-height: 1.35; }
.availability { color: var(--text-soft); font-size: 9px; font-weight: 700; }.availability--ready { color: var(--mint); }
.feature-list { display: grid; gap: 12px; }.feature-list li { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
.feature-list div { min-width: 0; }.status-pill { display: inline-flex; align-items: center; gap: 5px; flex: 0 0 auto; padding: 4px 7px; border-radius: 999px; color: var(--text-soft); background: var(--surface-2); font-size: 9px; font-weight: 750; }
.status-pill i { width: 5px; height: 5px; border-radius: 50%; background: currentColor; }.status-pill--applied { color: var(--mint); }.status-pill--partial { color: var(--amber); }.status-pill--unsupported { color: #b3bdca; }
.honesty-note { margin: 0; padding: 17px 20px 20px; color: var(--text-soft); background: rgba(255,255,255,.012); font-size: 10px; line-height: 1.55; }.honesty-note b { color: var(--muted); }
.subtle-empty { margin: 0; color: var(--text-soft); font-size: 10px; line-height: 1.55; }
.panel-skeleton { display: grid; gap: 12px; padding: 20px; }.panel-skeleton span { height: 36px; border-radius: 8px; background: linear-gradient(90deg, var(--surface-2), var(--surface-3), var(--surface-2)); background-size: 200% 100%; animation: shimmer 1.4s infinite; }
@keyframes shimmer { to { background-position: -200% 0; } }
@media (max-width: 1000px) { .capability-panel { position: static; } }
</style>
