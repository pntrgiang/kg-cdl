<script setup lang="ts">
const api = useApi()
const { data: items, pending } = await useAsyncData('vehicles-onsale', () =>
  api.get<any[]>('/api/vehicles?status=on_sale'),
)
const search = ref('')
const filtered = computed(() => {
  const q = search.value.toLowerCase().trim()
  const list = items.value || []
  if (!q) return list
  return list.filter((v) => `${v.name} ${v.brand} ${v.class}`.toLowerCase().includes(q))
})
</script>

<template>
  <section>
    <HomeBanner />

    <div class="mb-4 flex items-center justify-between gap-4">
      <h2 class="text-lg font-semibold">Xe đang mở bán</h2>
      <input v-model="search" class="input max-w-xs" placeholder="Tìm theo tên, hãng…" />
    </div>

    <div v-if="pending" class="py-12 text-center text-slate-400">Đang tải…</div>
    <div v-else-if="!filtered.length" class="card p-12 text-center text-slate-400">
      Chưa có xe nào đang mở bán.
    </div>
    <div v-else class="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
      <VehicleCard v-for="v in filtered" :key="v.id" :item="v" />
    </div>
  </section>
</template>
