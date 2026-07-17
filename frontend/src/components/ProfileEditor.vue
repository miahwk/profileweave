<script setup lang="ts">
import { computed, ref } from 'vue'
import ReportList from './ReportList.vue'
import { applyDesktopTemplate, desktopTemplates } from '@/domain/desktopTemplates'
import type { BrowserCapability, ConsistencyReport, ProfileDraft } from '@/domain/profile'

const model = defineModel<ProfileDraft>({ required: true })
const props = defineProps<{
  open: boolean
  title: string
  report: ConsistencyReport
  browsers: BrowserCapability[]
  saving: boolean
  hasErrors: boolean
}>()
const emit = defineEmits<{ close: []; save: [] }>()
const selectedTemplateID = ref('')

const tagsText = computed({
  get: () => model.value.tags.join(', '),
  set: (value: string) => { model.value.tags = value.split(',').map((item) => item.trim()).filter(Boolean) },
})
const languagesText = computed({
  get: () => model.value.fingerprint.languages.join(', '),
  set: (value: string) => { model.value.fingerprint.languages = value.split(',').map((item) => item.trim()).filter(Boolean) },
})
const selectedBrowserMissing = computed(() => model.value.browser.kind !== 'auto' && model.value.browser.kind !== 'custom'
  && !props.browsers.some((browser) => browser.id === model.value.browser.kind && browser.available))
const selectedTemplate = computed(() => desktopTemplates.find((template) => template.id === selectedTemplateID.value))

