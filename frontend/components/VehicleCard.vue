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
  }
}>()
const img = computed(() => props.item.image_url || '')
const hasDiscount = computed(() => props.item.discount_percent > 0)

const statusCls: Record<string, string> = {
  on_sale: 'bg-green-100 text-green-700',
  upcoming: 'bg-amber-100 text-amber-700',
  sold_out: 'bg-red-100 text-red-600',
  hidden: 'bg-slate-100 text-slate-500',
}
</script>

<template>
  <NuxtLink :to="`/vehicles/${item.id}`" class="card group block overflow-hidden transition hover:shadow-md">
    <div class="relative aspect-[16/10] bg-gradient-to-b from-slate-50 to-slate-100">
      <img v-if="img" :src="img" :alt="item.name" class="h-full w-full object-contain p-2" />
      <span
        v-if="hasDiscount"
        class="badge absolute left-2 top-2 bg-gold-500 text-brand-950 shadow"
      >-{{ Math.round(item.discount_percent) }}%</span>
    </div>
    <div class="p-3">
      <div class="flex items-center justify-between gap-2">
        <div class="truncate text-xs uppercase tracking-wide text-brand-500">{{ item.brand || '—' }} · {{ item.class }}</div>
        <span class="badge shrink-0 !px-2 !py-0.5 text-[10px]" :class="statusCls[item.status] || 'bg-slate-100 text-slate-500'">
          {{ vehicleStatusLabel(item.status) }}
        </span>
      </div>
      <h3 class="truncate font-semibold text-slate-800 group-hover:text-brand-800">{{ item.name }}</h3>
      <div class="mt-2 flex items-end justify-between">
        <div>
          <div v-if="hasDiscount" class="text-xs text-slate-400 line-through">{{ formatMoney(item.base_price) }}</div>
          <div class="font-bold text-brand-800">{{ formatMoney(item.final_price) }}</div>
        </div>
      </div>
    </div>
  </NuxtLink>
</template>
