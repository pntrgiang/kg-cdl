<script setup lang="ts">
definePageMeta({ layout: 'staff', middleware: 'staff' })
const api = useApi()

const { data: inventory, refresh: refreshInv } = await useAsyncData('sell-inv', () =>
  api.get<any[]>('/api/inventory?status=on_sale'),
)
const { data: customers, refresh: refreshCust } = await useAsyncData('sell-cust', () =>
  api.get<any[]>('/api/customers'),
)

const vehSearch = ref('')
const custSearch = ref('')
const selectedVeh = ref<any>(null)
const selectedCust = ref<any>(null)
const msg = ref('')
const ok = ref('')

// ── toast tồn kho (chỉ tắt khi nhân viên bấm ✕) ──
// Ngưỡng "sắp hết": còn ≤ 3 chiếc (vẫn > 0). "Đã hết": 0 chiếc.
const LOW_STOCK = 3
interface StockToast { id: number; key: string; type: 'warn' | 'danger'; title: string; msg: string }
const toasts = ref<StockToast[]>([])
let toastSeq = 0
function closeToast(id: number) { toasts.value = toasts.value.filter((t) => t.id !== id) }
// lưu số lượng lần đánh giá gần nhất theo từng xe -> chỉ phát toast khi số lượng THAY ĐỔI
// (tránh tự bật lại toast mà nhân viên vừa đóng, nhưng vẫn bật lại nếu kho tiếp tục giảm).
const lastQty = new Map<number, number>()
function renderStockToast(id: number, name: string, qty: number) {
  const key = `stock-${id}`
  toasts.value = toasts.value.filter((t) => t.key !== key) // bỏ toast cũ của xe này
  if (qty <= 0) {
    toasts.value.push({ id: ++toastSeq, key, type: 'danger', title: 'Xe đã hết hàng', msg: `${name} hiện không còn chiếc nào trong kho.` })
  } else if (qty <= LOW_STOCK) {
    toasts.value.push({ id: ++toastSeq, key, type: 'warn', title: 'Xe sắp hết hàng', msg: `${name} chỉ còn ${qty} chiếc trong kho.` })
  }
}
// đánh giá tồn kho của xe đang chọn dựa trên dữ liệu kho mới nhất.
// force=true: luôn đánh giá lại (khi mới chọn xe / sau khi tự bán).
// force=false: chỉ đánh giá khi số lượng đổi so với lần trước (dùng cho auto-refresh).
function evaluateSelected(force: boolean) {
  const veh = selectedVeh.value
  if (!veh) return
  const fresh = (inventory.value || []).find((i) => i.id === veh.id)
  const qty = fresh ? fresh.quantity : 0 // không còn trong danh sách đang bán = đã hết
  if (force || lastQty.get(veh.id) !== qty) {
    renderStockToast(veh.id, veh.name, qty)
    lastQty.set(veh.id, qty)
  }
}
// khi chọn xe để bán -> kiểm tra tồn kho ngay
watch(selectedVeh, (v) => { if (v) evaluateSelected(true) })

// tự làm mới kho định kỳ để bắt thay đổi do nhân viên KHÁC bán cùng lúc.
let pollTimer: any = null
onMounted(() => {
  pollTimer = setInterval(async () => {
    if (!selectedVeh.value) return
    await refreshInv()
    evaluateSelected(false)
  }, 8000)
})
onBeforeUnmount(() => { if (pollTimer) clearInterval(pollTimer) })

// voucher khả dụng của khách CHO XE đang chọn (còn hạn, đúng hạng, áp dụng được)
const vouchers = ref<any[]>([])
const useVoucherId = ref<number | null>(null)
async function loadVouchers() {
  useVoucherId.value = null
  vouchers.value = []
  const c = selectedCust.value
  const veh = selectedVeh.value
  if (!c || !veh) return
  try {
    const res = await api.get<{ vouchers: any[] }>(`/api/customers/${c.id}/prizes?catalog_id=${veh.catalog_id}`)
    vouchers.value = res.vouchers || []
  } catch {}
}
watch([selectedCust, selectedVeh], loadVouchers)
const hasVouchers = computed(() => vouchers.value.length > 0)

