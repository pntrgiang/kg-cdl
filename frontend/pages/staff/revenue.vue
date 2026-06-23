<script setup lang="ts">
definePageMeta({ layout: 'staff', middleware: 'staff' })
const api = useApi()
const auth = useAuthStore()
const { data: rep, pending, refresh } = await useAsyncData('revenue', () => api.get<any>('/api/reports/revenue'))

// ── hoàn trả (chỉ quản lý): hoàn xe bị bán sai ──
const refundFor = ref<number | null>(null)
const refundReason = ref('')
const refundMsg = ref('')
function openRefund(it: any) { refundFor.value = it.id; refundReason.value = ''; refundMsg.value = '' }
async function doRefund(id: number) {
  refundMsg.value = ''
  if (!refundReason.value.trim()) { refundMsg.value = 'Bắt buộc nhập lý do hoàn trả.'; return }
  try {
    await api.post(`/api/sales/${id}/refund`, { reason: refundReason.value.trim() })
    refundFor.value = null
    await refresh()
  } catch (e: any) { refundMsg.value = e?.data?.error || 'Hoàn trả thất bại.' }
}

const weeks = computed(() => rep.value?.weeks || [])
const maxWeek = computed(() => Math.max(1, ...weeks.value.map((w: any) => w.revenue)))
const topVehicles = computed(() => rep.value?.top_vehicles || [])
const topCustomers = computed(() => rep.value?.top_customers || [])

// lọc theo tuần
const weekFilter = ref('all')
const shownWeeks = computed(() =>
  weekFilter.value === 'all' ? weeks.value : weeks.value.filter((w: any) => w.week_start === weekFilter.value),
)
const filteredTotal = computed(() => shownWeeks.value.reduce((a: number, w: any) => ({ sales: a.sales + w.sales, revenue: a.revenue + w.revenue }), { sales: 0, revenue: 0 }))

// Lợi nhuận thực tế: doanh nghiệp chỉ nhận 10% doanh thu (90% còn lại là chi phí).
const PROFIT_RATE = 0.1
const profit = (revenue: number) => revenue * PROFIT_RATE

function d(s: string) { const [, m, dd] = s.split('-'); return `${dd}/${m}` }
const dt = (s: string) => formatDateTime(s)

// mặc định mở tuần mới nhất
const open = reactive<Record<string, boolean>>({})
watchEffect(() => { if (weeks.value.length && Object.keys(open).length === 0) open[weeks.value[0].week_start] = true })

