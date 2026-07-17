<script setup lang="ts">
defineProps<{
  label: string
  value: string | number
  detail: string
  tone?: 'neutral' | 'success' | 'warning'
}>()
</script>

<template>
  <article class="metric-tile" :class="`metric-tile--${tone ?? 'neutral'}`">
    <div class="metric-tile__top">
      <span class="metric-tile__label">{{ label }}</span>
      <span class="metric-tile__signal" aria-hidden="true"></span>
    </div>
    <strong>{{ value }}</strong>
    <p>{{ detail }}</p>
  </article>
</template>

<style scoped>
.metric-tile {
  min-width: 0;
  padding: 18px 20px;
  border: 1px solid var(--line);
  border-radius: var(--radius-lg);
  background: linear-gradient(145deg, rgba(255,255,255,.042), rgba(255,255,255,.012));
  box-shadow: var(--shadow-sm);
}
.metric-tile__top { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.metric-tile__label { color: var(--muted); font-size: 12px; font-weight: 650; letter-spacing: .08em; text-transform: uppercase; }
.metric-tile__signal { width: 7px; height: 7px; border-radius: 50%; background: var(--text-soft); box-shadow: 0 0 0 4px rgba(164,174,190,.08); }
.metric-tile--success .metric-tile__signal { background: var(--mint); box-shadow: 0 0 0 4px rgba(80,217,169,.1); }
.metric-tile--warning .metric-tile__signal { background: var(--amber); box-shadow: 0 0 0 4px rgba(240,176,84,.1); }
strong { display: block; margin-top: 15px; color: var(--text); font: 650 clamp(28px, 3vw, 38px)/1 var(--display); letter-spacing: -.04em; }
p { margin: 8px 0 0; color: var(--text-soft); font-size: 12px; line-height: 1.45; }
</style>
