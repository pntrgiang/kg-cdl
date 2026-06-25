<script setup lang="ts">
const api = useApi()
const { data: items, pending } = await useAsyncData('vehicles-onsale', () =>
  api.get<any[]>('/api/vehicles?status=on_sale'),
)
useSeo({
  title: 'Xe đang mở bán',
  description:
    'Danh sách xe đang mở bán tại đại lý Kanji Group — Lux City: giá tốt, ưu đãi giảm giá, thông số chi tiết và đặt lịch xem xe nhanh chóng.',
})

const search = ref('')
const filtered = computed(() => {
  const q = search.value.toLowerCase().trim()
  const list = items.value || []
  if (!q) return list
  return list.filter((v) => `${v.name} ${v.brand} ${v.class}`.toLowerCase().includes(q))
})

// ── Sắp xếp: theo giá & theo thời gian mở bán ──
const sortBy = ref<'default' | 'price_asc' | 'price_desc' | 'newest' | 'oldest'>('default')
const releaseTs = (v: any) => new Date(v.on_sale_at || v.created_at || 0).getTime()
const sorted = computed(() => {
  const list = [...filtered.value]
  switch (sortBy.value) {
    case 'price_asc':
      return list.sort((a, b) => a.final_price - b.final_price)
    case 'price_desc':
      return list.sort((a, b) => b.final_price - a.final_price)
    case 'newest':
      return list.sort((a, b) => releaseTs(b) - releaseTs(a))
    case 'oldest':
      return list.sort((a, b) => releaseTs(a) - releaseTs(b))
    default:
      return list
  }
})
</script>

<template>
  <section>
    <h1 class="sr-only">Kanji Group — Đại lý xe Lux City: xe đang mở bán, giá tốt &amp; ưu đãi</h1>
    <HomeBanner />

    <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
      <h2 class="text-lg font-semibold">Xe đang mở bán</h2>
      <div class="flex items-center gap-2">
        <select v-model="sortBy" class="input sort-select !w-auto shrink-0" aria-label="Sắp xếp xe">
          <option value="default">Sắp xếp: Mặc định</option>
          <option value="price_asc">Giá: thấp → cao</option>
          <option value="price_desc">Giá: cao → thấp</option>
          <option value="newest">Mở bán: mới nhất</option>
          <option value="oldest">Mở bán: cũ nhất</option>
        </select>
        <input v-model="search" class="input !w-40 sm:!w-52" placeholder="Tìm theo tên, hãng…" />
      </div>
    </div>

    <div v-if="pending" class="py-12 text-center text-slate-400">Đang tải…</div>
    <div v-else-if="!sorted.length" class="card p-12 text-center text-slate-400">
      Chưa có xe nào đang mở bán.
    </div>
    <div v-else class="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
      <VehicleCard v-for="v in sorted" :key="v.id" :item="v" />
    </div>
  </section>
</template>

<style scoped>
/* Thay mũi tên select mặc định bằng chevron SVG -> canh đều khoảng cách bên phải. */
.sort-select {
  appearance: none;
  -webkit-appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='14' height='14' viewBox='0 0 24 24' fill='none' stroke='%236b7280' stroke-width='2.2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M6 9l6 6 6-6'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 0.75rem center;
  background-size: 14px;
  padding-right: 2.25rem;
}
</style>
