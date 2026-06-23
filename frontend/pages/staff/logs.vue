<script setup lang="ts">
definePageMeta({ layout: 'staff', middleware: 'staff' })
const api = useApi()

const PAGE = 100
const action = ref('')
const page = ref(1) // 1-based

const { data: actions } = await useAsyncData('log-actions', () => api.get<string[]>('/api/logs/actions'))
const { data: logData, refresh } = await useAsyncData('logs', () => {
  const params = new URLSearchParams()
  if (action.value) params.set('action', action.value)
  params.set('limit', String(PAGE))
  params.set('offset', String((page.value - 1) * PAGE))
  return api.get<{ items: any[]; total: number }>(`/api/logs?${params.toString()}`)
})
watch(action, () => { page.value = 1; refresh() })
watch(page, () => refresh())

const logs = computed(() => logData.value?.items || [])
const total = computed(() => logData.value?.total || 0)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / PAGE)))

const actionLabel: Record<string, string> = {
  'sale.create': '💰 Bán xe',
  'inventory.add': '📦 Nhập kho',
  'inventory.discount': '🏷️ Giảm giá',
  'inventory.status': '🔁 Đổi trạng thái',
  'catalog.create': '🚗 Tạo mẫu xe',
  'customer.create': '➕ Tạo khách',
  'customer.update': '✏️ Sửa khách',
  'event.create': '🎉 Tạo sự kiện',
  'draw.run': '🎰 Quay số',
  'draw.redraw': '🔄 Quay số lại',
  'draw.confirm': '🏆 Công bố trúng thưởng',
  'voucher.create': '🎟️ Tạo voucher',
  'catalog.update': '✏️ Sửa thông tin xe',
  'settings.rank_limits': '⚙️ Đổi giới hạn hạng',
  'user.create': '👤 Tạo nhân viên',
  'user.role': '⚙️ Đổi quyền',
  'user.password': '🔑 Đổi mật khẩu',
  'user.delete': '🗑️ Xoá nhân viên',
  'user.deactivate': '🚫 Vô hiệu hoá nhân viên',
}
const fmt = (d: string) => formatDateTime(d)
function detail(d: any) {
  if (!d) return ''
  try { return Object.entries(typeof d === 'string' ? JSON.parse(d) : d).map(([k, v]) => `${k}: ${v}`).join(', ') }
  catch { return '' }
}
</script>

<template>
  <div>
    <div class="mb-4 flex items-center justify-between">
      <h1 class="font-serif text-2xl font-bold text-brand-900">📜 Nhật ký hoạt động</h1>
      <select v-model="action" class="input max-w-xs">
        <option value="">Tất cả hành động</option>
        <option v-for="a in actions" :key="a" :value="a">{{ actionLabel[a] || a }}</option>
      </select>
    </div>

    <div class="card overflow-x-auto">
      <table class="w-full min-w-[640px] text-sm">
        <thead class="bg-slate-50 text-left text-xs uppercase text-slate-500">
          <tr><th class="p-3">Thời gian</th><th class="p-3">Người thực hiện</th><th class="p-3">Hành động</th><th class="p-3">Chi tiết</th></tr>
        </thead>
        <tbody>
          <tr v-for="l in logs" :key="l.id" class="border-t">
            <td class="p-3 whitespace-nowrap text-slate-500">{{ fmt(l.created_at) }}</td>
            <td class="p-3">{{ l.actor_name || '—' }}</td>
            <td class="p-3"><span class="badge bg-brand-100 text-brand-800">{{ actionLabel[l.action] || l.action }}</span></td>
            <td class="p-3 text-slate-600">{{ detail(l.detail) }}</td>
          </tr>
          <tr v-if="!logs.length"><td colspan="4" class="p-8 text-center text-slate-400">Chưa có nhật ký.</td></tr>
        </tbody>
      </table>
    </div>

    <!-- phân trang (tối đa 100/trang) -->
    <div v-if="total > 0" class="mt-4 flex flex-wrap items-center justify-between gap-2 text-sm">
      <span class="text-slate-500">
        Tổng {{ total }} bản ghi · Trang {{ page }}/{{ totalPages }}
      </span>
      <div class="flex gap-2">
        <button class="btn-ghost !py-1.5 text-xs" :disabled="page <= 1" @click="page--">← Trước</button>
        <button class="btn-ghost !py-1.5 text-xs" :disabled="page >= totalPages" @click="page++">Sau →</button>
      </div>
    </div>
  </div>
</template>
