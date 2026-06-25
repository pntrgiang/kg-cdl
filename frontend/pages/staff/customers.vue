<script setup lang="ts">
definePageMeta({ layout: 'staff', middleware: 'staff' })
const api = useApi()
const auth = useAuthStore()

const search = ref('')
const { data: customers, refresh } = await useAsyncData('cust-list', () =>
  api.get<any[]>(`/api/customers${search.value ? '?search=' + encodeURIComponent(search.value) : ''}`),
)
let t: any
watch(search, () => { clearTimeout(t); t = setTimeout(() => refresh(), 300) })

const editing = ref<any>(null)
const form = reactive({ full_name: '', phone: '', national_id: '', gender: '', birth_date: '' })
const msg = ref('')
const okMsg = ref('')

// nhãn giới tính
const genderLabel = (g: string | null) =>
  g === 'male' ? 'Nam' : g === 'female' ? 'Nữ' : g === 'other' ? 'Khác' : '—'

// dropdown thao tác (mở qua Teleport để không bị cắt bởi overflow của bảng)
const menuCustomer = ref<any>(null)
const menuPos = reactive({ top: 0, left: 0 })
function toggleMenu(c: any, ev: MouseEvent) {
  if (menuCustomer.value?.id === c.id) { menuCustomer.value = null; return }
  const r = (ev.currentTarget as HTMLElement).getBoundingClientRect()
  menuPos.top = r.bottom + 4
  menuPos.left = Math.max(8, r.right - 176) // 176px = w-44
  menuCustomer.value = c
}
function runAction(fn: (c: any) => void) {
  const c = menuCustomer.value
  menuCustomer.value = null
  if (c) fn(c)
}

// giới hạn số lượng theo hạng — nhân viên/quản lý xem, chỉ dev sửa
const { data: limits, refresh: refreshLimits } = await useAsyncData('rank-limits', () => api.get<any>('/api/settings/rank-limits'))
const limForm = reactive({ svip: 3, vip: 5 })
watchEffect(() => { if (limits.value) { limForm.svip = limits.value.svip; limForm.vip = limits.value.vip } })
async function saveLimits() {
  msg.value = ''; okMsg.value = ''
  if (limForm.svip < 0 || limForm.vip < 0) { msg.value = 'Giới hạn phải ≥ 0.'; return }
  try {
    await api.put('/api/admin/rank-limits', { svip: Number(limForm.svip), vip: Number(limForm.vip) })
    okMsg.value = `Đã cập nhật giới hạn (SVIP ${limForm.svip}, VIP ${limForm.vip}) và xếp lại hạng khách.`
    await Promise.all([refreshLimits(), refresh()])
  } catch (e: any) { msg.value = e?.data?.error || 'Cập nhật giới hạn thất bại.' }
}
function startEdit(c: any) {
  editing.value = c
  form.full_name = c.full_name; form.phone = c.phone; form.national_id = c.national_id
  form.gender = c.gender || ''; form.birth_date = c.birth_date || ''
}
async function save() {
  msg.value = ''
  if (!isValidNationalID(form.national_id)) { msg.value = 'Số căn cước không hợp lệ. ' + NATIONAL_ID_HINT; return }
  try {
    await api.put(`/api/customers/${editing.value.id}`, { ...form })
    editing.value = null
    await refresh()
  } catch (e: any) { msg.value = e?.data?.error || 'Lưu thất bại.' }
}

// xem lịch sử xe khách đã mua
const salesFor = ref<any>(null)
const salesList = ref<any[]>([])
const salesLoading = ref(false)
async function openSales(c: any) {
  salesFor.value = c
  salesList.value = []
  salesLoading.value = true
  try {
    salesList.value = await api.get<any[]>(`/api/customers/${c.id}/sales`)
  } catch (e: any) { msg.value = e?.data?.error || 'Không tải được lịch sử mua.' }
  finally { salesLoading.value = false }
}
const salesSummary = computed(() => {
  const valid = salesList.value.filter((s) => !s.refunded)
  return { count: valid.length, total: valid.reduce((a, s) => a + s.final_price, 0), refunded: salesList.value.filter((s) => s.refunded).length }
})

