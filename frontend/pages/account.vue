<script setup lang="ts">
const auth = useAuthStore()
const api = useApi()

const vouchers = ref<any[]>([])
const bookings = ref<any[]>([])
const loaded = ref(false)

onMounted(async () => {
  auth.hydrate()
  if (!auth.isAuthed) { navigateTo('/customer/login'); return }
  if (auth.isCustomer) {
    try { const r = await api.get<{ vouchers: any[] }>('/api/me/prizes'); vouchers.value = r.vouchers || [] } catch {}
    try { const r = await api.get<{ bookings: any[] }>('/api/me/bookings'); bookings.value = r.bookings || [] } catch {}
  }
  loaded.value = true
})

const rankLabel: Record<string, string> = { regular: 'Mọi khách', vip: 'VIP trở lên', svip: 'Chỉ SVIP' }
function isExpired(v: any) { return v.expires_at && new Date(v.expires_at).getTime() < Date.now() }

const bookingStatus: Record<string, { label: string; cls: string }> = {
  pending: { label: 'Chờ tiếp nhận', cls: 'bg-amber-100 text-amber-700' },
  accepted: { label: 'Đã được tiếp nhận', cls: 'bg-green-100 text-green-700' },
  rejected: { label: 'Đã bị từ chối', cls: 'bg-red-100 text-red-600' },
}
</script>

<template>
  <section class="mx-auto max-w-md">
    <h1 class="mb-1 font-serif text-2xl font-bold text-brand-900">🔐 Tài khoản của tôi</h1>
    <p class="mb-6 text-sm text-slate-500">{{ auth.displayName }}</p>

    <!-- lịch đặt xem xe của tôi -->
    <div v-if="auth.isCustomer && loaded && bookings.length" class="mb-6">
      <h2 class="mb-2 font-semibold text-brand-900">📅 Lịch đặt xem xe</h2>
      <div class="space-y-3">
        <div v-for="b in bookings" :key="'bk' + b.id" class="card p-4">
          <div class="flex items-center justify-between gap-2">
            <div class="font-semibold text-brand-900">{{ b.vehicle_name }}</div>
            <span class="badge shrink-0" :class="bookingStatus[b.status]?.cls">{{ bookingStatus[b.status]?.label || b.status }}</span>
          </div>
          <div class="mt-1 text-sm text-slate-600">Ngày hẹn đến xem: <strong>{{ formatDate(b.visit_date) }}</strong></div>
          <div v-if="b.note" class="mt-0.5 text-xs italic text-slate-500">Ghi chú: “{{ b.note }}”</div>
          <div class="mt-1 text-xs text-slate-400">Đặt lúc {{ formatDateTime(b.created_at) }}</div>
          <p v-if="b.status === 'pending'" class="mt-1 text-xs text-amber-600">Đang chờ nhân viên tiếp nhận.</p>
          <p v-else-if="b.status === 'rejected'" class="mt-1 text-xs text-red-500">Lịch đã bị từ chối. Vui lòng liên hệ nhân viên hoặc đặt lịch khác.</p>
        </div>
      </div>
    </div>

    <!-- voucher của tôi -->
    <div v-if="auth.isCustomer && loaded" class="mb-6">
      <h2 class="mb-2 font-semibold text-brand-900">🎟️ Voucher của tôi</h2>

      <div v-if="!vouchers.length" class="card p-5 text-center text-sm text-slate-400">
        Bạn chưa có voucher nào. Tham gia các
        <NuxtLink to="/events" class="text-brand-600 hover:underline">sự kiện</NuxtLink> để nhận thưởng nhé!
      </div>

      <div v-else class="space-y-3">
        <div v-for="v in vouchers" :key="'vou' + v.id" class="card p-4" :class="(v.status === 'used' || v.status === 'cancelled' || isExpired(v)) ? 'opacity-80' : ''" :style="v.status === 'cancelled' ? 'border-color:#fecaca' : ''">
          <div class="flex items-start gap-3">
            <span class="text-2xl">{{ v.status === 'cancelled' ? '🚫' : '🎟️' }}</span>
            <div class="min-w-0 flex-1">
              <div class="flex items-center justify-between gap-2">
                <div class="font-semibold text-brand-900" :class="v.status === 'cancelled' ? 'line-through text-slate-500' : ''">{{ v.name }}</div>
                <span v-if="v.status === 'cancelled'" class="badge shrink-0 bg-red-100 text-red-600">Đã bị huỷ</span>
                <span v-else-if="v.status === 'used'" class="badge shrink-0 bg-slate-200 text-slate-600">Đã sử dụng</span>
                <span v-else-if="isExpired(v)" class="badge shrink-0 bg-red-100 text-red-700">Hết hạn</span>
                <span v-else class="badge shrink-0 bg-green-100 text-green-700">Chưa sử dụng</span>
              </div>
              <div class="text-xs text-slate-500">
                Giảm {{ v.discount_percent }}% · {{ v.max_amount > 0 ? 'tối đa ' + formatMoney(v.max_amount) : 'tối đa = giá trị xe' }}
              </div>
              <div class="mt-1 flex flex-wrap gap-1.5 text-xs">
                <span class="badge bg-slate-100 text-slate-600">HSD: {{ v.expires_at ? formatDate(v.expires_at) : 'không hạn' }}</span>
                <span class="badge bg-slate-100 text-slate-600">Hạng: {{ rankLabel[v.min_rank] || v.min_rank }}</span>
                <span class="badge bg-slate-100 text-slate-600">
                  {{ v.applies_to_all ? 'Mọi xe' : 'Xe: ' + (v.vehicles || []).map((x) => x.name).join(', ') }}
                </span>
              </div>
              <div v-if="v.status === 'cancelled'" class="mt-1.5 rounded-md bg-red-50 px-2 py-1.5 text-xs text-red-600">
                ⚠️ Voucher này đã bị huỷ<span v-if="v.cancel_reason"> (lý do: {{ v.cancel_reason }})</span>. Vui lòng <strong>liên hệ nhân viên</strong> để được giải quyết.
              </div>
              <div v-else-if="v.status === 'used'" class="mt-1 text-xs text-slate-600">
                Đã dùng lúc {{ formatDateTime(v.used_at) }}<span v-if="v.seller_name"> · Nhân viên áp dụng: <strong>{{ v.seller_name }}</strong></span>
              </div>
              <div v-else-if="!isExpired(v)" class="mt-1 text-xs text-slate-500">Yêu cầu nhân viên khi mua xe để được áp dụng.</div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <ChangePasswordForm />
  </section>
</template>
