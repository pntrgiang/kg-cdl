<script setup lang="ts">
definePageMeta({ layout: 'staff', middleware: 'manager' })
const api = useApi()

const { data: vouchers, refresh: refreshVouchers } = await useAsyncData('vouchers', () => api.get<any[]>('/api/vouchers'))
const msg = ref(''); const okMsg = ref('')

const vForm = reactive({
  name: '', discount_percent: 10,
  max_mode: 'amount' as 'amount' | 'full', max_amount: 5000,
  quantity: 1, expires_at: '', min_rank: 'regular',
  scope: 'all' as 'all' | 'specific',
})
const pickedVehicles = ref<any[]>([]) // [{id, name, brand}]

// tìm xe để chọn phạm vi áp dụng
const vehSearch = ref('')
const vehResults = ref<any[]>([])
let t: any
watch(vehSearch, (q) => {
  clearTimeout(t)
  t = setTimeout(async () => {
    if (q.trim().length < 2) { vehResults.value = []; return }
    vehResults.value = await api.get<any[]>(`/api/catalog?search=${encodeURIComponent(q)}&limit=15`)
  }, 250)
})
function addVehicle(c: any) {
  if (!pickedVehicles.value.some((x) => x.id === c.id)) pickedVehicles.value.push(c)
  vehSearch.value = ''; vehResults.value = []
}
function removeVehicle(id: number) { pickedVehicles.value = pickedVehicles.value.filter((x) => x.id !== id) }

async function createVoucher() {
  msg.value = ''; okMsg.value = ''
  if (!vForm.name.trim()) { msg.value = 'Cần nhập tên voucher.'; return }
  if (!(vForm.discount_percent > 0 && vForm.discount_percent <= 100)) { msg.value = '% giảm phải trong (0, 100].'; return }
  if (vForm.max_mode === 'amount' && !(vForm.max_amount > 0)) { msg.value = 'Nhập số tiền giảm tối đa > 0 (hoặc chọn "tối đa = giá trị xe").'; return }
  if (!(vForm.quantity >= 1)) { msg.value = 'Số lượng phải ≥ 1.'; return }
  if (!vForm.expires_at) { msg.value = 'Cần chọn hạn sử dụng.'; return }
  if (vForm.scope === 'specific' && !pickedVehicles.value.length) { msg.value = 'Cần chọn ít nhất 1 xe áp dụng.'; return }
  try {
    await api.post('/api/vouchers', {
      name: vForm.name,
      discount_percent: Number(vForm.discount_percent),
      max_amount: vForm.max_mode === 'full' ? 0 : Number(vForm.max_amount),
      quantity: Number(vForm.quantity),
      expires_at: `${vForm.expires_at}T23:59:59+07:00`,
      applies_to_all: vForm.scope === 'all',
      vehicle_ids: vForm.scope === 'all' ? [] : pickedVehicles.value.map((x) => x.id),
      min_rank: vForm.min_rank,
    })
    okMsg.value = 'Đã tạo voucher.'
    vForm.name = ''; vForm.quantity = 1; pickedVehicles.value = []
    await refreshVouchers()
  } catch (e: any) { msg.value = e?.data?.error || 'Tạo voucher thất bại.' }
}

const rankLabel: Record<string, string> = { regular: 'Mọi khách', vip: 'VIP trở lên', svip: 'Chỉ SVIP' }
const fmtDate = (s: string) => formatDate(s)

// ── huỷ voucher (bắt buộc lý do) ──
const cancelFor = ref<number | null>(null)
const cancelReason = ref('')
const cancelMsg = ref('')
function openCancel(v: any) { cancelFor.value = v.id; cancelReason.value = ''; cancelMsg.value = '' }
async function doCancel(id: number) {
  cancelMsg.value = ''
  if (!cancelReason.value.trim()) { cancelMsg.value = 'Bắt buộc nhập lý do huỷ.'; return }
  try {
    await api.post(`/api/vouchers/${id}/cancel`, { reason: cancelReason.value.trim() })
    cancelFor.value = null
    okMsg.value = 'Đã huỷ voucher và thu hồi các bản chưa dùng của khách.'
    await refreshVouchers()
  } catch (e: any) { cancelMsg.value = e?.data?.error || 'Huỷ voucher thất bại.' }
}
</script>