// ── Chuyển giao dịch sang khách thật (chỉ với tài khoản tạm như LUX00000) ──
const transferFor = ref<any>(null) // sale đang chuyển
const tSearch = ref('')
const tResults = ref<any[]>([])
const tPicked = ref<any>(null)
const transferring = ref(false)
let tt: any
watch(tSearch, (q) => {
  clearTimeout(tt)
  tt = setTimeout(async () => {
    if (q.trim().length < 2) { tResults.value = []; return }
    try {
      const all = await api.get<any[]>(`/api/customers?search=${encodeURIComponent(q)}`)
      // loại tài khoản tạm + chính nguồn đang xem
      tResults.value = (all || []).filter((c) => !c.exclude_from_rank && c.id !== salesFor.value?.id)
    } catch { tResults.value = [] }
  }, 250)
})
function openTransfer(sale: any) { transferFor.value = sale; tSearch.value = ''; tResults.value = []; tPicked.value = null }
async function doTransfer() {
  if (!tPicked.value) { msg.value = 'Chọn khách hàng nhận.'; return }
  transferring.value = true; msg.value = ''; okMsg.value = ''
  try {
    await api.post(`/api/sales/${transferFor.value.id}/transfer`, { customer_id: tPicked.value.id })
    okMsg.value = `Đã chuyển "${transferFor.value.vehicle_name}" sang ${tPicked.value.full_name} (${tPicked.value.national_id}).`
    transferFor.value = null
    const src = salesFor.value
    await Promise.all([openSales(src), refresh()]) // tải lại lịch sử + danh sách khách (rank/chi tiêu đổi)
  } catch (e: any) { msg.value = e?.data?.error || 'Chuyển giao dịch thất bại.' }
  finally { transferring.value = false }
}

// reset mật khẩu khách (chỉ dev) -> đặt lại bằng chính số căn cước.
async function resetPassword(c: any) {
  msg.value = ''; okMsg.value = ''
  if (!confirm(`Đặt lại mật khẩu cho "${c.full_name}" (${c.national_id})?\nMật khẩu mới sẽ chính là số căn cước: ${c.national_id}`)) return
  try {
    await api.post(`/api/customers/${c.id}/reset-password`)
    okMsg.value = `Đã đặt lại mật khẩu của "${c.full_name}" thành số căn cước (${c.national_id}). Báo khách đăng nhập rồi đổi lại mật khẩu.`
  } catch (e: any) { msg.value = e?.data?.error || 'Đặt lại mật khẩu thất bại.' }
}

// xoá khách (chỉ dev). Khách đã có giao dịch -> ngưng hoạt động (giữ lịch sử); chưa có -> xoá hẳn.
async function removeCustomer(c: any) {
  msg.value = ''; okMsg.value = ''
  if (!confirm(`Xoá khách "${c.full_name}" (${c.national_id})?\nNếu khách đã từng mua xe, hệ thống sẽ NGƯNG hoạt động để giữ lịch sử; nếu chưa, sẽ xoá hẳn.`)) return
  try {
    const r = await api.del<any>(`/api/customers/${c.id}`)
    okMsg.value = r.hard ? `Đã xoá hẳn khách "${c.full_name}".` : `Khách "${c.full_name}" đã có giao dịch nên được ngưng hoạt động (giữ lịch sử).`
    await refresh()
  } catch (e: any) { msg.value = e?.data?.error || 'Xoá khách thất bại.' }
}
</script>

