<script setup lang="ts">
const props = defineProps<{ rank: string }>()
const map: Record<string, { label: string; cls: string }> = {
  svip: { label: 'SVIP', cls: 'bg-gold-500 text-brand-950' },
  vip: { label: 'VIP', cls: 'bg-brand-200 text-brand-900' },
  regular: { label: 'Phổ thông', cls: 'bg-slate-200 text-slate-600' },
}
const info = computed(() => map[props.rank] || map.regular)
</script>

<template>
  <span class="badge" :class="info.cls">{{ info.label }}</span>
</template>
