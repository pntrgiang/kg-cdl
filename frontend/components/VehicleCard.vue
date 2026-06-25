<script setup lang="ts">
const props = defineProps<{
  item: {
    id: number
    name: string
    brand: string
    class: string
    image_url: string
    base_price: number
    final_price: number
    discount_percent: number
    quantity: number
    status: string
    seats?: number | null
    trunk_kg?: number
  }
}>()
const img = computed(() => props.item.image_url || '')
const hasDiscount = computed(() => props.item.discount_percent > 0)
const sub = computed(() => `${props.item.brand || '—'} · ${props.item.class || '—'}`)
const seatsLabel = computed(() => `${props.item.seats ?? '—'} ghế`)
const trunkLabel = computed(() => `${props.item.trunk_kg ?? 10} kg`)

// Trạng thái + màu badge theo template option_1:
// đang bán & tồn = 0 -> Hết hàng; tồn < 5 -> Sắp hết hàng; còn lại -> Đang mở bán.
const STATUS = {
  open: { label: 'Đang mở bán', bg: '#6d4fd8', color: '#ffffff' },
  low: { label: 'Sắp hết hàng', bg: '#ec4256', color: '#ffffff' },
  out: { label: 'Hết hàng', bg: '#e7e8ec', color: '#6b6e76' },
  soon: { label: 'Sắp mở bán', bg: '#f59e0b', color: '#ffffff' },
  hidden: { label: 'Đang ẩn', bg: '#e7e8ec', color: '#6b6e76' },
}
const badge = computed(() => {
  const i = props.item
  if (i.status === 'upcoming') return STATUS.soon
  if (i.status === 'sold_out') return STATUS.out
  if (i.status === 'hidden') return STATUS.hidden
  if (i.quantity <= 0) return STATUS.out
  if (i.quantity < 5) return STATUS.low
  return STATUS.open
})
// Xe "Sắp hết hàng" -> hover hiện viền lửa tím (flame) thay vì viền tím phẳng.
const isLow = computed(() => badge.value === STATUS.low)
</script>

<template>
  <NuxtLink :to="`/vehicles/${item.id}`" class="vc" :class="{ 'vc-fire': isLow }">
    <div class="vc-inner" :class="isLow ? 'vc-inner-fire' : 'vc-inner-normal'">
      <div class="truncate" style="font-size:17px;font-weight:700;letter-spacing:-0.01em;color:#1c1f24;">{{ item.name }}</div>
      <div class="truncate" style="font-size:13px;font-weight:500;color:#a2a7af;margin-top:3px;">{{ sub }}</div>

      <div style="height:118px;display:flex;align-items:center;justify-content:center;margin:24px 0 26px;">
        <img v-if="img" :src="img" :alt="`${item.name} — ${item.brand}`" loading="lazy" decoding="async" style="width:100%;height:100%;object-fit:contain;border-radius:8px;" />
        <span v-else style="font-size:32px;">🚗</span>
      </div>

      <div style="display:grid;grid-template-columns:1fr 1fr;gap:8px;margin-bottom:12px;">
        <div style="display:flex;align-items:center;gap:8px;font-size:14px;font-weight:500;color:#4a4d54;">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#a2a7af" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M4 11V8.5A2.5 2.5 0 0 1 6.5 6h11A2.5 2.5 0 0 1 20 8.5V11"></path><path d="M3 12.5A1.5 1.5 0 0 1 4.5 11 1.5 1.5 0 0 1 6 12.5V15h12v-2.5A1.5 1.5 0 0 1 19.5 11 1.5 1.5 0 0 1 21 12.5V17a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1z"></path><path d="M6 18v1.5M18 18v1.5"></path></svg>
          <span>{{ seatsLabel }}</span>
        </div>
        <div style="display:flex;align-items:center;gap:8px;font-size:14px;font-weight:500;color:#4a4d54;">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#a2a7af" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="5" r="2.6"></circle><path d="M7 8.2h10a1.6 1.6 0 0 1 1.55 1.2l2 8A1.6 1.6 0 0 1 19 19.4H5a1.6 1.6 0 0 1-1.55-2l2-8A1.6 1.6 0 0 1 7 8.2Z"></path></svg>
          <span>{{ trunkLabel }}</span>
        </div>
      </div>

      <div style="display:flex;align-items:center;justify-content:space-between;gap:8px;">
        <span style="font-size:15px;font-weight:700;color:#1c1f24;">
          <span v-if="hasDiscount" style="font-size:12px;font-weight:500;color:#a2a7af;text-decoration:line-through;margin-right:6px;">{{ formatMoney(item.base_price) }}</span>
          {{ formatMoney(item.final_price) }}
        </span>
        <span style="font-size:11px;font-weight:600;padding:3px 9px;border-radius:7px;white-space:nowrap;" :style="{ background: badge.bg, color: badge.color }">{{ badge.label }}</span>
      </div>
    </div>
  </NuxtLink>