<template>
  <div>
    <div class="mb-4 flex items-center justify-between">
      <h1 class="font-serif text-2xl font-bold text-brand-900">👥 Khách hàng</h1>
      <input v-model="search" class="input max-w-xs" placeholder="Tìm tên, SĐT, căn cước…" />
    </div>
    <p v-if="!auth.isManager" class="mb-3 text-sm text-slate-500">Bạn có quyền xem. Chỉ quản lý mới được cập nhật.</p>
    <div v-if="okMsg" class="mb-3 rounded-lg bg-green-50 px-4 py-2 text-sm text-green-700">{{ okMsg }}</div>
    <div v-if="msg" class="mb-3 rounded-lg bg-red-50 px-4 py-2 text-sm text-red-700">{{ msg }}</div>

    <!-- giới hạn số lượng theo hạng (nhân viên/quản lý xem, chỉ dev sửa) -->
    <div class="card mb-4 p-4">
      <h2 class="mb-1 font-semibold">Giới hạn số lượng khách hàng theo hạng</h2>
      <p class="mb-3 text-xs text-slate-500">
        Hạng tính theo tổng chi tiêu; còn lại là phổ thông.
        <span v-if="auth.isDev">Đổi giới hạn sẽ tự xếp lại hạng toàn bộ khách.</span>
        <span v-else>Chỉ Dev mới được chỉnh sửa số lượng này.</span>
      </p>
      <div class="flex flex-wrap items-end gap-3">
        <div>
          <label class="label">Số SVIP tối đa</label>
          <input
            v-model.number="limForm.svip" type="number" min="0"
            class="input w-32 disabled:cursor-not-allowed disabled:bg-slate-100 disabled:text-slate-500"
            :disabled="!auth.isDev"
          />
        </div>
        <div>
          <label class="label">Số VIP tối đa</label>
          <input
            v-model.number="limForm.vip" type="number" min="0"
            class="input w-32 disabled:cursor-not-allowed disabled:bg-slate-100 disabled:text-slate-500"
            :disabled="!auth.isDev"
          />
        </div>
        <button v-if="auth.isDev" class="btn-primary" @click="saveLimits">Lưu giới hạn</button>
      </div>
    </div>

    <div class="card overflow-x-auto">
      <table class="w-full min-w-[640px] text-sm">
        <thead class="bg-slate-50 text-left text-xs uppercase text-slate-500">
          <tr><th class="p-3">Họ tên</th><th class="p-3">SĐT</th><th class="p-3">Căn cước</th><th class="p-3">Giới tính</th><th class="p-3">Ngày sinh</th><th class="p-3">Hạng</th><th class="p-3">Đã mua</th><th class="p-3"></th></tr>
        </thead>
        <tbody>
          <tr v-for="c in customers" :key="c.id" class="border-t" :class="c.is_active === false ? 'opacity-50' : ''">
            <td class="p-3 font-medium">
              {{ c.full_name }}
              <span v-if="c.is_active === false" class="badge ml-1 bg-slate-200 text-slate-500">Đã ngưng</span>
            </td>
            <td class="p-3">{{ c.phone || '—' }}</td>
            <td class="p-3">{{ c.national_id }}</td>
            <td class="p-3">{{ genderLabel(c.gender) }}</td>
            <td class="whitespace-nowrap p-3">{{ c.birth_date || '—' }}</td>
            <td class="p-3">
              <span v-if="c.exclude_from_rank" class="badge bg-slate-100 text-slate-500" title="Tài khoản tạm — nhận xe thất lạc thông tin khách, không xếp hạng">🏷️ Tài khoản tạm</span>
              <RankBadge v-else :rank="c.rank" />
            </td>
            <td class="p-3 font-medium text-brand-800">{{ formatMoney(c.total_spent) }}</td>
            <td class="p-3 text-right">
              <button
                class="inline-flex items-center gap-1 rounded-md px-2.5 py-1 text-xs text-slate-600 ring-1 ring-slate-200 hover:bg-slate-50"
                @click="toggleMenu(c, $event)"
              >
                Thao tác <span class="text-[10px]">▾</span>
              </button>
            </td>
          </tr>
          <tr v-if="!customers?.length"><td colspan="8" class="p-8 text-center text-slate-400">Chưa có khách hàng.</td></tr>
        </tbody>
      </table>
    </div>

    <!-- dropdown thao tác (Teleport ra body để không bị cắt bởi overflow của bảng) -->
    <Teleport to="body">
      <div v-if="menuCustomer" class="fixed inset-0 z-40" @click="menuCustomer = null" />
      <div
        v-if="menuCustomer"
        class="fixed z-50 w-44 overflow-hidden rounded-lg border bg-white py-1 shadow-xl"
        :style="{ top: menuPos.top + 'px', left: menuPos.left + 'px' }"
      >
        <button class="block w-full px-3 py-1.5 text-left text-xs text-slate-700 hover:bg-brand-50" @click="runAction(openSales)">🚗 Xe đã mua</button>
        <button v-if="auth.isManager" class="block w-full px-3 py-1.5 text-left text-xs text-slate-700 hover:bg-brand-50" @click="runAction(startEdit)">✏️ Sửa thông tin</button>
        <button v-if="auth.isDev" class="block w-full px-3 py-1.5 text-left text-xs text-amber-600 hover:bg-amber-50" @click="runAction(resetPassword)">🔑 Reset mật khẩu</button>
        <button v-if="auth.isDev" class="block w-full border-t px-3 py-1.5 text-left text-xs text-red-600 hover:bg-red-50" @click="runAction(removeCustomer)">🗑 Xoá khách</button>
      </div>
    </Teleport>

    <!-- modal lịch sử xe đã mua -->
    <div v-if="salesFor" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" @click.self="salesFor = null">
      <div class="card flex max-h-[85vh] w-full max-w-2xl flex-col p-5">
        <div class="mb-3 flex items-start justify-between gap-2">
          <div>
            <h2 class="font-semibold text-brand-900">🚗 Xe đã mua — {{ salesFor.full_name }}</h2>
            <p class="text-xs text-slate-500">{{ salesFor.national_id }}<span v-if="salesFor.phone"> · {{ salesFor.phone }}</span></p>
            <p v-if="salesFor.exclude_from_rank" class="mt-1 text-xs text-amber-600">🏷️ Tài khoản tạm — bấm <strong>Chuyển</strong> để chuyển từng giao dịch sang khách hàng thật.</p>
          </div>
          <button class="text-slate-400 hover:text-slate-700" @click="salesFor = null">✕</button>
        </div>

        <div v-if="salesLoading" class="py-8 text-center text-sm text-slate-400">Đang tải…</div>
        <template v-else>
          <div class="mb-3 flex flex-wrap gap-4 rounded-lg bg-slate-50 p-3 text-sm">
            <div><span class="text-slate-500">Số xe đã mua:</span> <strong class="text-brand-800">{{ salesSummary.count }}</strong></div>
            <div><span class="text-slate-500">Tổng chi:</span> <strong class="text-gold-600">{{ formatMoney(salesSummary.total) }}</strong></div>
            <div v-if="salesSummary.refunded > 0"><span class="text-slate-500">Đã hoàn:</span> <strong class="text-red-600">{{ salesSummary.refunded }} xe</strong></div>
          </div>

          <div class="-mx-1 flex-1 overflow-auto">
            <table class="w-full min-w-[480px] text-sm">
              <thead class="bg-slate-50 text-left text-xs uppercase text-slate-500">
                <tr><th class="p-2.5">Xe</th><th class="p-2.5">Thời gian</th><th class="p-2.5">NV bán</th><th class="p-2.5 text-right">Thành tiền</th><th v-if="salesFor.exclude_from_rank" class="p-2.5"></th></tr>
              </thead>
              <tbody>
                <tr v-for="s in salesList" :key="s.id" class="border-t" :class="s.refunded ? 'text-slate-400' : ''">
                  <td class="p-2.5 font-medium" :class="s.refunded ? 'line-through' : ''">
                    {{ s.vehicle_name }}
                    <span v-if="s.refunded" class="badge ml-1 bg-red-100 text-red-600">Đã hoàn</span>
                  </td>
                  <td class="whitespace-nowrap p-2.5 text-xs text-slate-500">{{ formatDateTime(s.created_at) }}</td>
                  <td class="p-2.5 text-slate-500">{{ s.sold_by_name }}</td>
                  <td class="p-2.5 text-right font-semibold" :class="s.refunded ? 'line-through' : 'text-brand-800'">
                    {{ formatMoney(s.final_price) }}
                    <div v-if="s.voucher_discount > 0" class="text-[11px] font-normal text-green-600">voucher -{{ formatMoney(s.voucher_discount) }}</div>
                  </td>
                  <td v-if="salesFor.exclude_from_rank" class="p-2.5 text-right">
                    <button v-if="!s.refunded && auth.isManager" class="rounded-md px-2 py-0.5 text-xs text-brand-600 ring-1 ring-brand-200 hover:bg-brand-50" @click="openTransfer(s)">Chuyển</button>
                  </td>
                </tr>
                <tr v-if="!salesList.length"><td :colspan="salesFor.exclude_from_rank ? 5 : 4" class="p-8 text-center text-slate-400">Khách này chưa mua xe nào.</td></tr>
              </tbody>
            </table>
          </div>
        </template>
      </div>
    </div>

    <!-- modal chuyển giao dịch sang khách thật -->
    <div v-if="transferFor" class="fixed inset-0 z-[60] flex items-center justify-center bg-black/50 p-4" @click.self="transferFor = null">
      <div class="card w-full max-w-md p-5">
        <h3 class="mb-1 font-semibold text-brand-900">Chuyển giao dịch sang khách hàng thật</h3>
        <p class="mb-3 text-xs text-slate-500">
          Xe: <strong>{{ transferFor.vehicle_name }}</strong> · {{ formatMoney(transferFor.final_price) }}.
          Chi tiêu & hạng thành viên sẽ được tính lại cho khách nhận.
        </p>
        <label class="label">Tìm khách hàng nhận</label>
        <input v-model="tSearch" class="input" placeholder="Tìm theo tên, SĐT, căn cước…" />
        <div v-if="tResults.length" class="mt-1 max-h-44 space-y-1 overflow-auto rounded-lg border bg-white p-1">
          <button
            v-for="c in tResults" :key="c.id" type="button"
            class="block w-full rounded px-2 py-1.5 text-left text-sm hover:bg-brand-50"
            :class="tPicked?.id === c.id ? 'bg-brand-100' : ''"
            @click="tPicked = c; tResults = []; tSearch = ''"
          >
            <strong>{{ c.full_name }}</strong><span class="text-slate-400"> · {{ c.national_id }}<span v-if="c.phone"> · {{ c.phone }}</span></span>
          </button>
        </div>
        <div v-if="tPicked" class="mt-2 rounded-md bg-brand-50 px-3 py-2 text-sm text-brand-800 ring-1 ring-brand-100">✓ {{ tPicked.full_name }} ({{ tPicked.national_id }})</div>
        <div class="mt-5 flex justify-end gap-2">
          <button class="btn-ghost" @click="transferFor = null">Hủy</button>
          <button class="btn-primary" :disabled="!tPicked || transferring" @click="doTransfer">{{ transferring ? 'Đang chuyển…' : 'Chuyển' }}</button>
        </div>
      </div>
    </div>

    <!-- modal sửa -->
    <div v-if="editing" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" @click.self="editing = null">
      <div class="card w-full max-w-md p-5">
        <h2 class="mb-4 font-semibold">Cập nhật khách hàng</h2>
        <div class="space-y-3">
          <div><label class="label">Họ tên</label><input v-model="form.full_name" class="input" /></div>
          <div><label class="label">Số điện thoại</label><input v-model="form.phone" class="input" /></div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="label">Giới tính</label>
              <select v-model="form.gender" class="input">
                <option value="">— Chưa rõ —</option>
                <option value="male">Nam</option>
                <option value="female">Nữ</option>
                <option value="other">Khác</option>
              </select>
            </div>
            <div>
              <label class="label">Ngày sinh</label>
              <input v-model="form.birth_date" type="date" class="input" />
            </div>
          </div>
          <div>
            <label class="label">Số căn cước</label>
            <input v-model="form.national_id" class="input uppercase placeholder:normal-case" placeholder="LUX12345" @input="form.national_id = form.national_id.toUpperCase()" />
            <p class="mt-1 text-xs text-slate-400">{{ NATIONAL_ID_HINT }}</p>
          </div>
        </div>
        <div class="mt-5 flex justify-end gap-2">
          <button class="btn-ghost" @click="editing = null">Hủy</button>
          <button class="btn-primary" @click="save">Lưu</button>
        </div>
      </div>
    </div>
  </div>
</template>