// sửa giá bán cho riêng phiên này (không đổi giá gốc của xe). Đây là giá cuối, bỏ qua KM %; voucher vẫn áp dụng.
// Ô giá hiện sẵn = giá bán hiện tại của xe, nhân viên chỉnh trực tiếp.
const overridePrice = ref<number | null>(null)
watch(selectedVeh, (v) => { overridePrice.value = v ? Math.round(v.final_price) : null })
// có sửa giá không = giá nhập khác giá bán niêm yết hiện tại.
const priceChanged = computed(() => {
  const v = selectedVeh.value
  if (!v || overridePrice.value == null) return false
  return Math.round(overridePrice.value) !== Math.round(v.final_price)
})
const effectiveBase = computed(() => {
  if (overridePrice.value && overridePrice.value > 0) return overridePrice.value
  return selectedVeh.value?.final_price ?? 0
})

const availVeh = computed(() =>
  (inventory.value || []).filter(
    (v) => v.quantity > 0 && `${v.name} ${v.brand}`.toLowerCase().includes(vehSearch.value.toLowerCase()),
  ),
)
const availCust = computed(() =>
  (customers.value || []).filter((c) =>
    `${c.full_name} ${c.phone} ${c.national_id}`.toLowerCase().includes(custSearch.value.toLowerCase()),
  ),
)

// tạo khách mới
const showNew = ref(false)
const newCust = reactive({ full_name: '', phone: '', national_id: '' })
async function createCustomer() {
  msg.value = ''
  if (!newCust.full_name.trim()) { msg.value = 'Cần nhập họ tên.'; return }
  if (!isValidNationalID(newCust.national_id)) { msg.value = 'Số căn cước không hợp lệ. ' + NATIONAL_ID_HINT; return }
  try {
    const c = await api.post<any>('/api/customers', { ...newCust })
    await refreshCust()
    selectedCust.value = c
    showNew.value = false
    newCust.full_name = newCust.phone = newCust.national_id = ''
  } catch (e: any) {
    msg.value = e?.data?.error || 'Không tạo được khách.'
  }
}

async function confirmSale() {
  msg.value = ''
  ok.value = ''
  if (!selectedVeh.value || !selectedCust.value) {
    msg.value = 'Hãy chọn xe và khách hàng.'
    return
  }
  try {
    const body: any = { inventory_id: selectedVeh.value.id, customer_id: selectedCust.value.id }
    if (useVoucherId.value) body.customer_voucher_id = useVoucherId.value
    // chỉ áp dụng giá sửa khi nhân viên đổi khác giá niêm yết
    if (priceChanged.value) {
      if (!overridePrice.value || overridePrice.value <= 0) { msg.value = 'Giá bán phải lớn hơn 0.'; return }
      body.override_price = overridePrice.value
    }
    const sold = selectedVeh.value
    const res = await api.post<any>('/api/sales', body)
    ok.value = `Bán thành công ${res.sale.vehicle_name} cho ${res.sale.customer_name} — ${formatMoney(res.sale.final_price)}.` +
      (res.sale.voucher_discount > 0 ? ` (đã giảm ${formatMoney(res.sale.voucher_discount)} từ voucher)` : '') +
      (res.rank_changed_to ? ` Khách lên hạng ${res.rank_changed_to.toUpperCase()}!` : '')
    selectedVeh.value = null
    useVoucherId.value = null
    overridePrice.value = null
    await Promise.all([refreshInv(), refreshCust()])
    // kiểm tra tồn kho sau khi bán: nếu xe vừa bán đã hết / sắp hết -> toast
    const updated = (inventory.value || []).find((i) => i.id === sold.id)
    const newQty = updated ? updated.quantity : 0 // không còn trong danh sách đang bán = đã hết
    renderStockToast(sold.id, sold.name, newQty)
    lastQty.set(sold.id, newQty)
    await loadVouchers()
  } catch (e: any) {
    msg.value = e?.data?.error || 'Bán xe thất bại.'
  }
}
</script>

