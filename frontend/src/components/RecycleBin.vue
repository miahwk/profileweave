<script setup lang="ts">
import type { TrashItem } from '@/domain/profile'

defineProps<{ open: boolean; items: TrashItem[]; busyIds: Set<string> }>()
defineEmits<{ close: []; restore: [item: TrashItem]; purge: [item: TrashItem] }>()

function formatDeletedAt(value: string) {
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit',
  }).format(new Date(value))
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="recycle-layer" @keydown.esc="$emit('close')">
      <button class="recycle-backdrop" type="button" aria-label="关闭回收站" @click="$emit('close')"></button>
      <section class="recycle-panel" role="dialog" aria-modal="true" aria-labelledby="recycle-title">
        <header>
          <div><span class="eyebrow">Recoverable deletion</span><h2 id="recycle-title">回收站</h2></div>
          <button class="close-button" type="button" aria-label="关闭回收站" @click="$emit('close')">×</button>
        </header>
        <p class="recycle-intro">删除的 Profile 会保留在本地，可连同仍存在的浏览器数据一起恢复。永久删除不可撤销。</p>

        <div v-if="!items.length" class="empty-recycle">
          <span aria-hidden="true">✓</span><h3>回收站为空</h3><p>当前没有待恢复或待清理的 Profile。</p>
        </div>
        <ul v-else class="recycle-list">
          <li v-for="item in items" :key="item.profile.id">
            <div class="entry-mark" aria-hidden="true">{{ item.profile.name.slice(0, 1).toUpperCase() }}</div>
            <div class="entry-copy">
              <b>{{ item.profile.name }}</b>
              <span>删除于 {{ formatDeletedAt(item.deletedAt) }}</span>
              <small>{{ item.hasBrowserData ? '浏览器数据可恢复' : '仅可恢复 Profile 配置' }}</small>
            </div>
            <div class="entry-actions">
              <button class="button button--quiet" type="button" :disabled="busyIds.has(item.profile.id)" @click="$emit('restore', item)">
                {{ busyIds.has(item.profile.id) ? '处理中…' : '恢复' }}
              </button>
              <button class="purge-button" type="button" :disabled="busyIds.has(item.profile.id)" @click="$emit('purge', item)">永久删除</button>
            </div>
          </li>
        </ul>
      </section>
    </div>
  </Teleport>
</template>

<style scoped>
.recycle-layer { position: fixed; z-index: 70; inset: 0; display: flex; justify-content: flex-end; }
.recycle-backdrop { position: absolute; inset: 0; border: 0; background: rgba(3,5,8,.66); backdrop-filter: blur(5px); }
.recycle-panel { position: relative; width: min(100%, 520px); height: 100%; padding: 28px; overflow-y: auto; border-left: 1px solid var(--line-strong); background: var(--surface-1); box-shadow: var(--shadow-lg); }
header { display: flex; align-items: center; justify-content: space-between; gap: 20px; }h2 { margin: 3px 0 0; color: var(--text); font: 650 25px var(--display); }.close-button { width: 36px; height: 36px; border: 1px solid var(--line); border-radius: 10px; color: var(--text-soft); background: var(--surface-2); font-size: 20px; cursor: pointer; }
.recycle-intro { margin: 18px 0 24px; color: var(--text-soft); font-size: 11px; line-height: 1.7; }.empty-recycle { padding: 70px 20px; border: 1px dashed var(--line-strong); border-radius: var(--radius-lg); text-align: center; }.empty-recycle span { display: grid; place-items: center; width: 42px; height: 42px; margin: auto; border-radius: 50%; color: var(--mint); background: rgba(80,217,169,.12); }.empty-recycle h3 { margin: 14px 0 6px; color: var(--text); font: 650 15px var(--display); }.empty-recycle p { margin: 0; color: var(--text-soft); font-size: 10px; }
.recycle-list { display: grid; gap: 10px; margin: 0; padding: 0; list-style: none; }.recycle-list li { display: flex; align-items: center; gap: 12px; padding: 15px; border: 1px solid var(--line); border-radius: 13px; background: var(--surface-2); }.entry-mark { display: grid; place-items: center; flex: 0 0 38px; height: 38px; border: 1px solid var(--line-strong); border-radius: 10px; color: var(--text); font: 700 12px var(--display); }.entry-copy { display: grid; min-width: 0; gap: 3px; flex: 1; }.entry-copy b { overflow: hidden; color: var(--text); font: 650 12px var(--display); text-overflow: ellipsis; white-space: nowrap; }.entry-copy span,.entry-copy small { color: var(--text-soft); font-size: 9px; }.entry-copy small { color: var(--muted); }.entry-actions { display: flex; align-items: center; gap: 7px; }.purge-button { padding: 8px 9px; border: 0; color: #ffaaa5; background: transparent; font-size: 9px; cursor: pointer; }.purge-button:hover:not(:disabled) { border-radius: 8px; background: rgba(255,105,97,.1); }.purge-button:disabled { cursor: not-allowed; opacity: .42; }
@media (max-width: 560px) { .recycle-panel { padding: 22px 16px; }.recycle-list li { align-items: flex-start; flex-wrap: wrap; }.entry-actions { width: 100%; justify-content: flex-end; } }
</style>