</template>

<style scoped>
.vc {
  position: relative;
  display: block;
  transition: transform 0.2s ease;
}
.vc:hover {
  transform: translateY(-4px);
}
.vc-inner {
  position: relative;
  z-index: 1;
  background: #ffffff;
  border: 1px solid #ededf1;
  border-radius: 14px;
  padding: 26px 18px 24px;
  box-shadow: 0 1px 2px rgba(20, 23, 28, 0.04);
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}
/* Card thường: hover viền tím + bóng tím nhẹ */
.vc:hover .vc-inner-normal {
  border-color: #6d4fd8;
  box-shadow: 0 10px 28px -8px rgba(109, 79, 216, 0.28);
}
/* Card "Sắp hết hàng": hover viền sáng tím + quầng nóng */
.vc:hover .vc-inner-fire {
  border-color: #a855f7;
  box-shadow: 0 0 16px -2px rgba(168, 85, 247, 0.5);
}

/* ── Viền lửa tím (chỉ card sắp hết hàng, khi hover) ──
   2 lớp gradient bị bộ lọc #vc-flame (nhiễu fractal động) làm méo -> lưỡi lửa uốn lượn. */
.vc-fire::before,
.vc-fire::after {
  content: '';
  position: absolute;
  inset: -5px;
  border-radius: 19px;
  z-index: 0;
  opacity: 0;
  pointer-events: none;
  filter: blur(6px);
  transition: opacity 0.25s ease;
}
/* lớp lưỡi lửa: nhiều ngọn sáng (tip gần trắng tím) chen nhau */
.vc-fire::before {
  background: conic-gradient(from 0deg,
    #4c1d95, #7c3aed, #c084fc, #e9d5ff, #a855f7, #7c3aed,
    #c084fc, #f3e8ff, #a855f7, #7c3aed, #c084fc, #4c1d95);
}
/* lớp quầng nóng phía ngoài */
.vc-fire::after {
  background: conic-gradient(from 90deg, #6d28d9, #a855f7, #d8b4fe, #a855f7, #7c3aed, #6d28d9);
}
.vc-fire:hover::before {
  opacity: 0.95;
  filter: url(#vc-flame) blur(2.5px);
  animation: vcFireA 1.3s steps(6, end) infinite;
}
.vc-fire:hover::after {
  opacity: 0.6;
  filter: url(#vc-flame) blur(9px);
  animation: vcFireB 1.8s steps(5, end) infinite;
}
/* flicker độ sáng/độ phồng như ngọn lửa lay động */
@keyframes vcFireA {
  0%   { transform: scale(1);     opacity: 0.85; }
  20%  { transform: scale(1.03);  opacity: 1; }
  40%  { transform: scale(0.99);  opacity: 0.8; }
  60%  { transform: scale(1.04);  opacity: 1; }
  80%  { transform: scale(1.01);  opacity: 0.9; }
  100% { transform: scale(1);     opacity: 0.85; }
}
@keyframes vcFireB {
  0%   { transform: scale(1);     opacity: 0.55; }
  35%  { transform: scale(1.05);  opacity: 0.75; }
  70%  { transform: scale(0.98);  opacity: 0.5; }
  100% { transform: scale(1);     opacity: 0.55; }
}
@media (prefers-reduced-motion: reduce) {
  .vc-fire:hover::before,
  .vc-fire:hover::after {
    animation: none;
  }
}
</style>
