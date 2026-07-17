<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import CapabilityPanel from '@/components/CapabilityPanel.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import MetricTile from '@/components/MetricTile.vue'
import ProfileCard from '@/components/ProfileCard.vue'
import ProfileEditor from '@/components/ProfileEditor.vue'
import RecycleBin from '@/components/RecycleBin.vue'
import { useProfileManager } from '@/composables/useProfileManager'
import type { Profile, TrashItem } from '@/domain/profile'

const manager = useProfileManager()
const {
  profiles, trash, capabilities, reports, search, loading, loadError, notice, editorOpen, editingProfile,
  draft, actionIds, filteredProfiles, sessionByProfile, runningCount, featureCoverage, draftReport, draftHasErrors,
} = manager
const deleteTarget = ref<Profile | null>(null)
const purgeTarget = ref<TrashItem | null>(null)
const trashOpen = ref(false)
let sessionTimer: number | undefined

const availableBrowserCount = computed(() => capabilities.value.browsers.filter((item) => item.available).length)
const coverageLabel = computed(() => capabilities.value.features.length ? `${Math.round(featureCoverage.value * 100)}%` : '—')
const editorTitle = computed(() => editingProfile.value ? `编辑 ${editingProfile.value.name}` : '创建隔离 Profile')

async function confirmDelete() {
  if (!deleteTarget.value) return
  await manager.remove(deleteTarget.value)
  deleteTarget.value = null
}

async function confirmPurge() {
  if (!purgeTarget.value) return
  await manager.purge(purgeTarget.value)
  purgeTarget.value = null
}

onMounted(() => {
  void manager.load()
  sessionTimer = window.setInterval(() => void manager.refreshSessions(), 6000)
})
onBeforeUnmount(() => { if (sessionTimer !== undefined) window.clearInterval(sessionTimer) })
</script>

