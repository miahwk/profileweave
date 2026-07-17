<script setup lang="ts">
defineProps<{ open: boolean; title: string; description: string; busy?: boolean }>()
defineEmits<{ confirm: []; cancel: [] }>()
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="dialog-layer" role="presentation" @keydown.esc="$emit('cancel')">
      <button class="dialog-backdrop" aria-label="关闭确认对话框" @click="$emit('cancel')"></button>
      <section class="confirm-dialog" role="alertdialog" aria-modal="true" aria-labelledby="confirm-title" aria-describedby="confirm-description">
        <span class="dialog-mark" aria-hidden="true">!</span>
        <h2 id="confirm-title">{{ title }}</h2>
        <p id="confirm-description">{{ description }}</p>
        <div class="dialog-actions">
          <button class="button button--quiet" type="button" :disabled="busy" @click="$emit('cancel')">取消</button>
          <button class="button button--danger" type="button" :disabled="busy" autofocus @click="$emit('confirm')">{{ busy ? '删除中…' : '确认删除' }}</button>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<style scoped>
.dialog-layer { position: fixed; z-index: 80; inset: 0; display: grid; place-items: center; padding: 20px; }
.dialog-backdrop { position: absolute; inset: 0; border: 0; background: rgba(3,5,8,.76); backdrop-filter: blur(6px); }
.confirm-dialog { position: relative; width: min(100%, 400px); padding: 26px; border: 1px solid var(--line-strong); border-radius: var(--radius-lg); background: var(--surface-1); box-shadow: var(--shadow-lg); text-align: center; }
.dialog-mark { display: grid; place-items: center; width: 42px; height: 42px; margin: 0 auto 15px; border-radius: 50%; color: #ffc1bd; background: rgba(255,105,97,.14); font: 800 18px var(--display); }
h2 { margin: 0; color: var(--text); font: 650 18px var(--display); }p { margin: 10px 0 22px; color: var(--text-soft); font-size: 12px; line-height: 1.6; }
.dialog-actions { display: flex; justify-content: center; gap: 9px; }
</style>
