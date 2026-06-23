<script setup lang="ts">
definePageMeta({ layout: 'staff', middleware: 'staff' })
const api = useApi()
const auth = useAuthStore()

const msg = ref(''); const okMsg = ref('')

// ── danh sách yêu cầu đặt lịch (nhân viên + quản lý) ──
const statusFilter = ref('') // '' = tất cả
const { data: bookings, refresh: refreshBookings } = await useAsyncData(
  'bookings',
  () => api.get<any[]>(`/api/bookings${statusFilter.value ? '?status=' + statusFilter.value : ''}`),
  { watch: [statusFilter] },
)

async function handle(b: any, status: 'accepted' | 'rejected') {
  msg.value = ''; okMsg.value = ''
  try {
    await api.patch(`/api/bookings/${b.id}`, { status })
    okMsg.value = status === 'accepted' ? `Đã nhận lịch của ${b.customer_name}.` : `Đã từ chối lịch của ${b.customer_name}.`
    await refreshBookings()
  } catch (e: any) { msg.value = e?.data?.error || 'Xử lý lịch thất bại.' }
}

// ── mở/đóng nhận đặt lịch theo xe (chỉ quản lý) ──
const { data: inventory, refresh: refreshInv } = await useAsyncData('booking-inv', () =>
  auth.isManager ? api.get<any[]>('/api/inventory') : Promise.resolve([] as any[]),
)

const invSearch = ref('')
const shownInv = computed(() =>
  (inventory.value || []).filter((i: any) => `${i.name} ${i.brand}`.toLowerCase().includes(invSearch.value.toLowerCase())),
)
async function toggleBooking(i: any) {
  msg.value = ''; okMsg.value = ''
  try {
    await api.patch(`/api/inventory/${i.id}/booking`, { open: !i.booking_open })
    await refreshInv()
  } catch (e: any) { msg.value = e?.data?.error || 'Đổi trạng thái nhận đặt lịch thất bại.' }
}

const statusMeta: Record<string, { label: string; cls: string }> = {
  pending: { label: 'Chờ duyệt', cls: 'bg-amber-100 text-amber-700' },
  accepted: { label: 'Đã nhận', cls: 'bg-green-100 text-green-700' },
  rejected: { label: 'Đã từ chối', cls: 'bg-red-100 text-red-600' },
}
const pendingCount = computed(() => (bookings.value || []).filter((b: any) => b.status === 'pending').length)
const dt = (s: string) => formatDateTime(s)
const fd = (s: string) => formatDate(s)
</script>

