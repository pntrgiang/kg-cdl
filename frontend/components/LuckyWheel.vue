<script setup lang="ts">
// Vòng quay may mắn: mỗi khách = 1 ô màu. Gọi spin(winnerId) để quay tới đúng ô người trúng.
const props = defineProps<{ entries: { customer_id: number; customer_name: string }[] }>()

const CX = 180, CY = 180, R = 172
const COLORS = ['#ef4444', '#3b82f6', '#22c55e', '#f59e0b', '#a855f7', '#ec4899', '#14b8a6', '#f97316']

const rotation = ref(0)
const spinning = ref(false)
const SPIN_MS = 4500

const n = computed(() => props.entries.length)
const seg = computed(() => 360 / Math.max(1, n.value))
const fontSize = computed(() => {
  const c = n.value
  if (c <= 8) return 13
  if (c <= 16) return 11
  if (c <= 28) return 9
  return 7
})
function clip(name: string) {
  const max = n.value <= 10 ? 14 : n.value <= 20 ? 9 : 6
  return name.length > max ? name.slice(0, max - 1) + '…' : name
}

function slicePath(i: number) {
  if (n.value === 1) return '' // 1 người -> vẽ full circle bằng <circle> riêng
  const a0 = (i * seg.value) * Math.PI / 180
  const a1 = ((i + 1) * seg.value) * Math.PI / 180
  const x0 = CX + R * Math.cos(a0), y0 = CY + R * Math.sin(a0)
  const x1 = CX + R * Math.cos(a1), y1 = CY + R * Math.sin(a1)
  const large = seg.value > 180 ? 1 : 0
  return `M${CX},${CY} L${x0.toFixed(2)},${y0.toFixed(2)} A${R},${R} 0 ${large} 1 ${x1.toFixed(2)},${y1.toFixed(2)} Z`
}
function labelPos(i: number) {
  const a = (i + 0.5) * seg.value
  return { x: CX + R * 0.62, y: CY, rot: a }
}

async function spin(winnerId: number): Promise<void> {
  if (spinning.value || !n.value) return
  const idx = props.entries.findIndex((e) => e.customer_id === winnerId)
  const target = idx >= 0 ? idx : 0
  const segCenter = (target + 0.5) * seg.value // góc tâm ô (0° = hướng Đông, nơi đặt kim)
  const base = rotation.value - (rotation.value % 360)
  rotation.value = base + 360 * 7 + (360 - segCenter) // quay 7 vòng rồi dừng đúng ô
  spinning.value = true
  await new Promise((r) => setTimeout(r, SPIN_MS + 150))
  spinning.value = false
}
defineExpose({ spin, spinning })
</script>

<template>
  <div class="relative mx-auto w-full" style="max-width: 360px">
    <svg viewBox="0 0 360 360" class="w-full drop-shadow">
      <g :style="{ transform: `rotate(${rotation}deg)`, transformOrigin: '180px 180px', transition: spinning ? `transform ${SPIN_MS}ms cubic-bezier(.15,.7,.15,1)` : 'none' }">
        <template v-if="n === 1">
          <circle :cx="CX" :cy="CY" :r="R" :fill="COLORS[0]" stroke="#fff" stroke-width="2" />
          <text :x="CX" :y="CY - R * 0.5" text-anchor="middle" dominant-baseline="middle" fill="#fff" :font-size="14" font-weight="700">{{ clip(entries[0].customer_name) }}</text>
        </template>
        <template v-else>
          <g v-for="(e, i) in entries" :key="e.customer_id">
            <path :d="slicePath(i)" :fill="COLORS[i % COLORS.length]" stroke="#ffffff" stroke-width="1.5" />
            <text
              :x="labelPos(i).x" :y="labelPos(i).y"
              :transform="`rotate(${labelPos(i).rot}, ${CX}, ${CY})`"
              text-anchor="middle" dominant-baseline="middle"
              fill="#ffffff" :font-size="fontSize" font-weight="700"
            >{{ clip(e.customer_name) }}</text>
          </g>
        </template>
      </g>
      <!-- trục giữa -->
      <circle :cx="CX" :cy="CY" r="26" fill="#fff" stroke="#e5e7eb" stroke-width="2" />
      <text :x="CX" :y="CY" text-anchor="middle" dominant-baseline="central" font-size="20">🎯</text>
    </svg>
    <!-- kim chỉ ở bên phải, trỏ vào trong (hướng Đông) -->
    <div class="pointer-events-none absolute right-[-10px] top-1/2 -translate-y-1/2">
      <div class="h-0 w-0 border-y-[12px] border-r-[20px] border-y-transparent border-r-gold-500 drop-shadow"></div>
    </div>
  </div>
</template>