function useSelectedTemplate() {
  if (!selectedTemplateID.value) return
  model.value = applyDesktopTemplate(model.value, selectedTemplateID.value)
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="editor-layer" @keydown.esc="emit('close')">
      <button class="editor-backdrop" type="button" aria-label="关闭编辑器" @click="emit('close')"></button>
      <form class="editor" role="dialog" aria-modal="true" aria-labelledby="editor-title" @submit.prevent="emit('save')">
        <header class="editor__header">
          <div><span class="eyebrow">Profile configuration</span><h2 id="editor-title">{{ title }}</h2></div>
          <button class="close-button" type="button" aria-label="关闭编辑器" @click="emit('close')">×</button>
        </header>

        <div class="editor__body">
          <aside class="template-picker" aria-labelledby="template-title">
            <div><span class="eyebrow">Controlled presets</span><h3 id="template-title">从稳定桌面模板开始</h3><p>{{ selectedTemplate?.summary ?? '选择后仍可逐项调整。' }}</p></div>
            <div class="template-picker__controls">
              <select v-model="selectedTemplateID" aria-label="桌面模板">
                <option value="">选择模板</option>
                <option v-for="template in desktopTemplates" :key="template.id" :value="template.id">{{ template.label }}</option>
              </select>
              <button class="button button--quiet" type="button" :disabled="!selectedTemplateID" @click="useSelectedTemplate">应用模板</button>
            </div>
            <small>模板只是确定性的配置起点；保存模板不代表所有字段已由浏览器应用，也不代表环境不可检测。</small>
          </aside>
          <section aria-labelledby="basic-title">
            <div class="section-title"><span>01</span><div><h3 id="basic-title">基本信息</h3><p>命名并选择用于启动隔离会话的本地浏览器。</p></div></div>
            <div class="field-grid">
              <label class="field field--wide"><span>Profile 名称 <b>*</b></span><input v-model.trim="model.name" required maxlength="80" placeholder="例如：上海 QA · Chrome" /></label>
              <label class="field field--wide"><span>备注</span><textarea v-model.trim="model.notes" rows="2" maxlength="500" placeholder="记录环境用途，不要存放密码或凭据"></textarea></label>
              <label class="field"><span>启动页 <b>*</b></span><input v-model.trim="model.startURL" required type="url" placeholder="https://example.com" /></label>
              <label class="field"><span>标签</span><input v-model="tagsText" placeholder="QA, 上海, 客户 A" /><small>使用逗号分隔</small></label>
              <label class="field field--wide"><span>浏览器</span>
                <select v-model="model.browser.kind">
                  <option value="auto">自动选择已发现的浏览器</option>
                  <option v-for="browser in browsers" :key="browser.id" :value="browser.id" :disabled="!browser.available">{{ browser.name }}{{ browser.available ? '' : '（未找到）' }}</option>
                  <option value="custom">自定义可执行文件</option>
                </select>
                <small v-if="selectedBrowserMissing" class="field-error">当前选择不可用，请改用已发现的浏览器。</small>
              </label>
              <label v-if="model.browser.kind === 'custom'" class="field field--wide"><span>浏览器可执行文件路径 <b>*</b></span><input v-model.trim="model.browser.customPath" required placeholder="C:\Program Files\Chromium\chrome.exe" /><small>只填写文件路径，不要附加命令行参数。</small></label>
            </div>
          </section>

          <section aria-labelledby="network-title">
            <div class="section-title"><span>02</span><div><h3 id="network-title">网络</h3><p>代理只影响该浏览器的流量，不是系统 VPN；MVP 不保存代理凭据。</p></div></div>
            <fieldset class="choice-group"><legend>连接方式</legend>
              <label><input v-model="model.proxy.mode" type="radio" value="direct" /><span><b>直连</b><small>使用本机网络</small></span></label>
              <label><input v-model="model.proxy.mode" type="radio" value="http" /><span><b>HTTP</b><small>无认证代理</small></span></label>
              <label><input v-model="model.proxy.mode" type="radio" value="socks5" /><span><b>SOCKS5</b><small>无认证代理</small></span></label>
            </fieldset>
            <div v-if="model.proxy.mode !== 'direct'" class="field-grid field-grid--proxy">
              <label class="field"><span>代理主机 <b>*</b></span><input v-model.trim="model.proxy.host" required placeholder="127.0.0.1" /></label>
              <label class="field"><span>端口 <b>*</b></span><input v-model.number="model.proxy.port" required type="number" min="1" max="65535" placeholder="7890" /></label>
            </div>
          </section>

          <section aria-labelledby="fingerprint-title">
            <div class="section-title"><span>03</span><div><h3 id="fingerprint-title">指纹与一致性</h3><p>这些是环境意图；能力矩阵会明确哪些设置可应用、部分应用或未支持。</p></div></div>
            <div class="field-grid">
              <label class="field"><span>目标操作系统</span><select v-model="model.fingerprint.os"><option value="native">跟随本机（推荐）</option><option value="windows">Windows</option><option value="macos">macOS</option><option value="linux">Linux</option></select></label>
              <label class="field"><span>User-Agent 策略</span><select v-model="model.fingerprint.uaMode"><option value="native">使用浏览器原生 UA（推荐）</option><option value="custom">自定义 UA（部分应用）</option></select></label>
              <label v-if="model.fingerprint.uaMode === 'custom'" class="field field--wide"><span>自定义 User-Agent <b>*</b></span><textarea v-model.trim="model.fingerprint.userAgent" required rows="2" placeholder="Mozilla/5.0 ..."></textarea></label>
              <label class="field"><span>Locale</span><input v-model.trim="model.fingerprint.locale" required placeholder="zh-CN" /></label>
              <label class="field"><span>语言列表</span><input v-model="languagesText" required placeholder="zh-CN, zh, en-US" /></label>
              <label class="field field--wide"><span>IANA 时区</span><input v-model.trim="model.fingerprint.timezone" required placeholder="Asia/Shanghai" /><small>时区在 MVP 中仅校验和展示，尚未注入浏览器内核。</small></label>
              <label class="field"><span>屏幕宽度</span><input v-model.number="model.fingerprint.screen.width" type="number" min="320" max="7680" /></label>
              <label class="field"><span>屏幕高度</span><input v-model.number="model.fingerprint.screen.height" type="number" min="320" max="4320" /></label>
              <label class="field"><span>DPR</span><input v-model.number="model.fingerprint.screen.dpr" type="number" min="0.5" max="5" step="0.25" /></label>
              <label class="field"><span>WebRTC 策略</span><select v-model="model.fingerprint.webrtcPolicy"><option value="native">原生</option><option value="proxy_only">仅代理（部分应用）</option></select></label>
              <label class="field"><span>CPU 核心意图</span><input v-model.number="model.fingerprint.cpuCores" type="number" min="1" max="64" /></label>
              <label class="field"><span>设备内存意图（GB）</span><input v-model.number="model.fingerprint.memoryGB" type="number" min="1" max="128" /></label>
            </div>
            <ReportList class="live-report" :report="report" />
          </section>
        </div>

        <footer class="editor__footer">
          <p><b>保存前实时诊断</b><span v-if="hasErrors">存在阻止保存的错误</span><span v-else>服务端将在启动前再次校验</span></p>
          <div><button class="button button--quiet" type="button" @click="emit('close')">取消</button><button class="button button--primary" type="submit" :disabled="saving || hasErrors || selectedBrowserMissing">{{ saving ? '保存中…' : '保存 Profile' }}</button></div>
        </footer>
      </form>
    </div>
  </Teleport>
</template>

<style scoped>
.editor-layer { position: fixed; z-index: 60; inset: 0; display: flex; justify-content: flex-end; }.editor-backdrop { position: absolute; inset: 0; border: 0; background: rgba(3,5,8,.72); backdrop-filter: blur(5px); }
.editor { position: relative; display: flex; flex-direction: column; width: min(720px, 94vw); height: 100%; border-left: 1px solid var(--line-strong); background: var(--surface-0); box-shadow: -24px 0 80px rgba(0,0,0,.42); }
.editor__header { display: flex; align-items: center; justify-content: space-between; gap: 20px; flex: 0 0 auto; padding: 21px 26px; border-bottom: 1px solid var(--line); background: rgba(15,19,26,.94); }.eyebrow { color: var(--mint); font-size: 9px; font-weight: 800; letter-spacing: .16em; text-transform: uppercase; }.editor__header h2 { margin: 5px 0 0; color: var(--text); font: 650 21px var(--display); }.close-button { width: 35px; height: 35px; border: 1px solid var(--line); border-radius: 10px; color: var(--muted); background: var(--surface-2); font-size: 20px; cursor: pointer; }
.editor__body { min-height: 0; overflow-y: auto; padding: 0 26px 40px; }.editor__body section { padding: 27px 0; border-bottom: 1px solid var(--line); }.section-title { display: flex; gap: 13px; margin-bottom: 20px; }.section-title > span { display: grid; place-items: center; flex: 0 0 29px; height: 29px; border: 1px solid var(--line); border-radius: 8px; color: var(--mint); background: rgba(80,217,169,.06); font: 750 9px var(--mono); }.section-title h3 { margin: 0; color: var(--text); font: 650 14px var(--display); }.section-title p { margin: 4px 0 0; color: var(--text-soft); font-size: 10px; line-height: 1.45; }
.template-picker { display: grid; grid-template-columns: 1fr auto; gap: 12px 20px; margin-top: 22px; padding: 16px; border: 1px solid rgba(80,217,169,.2); border-radius: var(--radius-md); background: rgba(80,217,169,.045); }.template-picker h3 { margin: 4px 0 0; color: var(--text); font: 650 13px var(--display); }.template-picker p,.template-picker small { margin: 4px 0 0; color: var(--text-soft); font-size: 9px; line-height: 1.5; }.template-picker > small { grid-column: 1 / -1; }.template-picker__controls { display: flex; align-items: center; gap: 8px; }.template-picker__controls select { width: 142px; }.template-picker__controls .button { white-space: nowrap; }
.field-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 15px; }.field-grid--proxy { margin-top: 15px; }.field--wide { grid-column: 1 / -1; }.field { display: grid; align-content: start; gap: 7px; color: var(--muted); font-size: 10px; font-weight: 700; }.field > span b { color: var(--danger); }.field small { color: var(--text-soft); font-size: 9px; font-weight: 450; line-height: 1.45; }.field .field-error { color: #ffaaa5; }
input,select,textarea { width: 100%; min-width: 0; padding: 10px 11px; border: 1px solid var(--line); border-radius: 9px; outline: none; color: var(--text); background: var(--surface-2); font: 500 11px/1.4 var(--sans); transition: border-color .18s, box-shadow .18s; }textarea { resize: vertical; }input:focus,select:focus,textarea:focus { border-color: var(--mint); box-shadow: 0 0 0 3px rgba(80,217,169,.1); }input::placeholder,textarea::placeholder { color: #596272; }
.choice-group { display: grid; grid-template-columns: repeat(3,1fr); gap: 9px; margin: 0; padding: 0; border: 0; }.choice-group legend { margin-bottom: 8px; color: var(--muted); font-size: 10px; font-weight: 700; }.choice-group label { position: relative; cursor: pointer; }.choice-group input { position: absolute; opacity: 0; pointer-events: none; }.choice-group span { display: block; padding: 11px; border: 1px solid var(--line); border-radius: 9px; background: var(--surface-2); }.choice-group b,.choice-group small { display: block; }.choice-group b { color: var(--muted); font-size: 10px; }.choice-group small { margin-top: 3px; color: var(--text-soft); font-size: 9px; }.choice-group input:checked + span { border-color: rgba(80,217,169,.55); background: rgba(80,217,169,.075); box-shadow: 0 0 0 2px rgba(80,217,169,.05); }.choice-group input:checked + span b { color: var(--mint); }.choice-group input:focus-visible + span { outline: 2px solid var(--blue); outline-offset: 2px; }
.live-report { margin-top: 20px; }.editor__footer { display: flex; align-items: center; justify-content: space-between; gap: 18px; flex: 0 0 auto; padding: 15px 24px; border-top: 1px solid var(--line); background: rgba(15,19,26,.97); }.editor__footer p { display: grid; gap: 2px; margin: 0; }.editor__footer p b { color: var(--muted); font-size: 10px; }.editor__footer p span { color: var(--text-soft); font-size: 9px; }.editor__footer > div { display: flex; gap: 8px; }
@media (max-width: 620px) { .editor { width: 100%; }.editor__header,.editor__body { padding-inline: 18px; }.template-picker { grid-template-columns: 1fr; }.template-picker__controls select { width: auto; flex: 1; }.field-grid { grid-template-columns: 1fr; }.field--wide { grid-column: auto; }.choice-group { grid-template-columns: 1fr; }.editor__footer { align-items: stretch; flex-direction: column; }.editor__footer > div { display: grid; grid-template-columns: 1fr 1fr; } }
</style>
