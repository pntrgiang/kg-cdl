<script setup lang="ts">
definePageMeta({ layout: 'staff', middleware: 'staff' })
const api = useApi()
const auth = useAuthStore()

const { data: inventory } = await useAsyncData('st-inv', () => api.get<any[]>('/api/inventory'))
const { data: customers } = await useAsyncData('st-cust', () => api.get<any[]>('/api/customers'))
const { data: sales } = await useAsyncData('st-sales', () => api.get<any[]>('/api/sales?limit=500'))

const stats = computed(() => {
  const inv = inventory.value || []
  const sl = sales.value || []
  return {
    onSale: inv.filter((i) => i.status === 'on_sale').length,
    stock: inv.reduce((a, i) => a + i.quantity, 0),
    customers: (customers.value || []).length,
    revenue: sl.reduce((a, s) => a + s.final_price, 0),
    sales: sl.length,
  }
})
</script>

<template>
  <div>
    <h1 class="mb-1 font-serif text-2xl font-bold text-brand-900">Xin chào, {{ auth.displayName }} 👋</h1>
    <p class="mb-6 text-slate-500">Tổng quan hoạt động đại lý.</p>

    <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
      <div class="card p-4"><div class="text-xs text-slate-500">Xe đang bán</div><div class="text-2xl font-bold text-brand-800">{{ stats.onSale }}</div></div>
      <div class="card p-4"><div class="text-xs text-slate-500">Tồn kho</div><div class="text-2xl font-bold text-brand-800">{{ stats.stock }}</div></div>
      <div class="card p-4"><div class="text-xs text-slate-500">Khách hàng</div><div class="text-2xl font-bold text-brand-800">{{ stats.customers }}</div></div>
      <div class="card p-4"><div class="text-xs text-slate-500">Doanh thu ({{ stats.sales }} đơn)</div><div class="text-xl font-bold text-gold-600">{{ formatMoney(stats.revenue) }}</div></div>
    </div>

    <div class="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <NuxtLink to="/staff/sell" class="card p-5 transition hover:shadow-md">💰 <span class="font-semibold">Bán xe</span><p class="text-sm text-slate-500">Tạo giao dịch cho khách</p></NuxtLink>
      <NuxtLink to="/staff/inventory" class="card p-5 transition hover:shadow-md">📦 <span class="font-semibold">Nhập kho</span><p class="text-sm text-slate-500">Thêm xe, đặt giảm giá</p></NuxtLink>
      <NuxtLink to="/staff/customers" class="card p-5 transition hover:shadow-md">👥 <span class="font-semibold">Khách hàng</span><p class="text-sm text-slate-500">Quản lý danh sách khách</p></NuxtLink>
    </div>
  </div>
</template>
