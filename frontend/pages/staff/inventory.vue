<script setup lang="ts">
definePageMeta({ layout: 'staff', middleware: 'staff' })
const api = useApi()
const auth = useAuthStore()
const canEdit = computed(() => auth.isManager) // chỉ quản lý/dev mới chỉnh kho; nhân viên chỉ xem

const { data: inventory, refresh } = await useAsyncData('inv-list', () => api.get<any[]>('/api/inventory'))

// ── tuần mở bán ──
const { data: weeks, refresh: refreshWeeks } = await useAsyncData('sales-weeks', () => api.get<any[]>('/api/sales-weeks'))
const selectedWeekId = ref<number | null>(null)
const selectedWeek = computed(() => (weeks.value || []).find((w: any) => w.id === selectedWeekId.value) || null)
watchEffect(() => { if (!selectedWeekId.value && weeks.value?.length) selectedWeekId.value = weeks.value[0].id })

const showNewWeek = ref(false)
const newWeekDate = ref('')
async function registerWeek() {
  msg.value = ''
  if (!newWeekDate.value) { msg.value = 'Chọn một ngày trong tuần muốn mở bán.'; return }
  try {
    const w = await api.post<any>('/api/sales-weeks', { date: newWeekDate.value })
    await refreshWeeks()
    selectedWeekId.value = w.id
    showNewWeek.value = false; newWeekDate.value = ''
    okMsg.value = `Đã đăng ký ${w.label}.`
  } catch (e: any) { msg.value = e?.data?.error || 'Đăng ký tuần thất bại.' }
}

// ── nhập kho từ catalog ──
const catSearch = ref('')
const catResults = ref<any[]>([])
const pickedCatalog = ref<any>(null)
const stockForm = reactive({ base_price: 0, quantity: 1, trunk_kg: 10 })
const msg = ref(''); const okMsg = ref('')

// khi chọn mẫu xe có sẵn -> lấy cốp xe hiện tại để quản lý điều chỉnh lại
watch(pickedCatalog, (c) => { if (c) stockForm.trunk_kg = c.trunk_kg ?? 10 })

let t: any
watch(catSearch, (q) => {
  clearTimeout(t)
  t = setTimeout(async () => {
    if (q.trim().length < 2) { catResults.value = []; return }
    catResults.value = await api.get<any[]>(`/api/catalog?search=${encodeURIComponent(q)}&limit=20`)
  }, 250)
})

async function addStock() {
  msg.value = ''; okMsg.value = ''
  if (!pickedCatalog.value) { msg.value = 'Chọn một mẫu xe.'; return }
  if (!(stockForm.trunk_kg > 0)) { msg.value = 'Cần nhập cốp xe (kg) lớn hơn 0.'; return }
  if (!selectedWeekId.value) { msg.value = 'Chọn tuần mở bán (hoặc đăng ký tuần mới).'; return }
  try {
    // nếu quản lý điều chỉnh cốp xe -> cập nhật lại vào mẫu xe
    if (Number(stockForm.trunk_kg) !== (pickedCatalog.value.trunk_kg ?? 10)) {
      await api.patch(`/api/catalog/${pickedCatalog.value.id}`, {
        description: pickedCatalog.value.description || '',
        seats: pickedCatalog.value.seats ?? null,
        trunk_kg: Number(stockForm.trunk_kg),
        rate_speed: pickedCatalog.value.rate_speed,
        rate_accel: pickedCatalog.value.rate_accel,
        rate_braking: pickedCatalog.value.rate_braking,
        rate_traction: pickedCatalog.value.rate_traction,
      })
      pickedCatalog.value.trunk_kg = Number(stockForm.trunk_kg)
    }
    await api.post('/api/inventory', {
      catalog_id: pickedCatalog.value.id,
      base_price: Number(stockForm.base_price),
      quantity: Number(stockForm.quantity),
      sales_week_id: selectedWeekId.value,
    })
    const st = selectedWeek.value?.is_current ? 'đang mở bán' : 'sắp mở bán'
    okMsg.value = `Đã nhập kho ${pickedCatalog.value.name} cho ${selectedWeek.value?.label} (${st}).`
    pickedCatalog.value = null; catSearch.value = ''; catResults.value = []
    await refresh()
  } catch (e: any) { msg.value = e?.data?.error || 'Nhập kho thất bại.' }
}