<template>
  <div class="app-shell">
    <header class="topbar">
      <a class="brand" href="#main" aria-label="ProfileWeave 首页">
        <span class="brand__mark" aria-hidden="true">P</span>
        <span><b>ProfileWeave</b><small>Local Browser Runtime</small></span>
      </a>
      <div class="topbar__meta"><span class="endpoint"><i aria-hidden="true"></i>127.0.0.1 · 本地服务</span><span class="divider"></span><span class="boundary">授权 QA · 隐私研究 · 会话隔离</span></div>
      <button class="button button--quiet" type="button" @click="trashOpen = true">回收站 <span v-if="trash.length">{{ trash.length }}</span></button>
      <button class="button button--primary topbar__create" type="button" @click="manager.create"><span aria-hidden="true">＋</span>新建 Profile</button>
    </header>

    <main id="main" class="main-content">
      <section class="hero" aria-labelledby="page-title">
        <div><span class="eyebrow">Local isolation workspace</span><h1 id="page-title">浏览器环境，<em>清楚地隔离。</em></h1><p>管理可复用的本地 Profile，在启动之前了解配置是否自洽、哪些设置真正生效。</p></div>
        <div class="hero__assurance"><span aria-hidden="true">◎</span><p><b>存储隔离是交付能力</b><small>不承诺绕过站点风控或检测</small></p></div>
      </section>

      <div v-if="loadError" class="error-banner" role="alert">
        <span class="error-banner__mark" aria-hidden="true">!</span><div><b>本地服务请求未完成</b><p>{{ loadError }}</p></div>
        <button class="button button--quiet" type="button" @click="manager.load">重试</button>
        <button class="dismiss" type="button" aria-label="关闭错误提示" @click="loadError = ''">×</button>
      </div>

      <section class="metrics" aria-label="运行概览">
        <MetricTile label="Profiles" :value="loading ? '—' : profiles.length" detail="本地持久化的隔离环境" />
        <MetricTile label="Running" :value="loading ? '—' : runningCount" detail="由本服务持有的浏览器进程" :tone="runningCount ? 'success' : 'neutral'" />
        <MetricTile label="Applied coverage" :value="loading ? '—' : coverageLabel" detail="能力矩阵中标记为已应用" :tone="featureCoverage === 1 ? 'success' : 'warning'" />
        <MetricTile label="Browsers ready" :value="loading ? '—' : availableBrowserCount" detail="本机发现且可启动" :tone="availableBrowserCount ? 'success' : 'warning'" />
      </section>

      <div class="workspace">
        <section class="profiles-panel" aria-labelledby="profiles-title">
          <div class="panel-toolbar">
            <div><span class="eyebrow">Environments</span><h2 id="profiles-title">隔离 Profiles</h2></div>
            <label class="search-box"><span aria-hidden="true">⌕</span><span class="sr-only">搜索 Profile</span><input v-model="search" type="search" placeholder="搜索名称、标签、地区…" /></label>
          </div>

          <div v-if="loading" class="profiles-grid" aria-label="正在加载 Profiles">
            <article v-for="index in 4" :key="index" class="card-skeleton"><span></span><i></i><i></i><i></i><b></b></article>
          </div>
          <div v-else-if="!profiles.length" class="empty-state">
            <div class="empty-state__graphic" aria-hidden="true"><span>P</span><i></i><i></i></div>
            <span class="eyebrow">No profiles yet</span><h3>创建第一个可复用的隔离环境</h3>
            <p>从常见桌面配置开始，保存语言、时区、窗口与代理意图；每个 Profile 都有独立浏览器数据目录。</p>
            <button class="button button--primary" type="button" @click="manager.create">＋ 创建 Profile</button>
          </div>
          <div v-else-if="!filteredProfiles.length" class="empty-state empty-state--search">
            <span class="search-glyph" aria-hidden="true">⌕</span><h3>没有匹配的 Profile</h3><p>尝试搜索名称、备注、标签、浏览器或 Locale。</p><button class="button button--quiet" type="button" @click="search = ''">清除搜索</button>
          </div>
          <div v-else class="profiles-grid">
            <ProfileCard
              v-for="profile in filteredProfiles" :key="profile.id" :profile="profile"
              :session="sessionByProfile[profile.id]" :report="reports[profile.id]" :busy="actionIds.has(profile.id)"
              @edit="manager.edit(profile)" @duplicate="manager.duplicate(profile)" @remove="deleteTarget = profile"
              @launch="manager.launch(profile)" @stop="manager.stop(profile)"
            />
          </div>
        </section>
        <CapabilityPanel :capabilities="capabilities" :loading="loading" />
      </div>
    </main>

    <ProfileEditor v-model="draft" :open="editorOpen" :title="editorTitle" :report="draftReport" :has-errors="draftHasErrors"
      :browsers="capabilities.browsers" :saving="actionIds.has(editingProfile?.id ?? 'create')" @close="manager.closeEditor" @save="manager.save" />
    <RecycleBin :open="trashOpen" :items="trash" :busy-ids="actionIds" @close="trashOpen = false" @restore="manager.restore" @purge="purgeTarget = $event" />
    <ConfirmDialog :open="Boolean(deleteTarget)" title="移入回收站？" :description="`“${deleteTarget?.name ?? ''}”的配置与浏览器数据会移入本地回收站，之后仍可恢复；运行中必须先停止。`"
      :busy="Boolean(deleteTarget && actionIds.has(deleteTarget.id))" @cancel="deleteTarget = null" @confirm="confirmDelete" />
    <ConfirmDialog :open="Boolean(purgeTarget)" title="永久删除？" :description="`“${purgeTarget?.profile.name ?? ''}”的配置与回收站内浏览器数据将被永久删除，此操作不可撤销。`"
      :busy="Boolean(purgeTarget && actionIds.has(purgeTarget.profile.id))" @cancel="purgeTarget = null" @confirm="confirmPurge" />
    <Transition name="toast"><div v-if="notice" class="toast" role="status"><span aria-hidden="true">✓</span>{{ notice }}</div></Transition>
  </div>
</template>