<template>
  <div>
    <!-- toast tồn kho: cố định góc phải, chỉ tắt khi bấm ✕ -->
    <div class="pointer-events-none fixed right-4 top-4 z-50 flex w-80 max-w-[calc(100vw-2rem)] flex-col gap-2">
      <div
        v-for="t in toasts" :key="t.id"
        class="pointer-events-auto flex items-start gap-3 rounded-xl border p-3 shadow-lg"
        :class="t.type === 'danger' ? 'border-red-200 bg-red-50' : 'border-amber-200 bg-amber-50'"
      >
        <span class="text-lg">{{ t.type === 'danger' ? '🚫' : '⚠️' }}</span>
        <div class="min-w-0 flex-1">
          <div class="text-sm font-semibold" :class="t.type === 'danger' ? 'text-red-700' : 'text-amber-700'">{{ t.title }}</div>
          <div class="text-xs" :class="t.type === 'danger' ? 'text-red-600' : 'text-amber-600'">{{ t.msg }}</div>
        </div>
        <button class="shrink-0 rounded-md px-1.5 text-slate-400 hover:text-slate-700" title="Đóng" @click="closeToast(t.id)">✕</button>
      </div>
    </div>

    <h1 class="mb-4 font-serif text-2xl font-bold text-brand-900">💰 Bán xe</h1>

    <div v-if="ok" class="mb-4 rounded-lg bg-green-50 px-4 py-3 text-sm text-green-700">{{ ok }}</div>
    <div v-if="msg" class="mb-4 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700">{{ msg }}</div>

    <div class="grid gap-6 lg:grid-cols-2">
      <!-- chọn xe -->
      <div class="card p-4">
        <h2 class="mb-2 font-semibold">1. Chọn xe (đang bán, còn hàng)</h2>
        <input v-model="vehSearch" class="input mb-3" placeholder="Tìm xe…" />
        <div class="max-h-72 space-y-2 overflow-auto">
          <button
            v-for="v in availVeh" :key="v.id"
            class="flex w-full items-center justify-between rounded-lg border px-3 py-2 text-left text-sm hover:bg-brand-50"
            :class="selectedVeh?.id === v.id ? 'border-brand-500 bg-brand-50' : 'border-slate-200'"
            @click="selectedVeh = v"
          >
            <span><strong>{{ v.name }}</strong> · còn {{ v.quantity }}</span>
            <span class="text-brand-700">{{ formatMoney(v.final_price) }}</span>
          </button>
          <p v-if="!availVeh.length" class="py-6 text-center text-sm text-slate-400">Không có xe phù hợp.</p>
        </div>
      </div>

      <!-- chọn khách -->
      <div class="card p-4">
        <div class="mb-2 flex items-center justify-between">
          <h2 class="font-semibold">2. Chọn khách hàng</h2>
          <button class="text-sm text-brand-600 hover:underline" @click="showNew = !showNew">
            {{ showNew ? 'Đóng' : '+ Tạo mới' }}
          </button>
        </div>

        <div v-if="showNew" class="mb-3 space-y-2 rounded-lg bg-slate-50 p-3">
          <input v-model="newCust.full_name" class="input" placeholder="Họ tên *" />
          <input v-model="newCust.phone" class="input" placeholder="Số điện thoại" />
          <input v-model="newCust.national_id" class="input uppercase placeholder:normal-case" placeholder="Số căn cước * (LUX12345)" @input="newCust.national_id = newCust.national_id.toUpperCase()" />
          <button class="btn-primary w-full !py-1.5 text-sm" @click="createCustomer">Lưu khách</button>
        </div>

        <input v-model="custSearch" class="input mb-3" placeholder="Tìm khách…" />
        <div class="max-h-56 space-y-2 overflow-auto">
          <button
            v-for="c in availCust" :key="c.id"
            class="flex w-full items-center justify-between rounded-lg border px-3 py-2 text-left text-sm hover:bg-brand-50"
            :class="selectedCust?.id === c.id ? 'border-brand-500 bg-brand-50' : 'border-slate-200'"
            @click="selectedCust = c"
          >
            <span><strong>{{ c.full_name }}</strong> · {{ c.national_id }}</span>
            <RankBadge :rank="c.rank" />
          </button>
          <p v-if="!availCust.length" class="py-6 text-center text-sm text-slate-400">Không có khách phù hợp.</p>
        </div>
      </div>
    </div>

    <!-- voucher áp dụng được cho xe đang chọn (chỉ hiện khi có) -->
    <div v-if="selectedCust && selectedVeh && hasVouchers" class="card mt-6 border-gold-400 p-4">
      <h2 class="mb-2 font-semibold text-brand-900">🎟️ Voucher áp dụng được cho xe này</h2>
      <label v-for="v in vouchers" :key="v.id" class="mb-1 flex items-center gap-2 rounded-lg border p-2 text-sm"
        :class="useVoucherId === v.id ? 'border-gold-500 bg-gold-500/10' : ''">
        <input type="radio" name="voucher" :value="v.id" :checked="useVoucherId === v.id" @change="useVoucherId = v.id" />
        <span>{{ v.name }} — giảm {{ v.discount_percent }}%<span v-if="v.max_amount > 0"> (tối đa {{ formatMoney(v.max_amount) }})</span><span v-else> (tối đa = giá trị xe)</span></span>
      </label>
      <button class="mt-2 text-xs text-slate-400 hover:underline" @click="useVoucherId = null">Không dùng voucher</button>
    </div>

    <!-- sửa giá bán cho riêng phiên này (hiện khi đã chọn xe + khách) -->
    <div v-if="selectedVeh && selectedCust" class="card mt-6 p-4">
      <div class="text-sm font-semibold text-brand-900">✏️ Giá bán cho phiên này</div>
      <p class="mt-1 text-xs text-slate-500">Đã điền sẵn giá bán hiện tại — chỉnh trực tiếp nếu muốn bán giá khác. Chỉ áp dụng cho lần bán này, không đổi giá gốc; voucher của khách vẫn được trừ thêm nếu có.</p>
      <div class="mt-3 flex flex-wrap items-center gap-3">
        <div class="relative">
          <span class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-sm text-slate-400">$</span>
          <input v-model.number="overridePrice" type="number" min="0" class="input w-52 pl-7 font-semibold" />
        </div>
        <span class="text-xs text-slate-400">Giá niêm yết: {{ formatMoney(selectedVeh.final_price) }}</span>
        <span v-if="priceChanged" class="badge bg-gold-500/20 text-gold-600">Đã chỉnh giá</span>
      </div>
    </div>

    <!-- xác nhận -->
    <div class="card mt-6 flex flex-col items-center justify-between gap-3 p-4 sm:flex-row">
      <div class="text-sm">
        <span v-if="selectedVeh" class="font-medium text-brand-800">
          {{ selectedVeh.name }}
          <span :class="priceChanged ? 'text-gold-600' : ''">({{ formatMoney(effectiveBase) }}<span v-if="priceChanged"> · giá sửa</span>)</span>
        </span>
        <span v-else class="text-slate-400">Chưa chọn xe</span>
        <span class="mx-2 text-slate-300">→</span>
        <span v-if="selectedCust" class="font-medium text-brand-800">{{ selectedCust.full_name }}</span>
        <span v-else class="text-slate-400">Chưa chọn khách</span>
      </div>
      <button class="btn-gold" :disabled="!selectedVeh || !selectedCust" @click="confirmSale">Xác nhận bán</button>
    </div>
  </div>
</template>