// ── tạo xe mod mới ──
const showMod = ref(false)
const modForm = reactive({
  name: '', brand: '', class: '', model_code: '', image_url: '', description: '',
  seats: 2, trunk_kg: 10, rate_speed: 50, rate_accel: 50, rate_braking: 50, rate_traction: 50,
})
async function createMod() {
  msg.value = ''
  if (!modForm.name.trim()) { msg.value = 'Cần nhập tên xe.'; return }
  if (!(modForm.trunk_kg > 0)) { msg.value = 'Cần nhập cốp xe (kg) lớn hơn 0.'; return }
  try {
    const v = await api.post<any>('/api/catalog', { ...modForm })
    pickedCatalog.value = v
    catSearch.value = v.name
    showMod.value = false
    okMsg.value = `Đã tạo mẫu xe mod "${v.name}". Giờ nhập số lượng & giá để vào kho.`
    modForm.name = modForm.brand = modForm.class = modForm.model_code = modForm.image_url = modForm.description = ''
    modForm.seats = 2; modForm.trunk_kg = 10
    modForm.rate_speed = modForm.rate_accel = modForm.rate_braking = modForm.rate_traction = 50
  } catch (e: any) { msg.value = e?.data?.error || 'Tạo xe mod thất bại.' }
}

// ── giảm giá & trạng thái ──
const discountFor = ref<number | null>(null)
const discountPct = ref(10)
const quickPcts = [0, 5, 10, 15, 20, 30, 50]
function openDiscount(i: any) { discountFor.value = i.id; discountPct.value = Math.round(i.discount_percent) || 0 }
async function applyDiscount(id: number, pct?: number) {
  const p = pct === undefined ? Number(discountPct.value) : pct
  if (p < 0 || p > 90) { msg.value = 'Phần trăm giảm phải trong khoảng 0–90.'; return }
  try { await api.post(`/api/inventory/${id}/discount`, { percent: p }); discountFor.value = null; await refresh() }
  catch (e: any) { msg.value = e?.data?.error || 'Lỗi giảm giá.' }
}
async function setStatus(id: number, status: string) {
  try { await api.patch(`/api/inventory/${id}/status`, { status }); await refresh() }
  catch (e: any) { msg.value = e?.data?.error || 'Lỗi đổi trạng thái.' }
}
const statusLabels: Record<string, string> = { on_sale: 'Đang bán', upcoming: 'Sắp bán', hidden: 'Ẩn', sold_out: 'Hết hàng' }
const statusClass: Record<string, string> = {
  on_sale: 'bg-green-100 text-green-700', upcoming: 'bg-amber-100 text-amber-700',
  hidden: 'bg-slate-100 text-slate-500', sold_out: 'bg-red-100 text-red-600',
}
</script>