// ── xuất CSV chi tiết các xe đã bán ──
function csvCell(v: any): string {
  const s = String(v ?? '')
  return /[",\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s
}
function exportCsv() {
  const header = ['Tuần', 'Thời gian', 'Xe', 'Khách hàng', 'Nhân viên bán', 'Giá gốc', 'Giảm %', 'Giảm voucher', 'Thành tiền', 'Đã hoàn', 'Lý do hoàn']
  const rows: string[] = [header.join(',')]
  for (const w of shownWeeks.value) {
    const wlabel = `${d(w.week_start)}-${d(w.week_end)}`
    for (const it of w.items) {
      rows.push([
        wlabel, dt(it.created_at), it.vehicle_name, it.customer_name, it.sold_by_name,
        Math.round(it.original_price), it.discount_percent, Math.round(it.voucher_discount),
        Math.round(it.final_price), it.refunded ? 'Có' : '', it.refund_reason || '',
      ].map(csvCell).join(','))
    }
  }
  // BOM để Excel đọc đúng tiếng Việt
  const blob = new Blob(['﻿' + rows.join('\n')], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `doanh-thu-${new Date().toISOString().slice(0, 10)}.csv`
  a.click()
  URL.revokeObjectURL(url)
}
</script>

<template>
  <div>
    <div class="mb-1 flex flex-wrap items-center justify-between gap-2">
      <h1 class="font-serif text-2xl font-bold text-brand-900">📈 Doanh thu theo tuần</h1>
      <button class="btn-ghost !py-1.5 text-sm" :disabled="!weeks.length" @click="exportCsv">⬇️ Xuất CSV</button>
    </div>
    <p class="mb-6 text-sm text-slate-500">Xe mở bán theo tuần; tồn kho tuần trước vẫn tính vào tuần được bán.</p>

    <div v-if="pending" class="py-12 text-center text-slate-400">Đang tải…</div>
    <template v-else-if="rep">
      <!-- tổng quan -->
      <div class="grid grid-cols-2 gap-4 lg:grid-cols-3">
        <div class="card p-4"><div class="text-xs text-slate-500">Tổng doanh thu</div><div class="text-xl font-bold text-gold-600">{{ formatMoney(rep.summary.revenue) }}</div></div>
        <div class="card border-green-200 bg-green-50 p-4" title="Doanh nghiệp nhận 10% doanh thu; 90% còn lại là chi phí">
          <div class="text-xs font-medium text-green-700">💰 Lợi nhuận thực tế (10%)</div>
          <div class="text-xl font-bold text-green-700">{{ formatMoney(profit(rep.summary.revenue)) }}</div>
        </div>
        <div class="card p-4"><div class="text-xs text-slate-500">Số đơn bán</div><div class="text-2xl font-bold text-brand-800">{{ rep.summary.sales }}</div></div>
        <div class="card p-4"><div class="text-xs text-slate-500">Trung bình / đơn</div><div class="text-xl font-bold text-brand-800">{{ formatMoney(rep.summary.avg_sale) }}</div></div>
        <div class="card p-4"><div class="text-xs text-slate-500">Voucher đã dùng</div><div class="text-2xl font-bold text-brand-800">{{ rep.summary.voucher_uses }} <span class="text-sm font-normal text-slate-400">lượt</span></div></div>
        <div class="card p-4"><div class="text-xs text-slate-500">Tổng giảm từ voucher</div><div class="text-lg font-bold text-red-600">- {{ formatMoney(rep.summary.voucher_total) }}</div></div>
      </div>

      <!-- biểu đồ tuần -->
      <div class="card mt-6 p-5">
        <h2 class="mb-4 font-semibold text-brand-900">Doanh thu các tuần</h2>
        <div v-if="weeks.length" class="flex h-40 items-end gap-3 overflow-x-auto">
          <div v-for="w in [...weeks].reverse()" :key="w.week_start" class="flex min-w-[56px] flex-1 flex-col items-center gap-1" :title="`${w.week_start} → ${w.week_end}: ${formatMoney(w.revenue)}`">
            <div class="text-[10px] font-medium text-brand-700">{{ formatMoney(w.revenue) }}</div>
            <div class="flex w-full items-end" style="height: 110px">
              <div class="w-full rounded-t bg-gradient-to-t from-brand-700 to-gold-500" :style="{ height: Math.max(3, (w.revenue / maxWeek) * 110) + 'px' }" />
            </div>
            <div class="whitespace-nowrap text-[10px] text-slate-400">{{ d(w.week_start) }}–{{ d(w.week_end) }}</div>
          </div>
        </div>
        <p v-else class="py-8 text-center text-sm text-slate-400">Chưa có giao dịch nào.</p>
      </div>

      <!-- chi tiết theo tuần -->
      <div class="mb-2 mt-6 flex flex-wrap items-center justify-between gap-2">
        <h2 class="font-semibold text-brand-900">Chi tiết xe đã bán theo tuần</h2>
        <div class="flex items-center gap-2">
          <label class="text-sm text-slate-500">Lọc theo tuần:</label>
          <select v-model="weekFilter" class="input max-w-[220px] !py-1.5 text-sm">
            <option value="all">Tất cả tuần</option>
            <option v-for="w in weeks" :key="w.week_start" :value="w.week_start">Tuần {{ d(w.week_start) }} – {{ d(w.week_end) }}</option>
          </select>
        </div>
      </div>
      <p v-if="weekFilter !== 'all'" class="mb-2 text-sm text-slate-600">
        Tuần đã chọn: <strong>{{ filteredTotal.sales }}</strong> đơn ·
        doanh thu <strong class="text-gold-600">{{ formatMoney(filteredTotal.revenue) }}</strong> ·
        lợi nhuận thực tế <strong class="text-green-700">{{ formatMoney(profit(filteredTotal.revenue)) }}</strong>
      </p>
      <div class="space-y-3">
        <div v-for="w in shownWeeks" :key="w.week_start" class="card overflow-hidden">
          <button class="flex w-full items-center justify-between gap-3 p-4 text-left hover:bg-slate-50" @click="open[w.week_start] = !open[w.week_start]">
            <div>
              <span class="font-semibold text-brand-900">Tuần {{ d(w.week_start) }} – {{ d(w.week_end) }}</span>
              <span class="ml-2 text-sm text-slate-500">({{ w.sales }} đơn)</span>
            </div>
            <div class="flex items-center gap-2">
              <div class="text-right">
                <div class="font-bold text-gold-600">{{ formatMoney(w.revenue) }}</div>
                <div class="text-xs text-green-700">LN: {{ formatMoney(profit(w.revenue)) }}</div>
              </div>
              <span class="text-slate-400">{{ open[w.week_start] ? '▲' : '▼' }}</span>
            </div>
          </button>
          <div v-if="open[w.week_start]" class="overflow-x-auto border-t">
            <table class="w-full min-w-[720px] text-sm">
              <thead class="bg-slate-50 text-left text-xs uppercase text-slate-500">
                <tr><th class="p-2.5">Thời gian</th><th class="p-2.5">Xe</th><th class="p-2.5">Khách</th><th class="p-2.5">NV bán</th><th class="p-2.5 text-right">Giá gốc</th><th class="p-2.5 text-right">Giảm</th><th class="p-2.5 text-right">Thành tiền</th><th v-if="auth.isManager" class="p-2.5 text-right">Hoàn trả</th></tr>
              </thead>
              <tbody>
                <template v-for="it in w.items" :key="it.id">
                  <tr class="border-t" :class="it.refunded ? 'bg-red-50/60 text-slate-400' : ''">
                    <td class="whitespace-nowrap p-2.5" :class="it.refunded ? '' : 'text-slate-500'">{{ dt(it.created_at) }}</td>
                    <td class="p-2.5 font-medium" :class="it.refunded ? 'line-through' : ''">{{ it.vehicle_name }}</td>
                    <td class="p-2.5" :class="it.refunded ? 'line-through' : ''">{{ it.customer_name }}</td>
                    <td class="p-2.5 text-slate-500">{{ it.sold_by_name }}</td>
                    <td class="p-2.5 text-right text-slate-500">{{ formatMoney(it.original_price) }}</td>
                    <td class="p-2.5 text-right" :class="it.refunded ? 'text-slate-400' : 'text-red-600'">
                      <span v-if="it.discount_percent > 0">-{{ Math.round(it.discount_percent) }}%</span>
                      <span v-if="it.voucher_discount > 0"> · -{{ formatMoney(it.voucher_discount) }}</span>
                      <span v-if="!it.discount_percent && !it.voucher_discount" class="text-slate-300">—</span>
                    </td>
                    <td class="p-2.5 text-right font-semibold" :class="it.refunded ? 'text-slate-400 line-through' : 'text-brand-800'">
                      {{ formatMoney(it.final_price) }}
                    </td>
                    <td v-if="auth.isManager" class="p-2.5 text-right">
                      <span v-if="it.refunded" class="badge bg-red-100 text-red-600">Đã hoàn</span>
                      <button v-else class="rounded-md px-2 py-0.5 text-xs text-red-600 ring-1 ring-red-200 hover:bg-red-50" @click="openRefund(it)">Hoàn trả</button>
                    </td>
                  </tr>
                  <!-- dòng lý do hoàn / form hoàn -->
                  <tr v-if="it.refunded && it.refund_reason" class="bg-red-50/40">
                    <td :colspan="auth.isManager ? 8 : 7" class="px-2.5 pb-2 pt-0 text-xs text-red-500">↳ Lý do hoàn: {{ it.refund_reason }}</td>
                  </tr>
                  <tr v-if="auth.isManager && refundFor === it.id" class="bg-red-50">
                    <td :colspan="8" class="p-3">
                      <div class="flex flex-wrap items-center gap-2">
                        <span class="text-xs font-medium text-red-700">Lý do hoàn trả xe "{{ it.vehicle_name }}" cho {{ it.customer_name }}:</span>
                        <input v-model="refundReason" class="input flex-1 !py-1.5 text-sm" placeholder="VD: bán sai người, nhân viên xử lý sai…" @keyup.enter="doRefund(it.id)" />
                        <button class="rounded-md bg-red-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-red-700" @click="doRefund(it.id)">Xác nhận hoàn</button>
                        <button class="rounded-md px-2 py-1.5 text-xs text-slate-400 hover:text-slate-600" @click="refundFor = null">Huỷ</button>
                      </div>
                      <p class="mt-1 text-xs text-slate-500">Hoàn trả sẽ: cộng lại tồn kho, trừ chi tiêu &amp; tính lại hạng khách, khôi phục voucher đã dùng. Đơn này sẽ không tính vào doanh thu.</p>
                      <p v-if="refundMsg" class="mt-1 text-xs font-medium text-red-600">{{ refundMsg }}</p>
                    </td>
                  </tr>
                </template>
              </tbody>
            </table>
          </div>
        </div>
        <p v-if="!shownWeeks.length" class="card p-8 text-center text-slate-400">Không có giao dịch trong phạm vi đã chọn.</p>
      </div>

      <!-- top xe & khách -->
      <div class="mt-6 grid gap-6 lg:grid-cols-2">
        <div class="card overflow-x-auto">
          <h2 class="border-b p-4 font-semibold text-brand-900">🚗 Xe bán chạy</h2>
          <table class="w-full text-sm">
            <tbody>
              <tr v-for="(v, i) in topVehicles" :key="v.name" class="border-t">
                <td class="p-3 text-slate-400">{{ i + 1 }}</td><td class="p-3 font-medium">{{ v.name }}</td>
                <td class="p-3 text-slate-500">{{ v.sales }} đơn</td><td class="p-3 text-right font-medium text-brand-800">{{ formatMoney(v.revenue) }}</td>
              </tr>
              <tr v-if="!topVehicles.length"><td class="p-6 text-center text-slate-400" colspan="4">Chưa có dữ liệu.</td></tr>
            </tbody>
          </table>
        </div>
        <div class="card overflow-x-auto">
          <h2 class="border-b p-4 font-semibold text-brand-900">👥 Khách chi tiêu nhiều nhất</h2>
          <table class="w-full text-sm">
            <tbody>
              <tr v-for="(c, i) in topCustomers" :key="c.name" class="border-t">
                <td class="p-3 text-slate-400">{{ i + 1 }}</td><td class="p-3 font-medium">{{ c.name }}</td>
                <td class="p-3 text-slate-500">{{ c.sales }} đơn</td><td class="p-3 text-right font-medium text-brand-800">{{ formatMoney(c.revenue) }}</td>
              </tr>
              <tr v-if="!topCustomers.length"><td class="p-6 text-center text-slate-400" colspan="4">Chưa có dữ liệu.</td></tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>
  </div>
</template>