<template>
  <div>
    <h1 class="mb-4 font-serif text-2xl font-bold text-brand-900">🎟️ Voucher (Quản lý)</h1>
    <div v-if="okMsg" class="mb-4 rounded-lg bg-green-50 px-4 py-3 text-sm text-green-700">{{ okMsg }}</div>
    <div v-if="msg" class="mb-4 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700">{{ msg }}</div>

    <div class="grid gap-6 lg:grid-cols-2">
      <!-- TẠO VOUCHER -->
      <div class="card p-4">
        <h2 class="mb-3 font-semibold">Tạo voucher</h2>
        <div class="space-y-3">
          <div><label class="label">Tên voucher</label><input v-model="vForm.name" class="input" placeholder="VD: Voucher 10% tối đa 10.000$" /></div>

          <div class="grid grid-cols-2 gap-3">
            <div><label class="label">% giảm</label><input v-model.number="vForm.discount_percent" type="number" class="input" /></div>
            <div><label class="label">Số lượng (lượt dùng)</label><input v-model.number="vForm.quantity" type="number" class="input" /></div>
          </div>

          <div>
            <label class="label">Giảm tối đa</label>
            <div class="flex flex-wrap items-center gap-3 text-sm">
              <label class="flex items-center gap-1.5"><input v-model="vForm.max_mode" type="radio" value="amount" /> Số tiền cụ thể</label>
              <input v-if="vForm.max_mode === 'amount'" v-model.number="vForm.max_amount" type="number" class="input w-36" placeholder="VD: 5000" />
              <label class="flex items-center gap-1.5"><input v-model="vForm.max_mode" type="radio" value="full" /> Tối đa = giá trị xe</label>
            </div>
          </div>

          <div class="grid grid-cols-2 gap-3">
            <div><label class="label">Hạn sử dụng</label><input v-model="vForm.expires_at" type="date" class="input" /></div>
            <div>
              <label class="label">Hạng được dùng</label>
              <select v-model="vForm.min_rank" class="input">
                <option value="regular">Mọi khách</option>
                <option value="vip">VIP trở lên</option>
                <option value="svip">Chỉ SVIP</option>
              </select>
            </div>
          </div>

          <div>
            <label class="label">Loại xe áp dụng</label>
            <div class="mb-2 flex gap-4 text-sm">
              <label class="flex items-center gap-1.5"><input v-model="vForm.scope" type="radio" value="all" /> Tất cả xe</label>
              <label class="flex items-center gap-1.5"><input v-model="vForm.scope" type="radio" value="specific" /> Xe cụ thể</label>
            </div>
            <div v-if="vForm.scope === 'specific'">
              <input v-model="vehSearch" class="input" placeholder="Gõ ≥2 ký tự để tìm & thêm xe…" />
              <div v-if="vehResults.length" class="mt-1 max-h-36 space-y-1 overflow-auto rounded-lg border p-1">
                <button v-for="c in vehResults" :key="c.id" type="button" class="block w-full rounded px-2 py-1.5 text-left text-sm hover:bg-brand-50" @click="addVehicle(c)">
                  <strong>{{ c.name }}</strong> <span class="text-slate-400">· {{ c.brand }}</span>
                </button>
              </div>
              <div v-if="pickedVehicles.length" class="mt-2 flex flex-wrap gap-1.5">
                <span v-for="x in pickedVehicles" :key="x.id" class="badge bg-brand-100 text-brand-800">
                  {{ x.name }} <button class="ml-1 text-red-500" @click="removeVehicle(x.id)">✕</button>
                </span>
              </div>
            </div>
          </div>

          <button class="btn-primary w-full" @click="createVoucher">Tạo voucher</button>
        </div>
      </div>

      <!-- DANH SÁCH -->
      <div class="card p-4">
        <h2 class="mb-3 font-semibold">Voucher đã có ({{ vouchers?.length || 0 }})</h2>
        <div class="max-h-[34rem] space-y-2 overflow-auto">
          <div v-for="v in vouchers" :key="v.id" class="rounded-lg border px-3 py-2 text-sm" :class="v.cancelled_at ? 'border-red-200 bg-red-50/50 opacity-90' : ''">
            <div class="flex items-center justify-between gap-2">
              <strong :class="v.cancelled_at ? 'text-slate-500 line-through' : ''">{{ v.name }}</strong>
              <span v-if="v.cancelled_at" class="badge shrink-0 bg-red-100 text-red-600">Đã huỷ</span>
              <span v-else class="badge shrink-0" :class="v.remaining > 0 ? 'bg-green-100 text-green-700' : 'bg-slate-200 text-slate-500'">Còn {{ v.remaining }}/{{ v.quantity }}</span>
            </div>
            <div class="text-slate-500">
              Giảm {{ v.discount_percent }}% · {{ v.max_amount > 0 ? 'tối đa ' + formatMoney(v.max_amount) : 'tối đa = giá trị xe' }}
            </div>
            <div class="mt-1 flex flex-wrap gap-1.5 text-xs">
              <span class="badge bg-slate-100 text-slate-600">HSD: {{ v.expires_at ? fmtDate(v.expires_at) : 'không hạn' }}</span>
              <span class="badge bg-slate-100 text-slate-600">Hạng: {{ rankLabel[v.min_rank] || v.min_rank }}</span>
              <span class="badge bg-slate-100 text-slate-600">
                {{ v.applies_to_all ? 'Mọi xe' : 'Xe: ' + (v.vehicles || []).map((x:any)=>x.name).join(', ') }}
              </span>
            </div>

            <!-- lý do huỷ -->
            <p v-if="v.cancelled_at && v.cancel_reason" class="mt-1.5 text-xs text-red-600">↳ Lý do huỷ: {{ v.cancel_reason }}</p>

            <!-- trình huỷ -->
            <div v-else-if="cancelFor === v.id" class="mt-2 rounded-lg border border-red-200 bg-red-50 p-2">
              <input v-model="cancelReason" class="input !py-1.5 text-sm" placeholder="Lý do huỷ voucher…" @keyup.enter="doCancel(v.id)" />
              <p class="mt-1 text-xs text-slate-500">Huỷ sẽ thu hồi mọi bản voucher khách đang giữ nhưng <strong>chưa dùng</strong>; khách sẽ thấy thông báo liên hệ nhân viên.</p>
              <p v-if="cancelMsg" class="mt-1 text-xs font-medium text-red-600">{{ cancelMsg }}</p>
              <div class="mt-1.5 flex gap-2">
                <button class="rounded-md bg-red-600 px-3 py-1 text-xs font-medium text-white hover:bg-red-700" @click="doCancel(v.id)">Xác nhận huỷ</button>
                <button class="rounded-md px-2 py-1 text-xs text-slate-400 hover:text-slate-600" @click="cancelFor = null">Đóng</button>
              </div>
            </div>

            <!-- nút mở huỷ -->
            <div v-else class="mt-1.5 text-right">
              <button class="rounded-md px-2 py-0.5 text-xs text-red-600 ring-1 ring-red-200 hover:bg-red-50" @click="openCancel(v)">Huỷ voucher</button>
            </div>
          </div>
          <p v-if="!vouchers?.length" class="py-6 text-center text-sm text-slate-400">Chưa có voucher.</p>
        </div>
      </div>
    </div>
  </div>
</template>
