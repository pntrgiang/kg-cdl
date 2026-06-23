<script setup lang="ts">
// Ảnh xe có phóng to / rê xem chi tiết. Dùng ảnh render chính thức (đúng xe 100%).
const props = defineProps<{ src: string; alt?: string }>()

const zoom = ref(false)
const ox = ref(50)
const oy = ref(50)
const SCALE = 2.4

function track(e: MouseEvent | TouchEvent) {
  const el = e.currentTarget as HTMLElement
  const r = el.getBoundingClientRect()
  const p = 'touches' in e ? e.touches[0] : (e as MouseEvent)
  ox.value = Math.min(100, Math.max(0, ((p.clientX - r.left) / r.width) * 100))
  oy.value = Math.min(100, Math.max(0, ((p.clientY - r.top) / r.height) * 100))
}
</script>

<template>
  <div
    class="relative aspect-[16/10] select-none overflow-hidden bg-gradient-to-b from-slate-50 to-slate-100"
    :class="zoom ? 'cursor-zoom-out' : 'cursor-zoom-in'"
    @mouseenter="zoom = true"
    @mouseleave="zoom = false"
    @mousemove="track"
    @touchstart.prevent="zoom = !zoom"
    @touchmove.prevent="track"
  >
    <img
      v-if="src"
      :src="src"
      :alt="alt"
      draggable="false"
      class="h-full w-full object-contain p-3 transition-transform duration-200 will-change-transform"
      :style="zoom ? { transform: `scale(${SCALE})`, transformOrigin: `${ox}% ${oy}%` } : {}"
    />
    <span class="pointer-events-none absolute bottom-2 right-2 rounded bg-black/45 px-2 py-0.5 text-[11px] text-white">
      {{ zoom ? '🔍 Di chuyển để xem chi tiết' : '🔍 Rê chuột / chạm để phóng to' }}
    </span>
  </div>
</template>