<template>
  <div>
    <h1 class="mb-1 font-serif text-2xl font-bold text-brand-900">📅 Đặt lịch xem/mua xe</h1>
    <p class="mb-4 text-sm text-slate-500">Tiếp nhận lịch hẹn của khách đến xem/mua xe.</p>

    <div v-if="okMsg" class="mb-4 rounded-lg bg-green-50 px-4 py-3 text-sm text-green-700">{{ okMsg }}</div>
    <div v-if="msg" class="mb-4 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700">{{ msg }}</div>

    <!-- QUẢN LÝ: mở/đóng nhận đặt lịch theo xe -->
    <div v-if="auth.isManager" class="card mb-6 p-4">
      <h2 class="mb-1 font-semibold">Mở nhận đặt lịch theo xe</h2>
      <p class="mb-3 text-xs text-slate-500">Bật cho xe nào thì xe đó sẽ có nút “Đặt lịch” ở trang chi tiết để khách đặt hẹn.</p>
      <input v-model="invSearch" class="input mb-3 max-w-xs !py-1.5 text-sm" placeholder="Tìm xe trong kho…" />
      <div class="max-h-64 space-y-1.5 overflow-auto">
        <div v-for="i in shownInv" :key="i.id" class="flex items-center justify-between rounded-lg border px-3 py-2 text-sm"
          :class="i.booking_open ? 'border-green-300 bg-green-50/50' : 'border-slate-200'">
          <div>
            <strong>{{ i.name }}</strong>
            <span class="text-xs text-slate-400"> · {{ i.brand }} · tồn {{ i.quantity }}</span>
          </div>
          <button
            class="rounded-md px-3 py-1 text-xs font-medium ring-1 transition"
            :class="i.booking_open ? 'bg-green-600 text-white ring-green-600 hover:bg-green-700' : 'text-slate-600 ring-slate-300 hover:bg-slate-50'"
            @click="toggleBooking(i)"
          >{{ i.booking_open ? '✓ Đang nhận đặt lịch' : 'Bật nhận đặt lịch' }}</button>
        </div>
        <p v-if="!shownInv.length" class="py-6 text-center text-sm text-slate-400">Không có xe phù hợp.</p>
      </div>
    </div>

    <!-- DANH SÁCH YÊU CẦU ĐẶT LỊCH -->
    <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
      <h2 class="font-semibold text-brand-900">
        Yêu cầu đặt lịch
        <span v-if="pendingCount > 0" class="badge ml-1 bg-amber-100 text-amber-700">{{ pendingCount }} chờ duyệt</span>
      </h2>
      <div class="flex items-center gap-2">
        <label class="text-sm text-slate-500">Trạng thái:</label>
        <select v-model="statusFilter" class="input max-w-[180px] !py-1.5 text-sm">
          <option value="">Tất cả</option>
          <option value="pending">Chờ duyệt</option>
          <option value="accepted">Đã nhận</option>
          <option value="rejected">Đã từ chối</option>
        </select>
      </div>
    </div>

    <div class="card overflow-x-auto">
      <table class="w-full min-w-[720px] text-sm">
        <thead class="bg-slate-50 text-left text-xs uppercase text-slate-500">
          <tr>
            <th class="p-3">Xe</th>
            <th class="p-3">Khách hàng</th>
            <th class="p-3">Ngày hẹn xem</th>
            <th class="p-3">Đặt lúc</th>
            <th class="p-3">Trạng thái</th>
            <th class="p-3"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="b in bookings" :key="b.id" class="border-t align-top"
            :class="b.status === 'pending' ? 'bg-amber-50/40' : ''">
            <td class="p-3 font-medium">{{ b.vehicle_name }}</td>
            <td class="p-3">
              <div class="font-medium">{{ b.customer_name }}</div>
              <div class="text-xs text-slate-400">{{ b.customer_national_id }}<span v-if="b.customer_phone"> · {{ b.customer_phone }}</span></div>
              <div v-if="b.note" class="mt-0.5 text-xs italic text-slate-500">“{{ b.note }}”</div>
            </td>
            <td class="p-3 font-medium text-brand-800">{{ fd(b.visit_date) }}</td>
            <td class="whitespace-nowrap p-3 text-xs text-slate-500">{{ dt(b.created_at) }}</td>
            <td class="p-3">
              <span class="badge" :class="statusMeta[b.status]?.cls">{{ statusMeta[b.status]?.label || b.status }}</span>
              <div v-if="b.status !== 'pending' && b.handled_by_name" class="mt-0.5 text-xs text-slate-400">bởi {{ b.handled_by_name }}</div>
            </td>
            <td class="p-3 text-right">
              <div v-if="b.status === 'pending'" class="flex items-center justify-end gap-1.5">
                <button class="rounded-md bg-green-600 px-2.5 py-1 text-xs font-medium text-white hover:bg-green-700" @click="handle(b, 'accepted')">Nhận</button>
                <button class="rounded-md px-2.5 py-1 text-xs font-medium text-red-600 ring-1 ring-red-200 hover:bg-red-50" @click="handle(b, 'rejected')">Từ chối</button>
              </div>
            </td>
          </tr>
          <tr v-if="!bookings?.length"><td colspan="6" class="p-8 text-center text-slate-400">Chưa có yêu cầu đặt lịch nào.</td></tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
