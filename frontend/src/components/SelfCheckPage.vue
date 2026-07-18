<script setup lang="ts">
import { computed, ref } from 'vue'
import { collectBrowserEnvironment } from '@/domain/selfCheck'

const snapshot = ref(collectBrowserEnvironment())
const rows = computed(() => [
  ['User-Agent', snapshot.value.userAgent],
  ['Platform', snapshot.value.platform],
  ['Language', snapshot.value.language],
  ['Languages', snapshot.value.languages.join(', ') || '—'],
  ['Timezone', snapshot.value.timezone || '—'],
  ['Screen', `${snapshot.value.screen.width}×${snapshot.value.screen.height} / available ${snapshot.value.screen.availableWidth}×${snapshot.value.screen.availableHeight}`],
  ['Device pixel ratio', String(snapshot.value.screen.dpr)],
  ['Hardware concurrency', snapshot.value.hardwareConcurrency?.toString() ?? '未报告'],
  ['Device memory', snapshot.value.deviceMemoryGB ? `${snapshot.value.deviceMemoryGB} GB` : '未报告'],
  ['Cookies enabled', snapshot.value.cookieEnabled ? '是' : '否'],
  ['WebDriver exposed', snapshot.value.webdriver ? '是' : '否'],
])

function refresh() {
  snapshot.value = collectBrowserEnvironment()
}
</script>

<template>
  <main class="self-check-shell">
    <header>
      <a class="brand" href="/"><span class="brand__mark" aria-hidden="true">P</span><span><b>ProfileWeave</b><small>Runtime Self-check</small></span></a>
      <button class="button button--quiet" type="button" @click="refresh">重新读取</button>
    </header>
    <section class="self-check-hero">
      <span class="eyebrow">Observed by this browser</span>
      <h1>实际浏览器环境自检</h1>
      <p>本页直接读取当前浏览器暴露的标准 Web API 值。它用于验证配置是否真实生效，不表示环境不可检测，也不测试站点风控绕过。</p>
    </section>
    <section class="observed-panel" aria-labelledby="observed-title">
      <div class="panel-heading"><div><span class="eyebrow">Live snapshot</span><h2 id="observed-title">当前实际值</h2></div><time :datetime="snapshot.capturedAt">{{ new Date(snapshot.capturedAt).toLocaleString('zh-CN') }}</time></div>
      <dl><div v-for="row in rows" :key="row[0]"><dt>{{ row[0] }}</dt><dd>{{ row[1] }}</dd></div></dl>
    </section>
    <aside><b>如何验证 Profile</b><p>在 Profile 编辑器中选择“使用本地自检页”，保存后启动该 Profile。这里显示的是被启动浏览器的实际值，可与配置意图及能力矩阵逐项比较。</p></aside>
  </main>
</template>

<style scoped>
.self-check-shell { width: min(980px,100%); margin: 0 auto; padding: 28px clamp(18px,5vw,54px) 70px; }.self-check-shell > header { display: flex; align-items: center; justify-content: space-between; gap: 20px; }.self-check-hero { padding: clamp(55px,10vw,100px) 0 32px; }.self-check-hero h1 { margin: 8px 0 14px; color: var(--text); font: 650 clamp(34px,6vw,58px)/1 var(--display); letter-spacing: -.045em; }.self-check-hero p { max-width: 720px; margin: 0; color: var(--text-soft); font-size: 12px; line-height: 1.8; }
.observed-panel { overflow: hidden; border: 1px solid var(--line-strong); border-radius: var(--radius-xl); background: var(--surface-1); box-shadow: var(--shadow-md); }.panel-heading { display: flex; align-items: flex-end; justify-content: space-between; gap: 20px; padding: 22px 24px; border-bottom: 1px solid var(--line); }.panel-heading h2 { margin: 5px 0 0; font: 650 18px var(--display); }.panel-heading time { color: var(--text-soft); font: 9px var(--mono); }dl { margin: 0; }dl div { display: grid; grid-template-columns: 190px minmax(0,1fr); gap: 18px; padding: 14px 24px; border-bottom: 1px solid var(--line); }dl div:last-child { border-bottom: 0; }dt { color: var(--text-soft); font-size: 10px; }dd { margin: 0; overflow-wrap: anywhere; color: var(--muted); font: 11px/1.55 var(--mono); }
aside { margin-top: 18px; padding: 16px 18px; border: 1px solid rgba(115,167,250,.2); border-radius: var(--radius-md); background: rgba(115,167,250,.06); }aside b { color: var(--blue); font-size: 11px; }aside p { margin: 6px 0 0; color: var(--text-soft); font-size: 10px; line-height: 1.7; }
@media (max-width: 620px) { dl div { grid-template-columns: 1fr; gap: 6px; }.panel-heading { align-items: flex-start; flex-direction: column; } }
</style>