<template>
  <div>
    <h1 class="mb-4 font-serif text-2xl font-bold text-brand-900">📦 Nhập kho</h1>
    <div v-if="okMsg" class="mb-4 rounded-lg bg-green-50 px-4 py-3 text-sm text-green-700">{{ okMsg }}</div>
    <div v-if="msg" class="mb-4 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700">{{ msg }}</div>

    <div v-if="!canEdit" class="mb-6 flex items-center gap-2 rounded-lg border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-500">
      👁️ Bạn đang ở chế độ <strong class="text-slate-600">chỉ xem</strong>. Chỉ quản lý mới được nhập kho, chỉnh giá và đổi trạng thái.
    </div>

    <div v-if="canEdit" class="card mb-6 p-4">
      <div class="mb-2 flex items-center justify-between">
        <h2 class="font-semibold">Thêm xe vào kho</h2>
        <button class="text-sm text-brand-600 hover:underline" @click="showMod = !showMod">
          {{ showMod ? 'Đóng' : '+ Xe mod chưa có' }}
        </button>
      </div>

      <div v-if="showMod" class="mb-4 grid gap-2 rounded-lg bg-slate-50 p-3 sm:grid-cols-2">
        <input v-model="modForm.name" class="input" placeholder="Tên xe *" />
        <input v-model="modForm.brand" class="input" placeholder="Hãng" />
        <input v-model="modForm.class" class="input" placeholder="Class (Super, SUV…)" />
        <input v-model="modForm.model_code" class="input" placeholder="Spawn code (tuỳ chọn)" />
        <input v-model="modForm.image_url" class="input sm:col-span-2" placeholder="URL ảnh (tuỳ chọn)" />
        <textarea v-model="modForm.description" class="input sm:col-span-2" placeholder="Giới thiệu"></textarea>
        <div class="sm:col-span-2">
          <label class="label">Cốp xe (kg) *</label>
          <input v-model.number="modForm.trunk_kg" type="number" min="1" class="input" placeholder="VD: 250" />
          <p class="mt-1 text-xs font-medium text-red-600">Vui lòng điều chỉnh lại cốp xe (kg) cho chính xác trước khi tạo.</p>
        </div>
        <div class="sm:col-span-2">
          <div class="mb-1 text-xs font-medium text-slate-500">Thông số (số chỗ + điểm hiệu năng 0–100)</div>
          <div class="grid grid-cols-2 gap-2 sm:grid-cols-5">
            <input v-model.number="modForm.seats" type="number" class="input" placeholder="Số chỗ" title="Số chỗ" />
            <input v-model.number="modForm.rate_speed" type="number" class="input" placeholder="Tốc độ" title="Tốc độ 0-100" />
            <input v-model.number="modForm.rate_accel" type="number" class="input" placeholder="Tăng tốc" title="Tăng tốc 0-100" />
            <input v-model.number="modForm.rate_braking" type="number" class="input" placeholder="Phanh" title="Phanh 0-100" />
            <input v-model.number="modForm.rate_traction" type="number" class="input" placeholder="Độ bám" title="Độ bám 0-100" />
          </div>
        </div>
        <button class="btn-primary sm:col-span-2" @click="createMod">Tạo mẫu xe mod</button>
      </div>

      <div class="grid gap-4 md:grid-cols-2">
        <div>
          <label class="label">Chọn mẫu xe (từ {{ '881' }}+ xe GTA5 / xe mod)</label>
          <input v-model="catSearch" class="input" placeholder="Gõ ≥2 ký tự để tìm…" />
          <div v-if="catResults.length" class="mt-2 max-h-48 space-y-1 overflow-auto rounded-lg border p-1">
            <button
              v-for="c in catResults" :key="c.id"
              class="block w-full rounded px-2 py-1.5 text-left text-sm hover:bg-brand-50"
              :class="pickedCatalog?.id === c.id ? 'bg-brand-100' : ''"
              @click="pickedCatalog = c"
            >
              <strong>{{ c.name }}</strong> <span class="text-slate-400">· {{ c.brand }} · {{ c.class }}</span>
            </button>
          </div>
          <p v-if="pickedCatalog" class="mt-2 text-sm text-brand-700">Đã chọn: <strong>{{ pickedCatalog.name }}</strong></p>
        </div>
        <div class="space-y-2">
          <div><label class="label">Giá bán gốc</label><input v-model="stockForm.base_price" type="number" class="input" /></div>
          <div><label class="label">Số lượng</label><input v-model="stockForm.quantity" type="number" class="input" /></div>
          <div>
            <label class="label">Cốp xe (kg) *</label>
            <input v-model.number="stockForm.trunk_kg" type="number" min="1" class="input" placeholder="VD: 250" />
            <p class="mt-1 text-xs font-medium text-red-600">Vui lòng điều chỉnh lại cốp xe (kg) cho chính xác trước khi nhập kho.</p>
          </div>
          <div>
            <div class="mb-1 flex items-center justify-between">
              <label class="label !mb-0">Tuần mở bán</label>
              <button type="button" class="text-xs text-brand-600 hover:underline" @click="showNewWeek = !showNewWeek">
                {{ showNewWeek ? 'Đóng' : '+ Đăng ký tuần mới' }}
              </button>
            </div>
            <div v-if="showNewWeek" class="mb-2 flex gap-2 rounded-lg bg-slate-50 p-2">
              <input v-model="newWeekDate" type="date" class="input" title="Chọn ngày bất kỳ trong tuần (tuần luôn bắt đầu thứ 7)" />
              <button class="btn-primary !py-1.5 text-sm" @click="registerWeek">Đăng ký</button>
            </div>
            <select v-model.number="selectedWeekId" class="input">
              <option :value="null" disabled>— Chọn tuần —</option>
              <option v-for="w in weeks" :key="w.id" :value="w.id">
                {{ w.label }}{{ w.is_current ? ' (đang diễn ra)' : '' }}
              </option>
            </select>
            <p v-if="selectedWeek" class="mt-1 text-xs" :class="selectedWeek.is_current ? 'text-green-600' : 'text-amber-600'">
              → Trạng thái sẽ là: <strong>{{ selectedWeek.is_current ? 'Đang mở bán' : 'Sắp mở bán' }}</strong>
            </p>
            <p v-else-if="!weeks?.length" class="mt-1 text-xs text-slate-400">Chưa có tuần nào — hãy đăng ký tuần mới.</p>
          </div>
          <button class="btn-gold w-full" @click="addStock">Nhập kho</button>
        </div>
      </div>
    </div>

    <h2 class="mb-2 font-semibold">Danh sách kho ({{ inventory?.length || 0 }})</h2>
    <div class="card overflow-x-auto">
      <table class="w-full min-w-[760px] text-sm">
        <thead class="bg-slate-50 text-left text-xs uppercase text-slate-500">
          <tr>
            <th class="p-3">Xe</th>
            <th class="p-3">Giá bán</th>
            <th class="p-3 w-72">Khuyến mãi</th>
            <th class="p-3">Tồn</th>
            <th class="p-3">Tổng đã nhập</th>
            <th class="p-3">Trạng thái</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="i in inventory" :key="i.id" class="border-t align-top">
            <td class="p-3"><strong>{{ i.name }}</strong><div class="text-xs text-slate-400">{{ i.brand }} · {{ i.class }}</div></td>

            <!-- Giá bán: gạch giá gốc + giá sau giảm -->
            <td class="p-3">
              <template v-if="i.discount_percent > 0">
                <div class="text-xs text-slate-400 line-through">{{ formatMoney(i.base_price) }}</div>
                <div class="font-bold text-brand-800">{{ formatMoney(i.final_price) }}</div>
              </template>
              <div v-else class="font-semibold text-brand-800">{{ formatMoney(i.base_price) }}</div>
            </td>

            <!-- Khuyến mãi: badge gradient + tiết kiệm, hoặc trình chỉnh nhanh -->
            <td class="p-3">
              <!-- trình chỉnh (chỉ quản lý) -->
              <div v-if="canEdit && discountFor === i.id" class="rounded-xl border border-gold-300 bg-gold-500/5 p-2.5">
                <div class="mb-2 flex flex-wrap gap-1">
                  <button
                    v-for="p in quickPcts" :key="p"
                    class="rounded-md px-2 py-1 text-xs font-medium transition"
                    :class="Number(discountPct) === p ? 'bg-gold-500 text-brand-950' : 'bg-white text-slate-600 ring-1 ring-slate-200 hover:bg-gold-50'"
                    @click="discountPct = p"
                  >{{ p === 0 ? 'Bỏ' : `-${p}%` }}</button>
                </div>
                <div class="flex items-center gap-1.5">
                  <div class="relative flex-1">
                    <input v-model.number="discountPct" type="number" min="0" max="90" class="w-full rounded-md border px-2 py-1 pr-6 text-xs" />
                    <span class="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 text-xs text-slate-400">%</span>
                  </div>
                  <button class="rounded-md bg-brand-600 px-3 py-1 text-xs font-medium text-white hover:bg-brand-700" @click="applyDiscount(i.id)">Lưu</button>
                  <button class="rounded-md px-2 py-1 text-xs text-slate-400 hover:text-slate-600" @click="discountFor = null">✕</button>
                </div>
              </div>

              <!-- hiển thị -->
              <div v-else class="flex items-center gap-2">
                <template v-if="i.discount_percent > 0">
                  <span class="inline-flex items-center gap-1 rounded-full bg-gradient-to-r from-gold-400 to-gold-500 px-2.5 py-1 text-xs font-bold text-brand-950 shadow-sm">
                    🔥 -{{ Math.round(i.discount_percent) }}%
                  </span>
                  <span class="text-xs text-green-600">Tiết kiệm {{ formatMoney(i.base_price - i.final_price) }}</span>
                </template>
                <span v-else class="text-xs text-slate-400">Không có</span>
                <button v-if="canEdit" class="ml-auto rounded-md px-2 py-0.5 text-xs text-brand-600 ring-1 ring-brand-200 hover:bg-brand-50" @click="openDiscount(i)">
                  {{ i.discount_percent > 0 ? 'Sửa' : '+ Giảm giá' }}
                </button>
              </div>
            </td>

            <td class="p-3">{{ i.quantity }}</td>

            <!-- Tổng đã nhập -->
            <td class="p-3">
              <span class="font-medium text-brand-800">{{ i.total_imported }}</span>
              <span class="text-xs text-slate-400"> chiếc</span>
            </td>

            <!-- Trạng thái: quản lý đổi được, nhân viên chỉ thấy badge -->
            <td class="p-3">
              <select v-if="canEdit" :value="i.status" class="rounded border px-2 py-1 text-xs" @change="setStatus(i.id, ($event.target as HTMLSelectElement).value)">
                <option value="on_sale">Đang bán</option>
                <option value="upcoming">Sắp bán</option>
                <option value="hidden">Ẩn</option>
                <option value="sold_out">Hết hàng</option>
              </select>
              <span v-else class="badge" :class="statusClass[i.status]">{{ statusLabels[i.status] || i.status }}</span>
            </td>
          </tr>
          <tr v-if="!inventory?.length"><td colspan="6" class="p-8 text-center text-slate-400">Kho trống.</td></tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
