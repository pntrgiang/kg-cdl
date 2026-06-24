<script setup lang="ts">
definePageMeta({ layout: 'staff', middleware: 'manager' })
const api = useApi()

// danh sách voucher (chỉ để chọn làm phần thưởng — quản lý tạo voucher ở tab Voucher)
const { data: vouchers } = await useAsyncData('vouchers', () => api.get<any[]>('/api/vouchers'))
// chỉ voucher còn hiệu lực (chưa bị huỷ) mới được chọn làm phần thưởng
const availableVouchers = computed(() => (vouchers.value || []).filter((v: any) => !v.cancelled_at))
const { data: events, refresh: refreshEvents } = await useAsyncData('mgr-events', () => api.get<any[]>('/api/manage/events'))
const msg = ref(''); const okMsg = ref('')

// ── tạo sự kiện quay số (thưởng là voucher) ──
const eForm = reactive({
  title: '', description: '', deadline: '',
  voucher_id: null as number | null, winners_count: 1,
})

// hạn đăng ký (cuối ngày, GMT+7) để so với hạn voucher
const deadlineEnd = computed(() => (eForm.deadline ? new Date(`${eForm.deadline}T23:59:59+07:00`) : null))
// voucher hợp lệ làm thưởng: HSD phải SAU hạn đăng ký (người trúng nhận voucher sau khi quay số)
function voucherValid(v: any) {
  if (!v?.expires_at || !deadlineEnd.value) return true
  return new Date(v.expires_at) > deadlineEnd.value
}
const selectedVoucher = computed(() => availableVouchers.value.find((v: any) => v.id === eForm.voucher_id) || null)
// cảnh báo khi voucher đã chọn hết hạn ≤ hạn đăng ký (vd khi đổi hạn đăng ký muộn hơn HSD voucher)
const voucherWarn = computed(() => {
  const v = selectedVoucher.value
  if (!v || voucherValid(v)) return ''
  return `⚠️ Voucher "${v.name}" hết hạn ${formatDate(v.expires_at)} — phải SAU hạn đăng ký (${formatDate(eForm.deadline)}). Hãy chọn voucher khác hoặc đặt hạn đăng ký sớm hơn.`
})

async function createEvent() {
  msg.value = ''; okMsg.value = ''
  if (!eForm.title.trim()) { msg.value = 'Cần nhập tiêu đề sự kiện.'; return }
  if (!eForm.deadline) { msg.value = 'Cần chọn hạn đăng ký.'; return }
  if (!(Number(eForm.winners_count) >= 1)) { msg.value = 'Số người trúng phải ≥ 1.'; return }
  if (!eForm.voucher_id) { msg.value = 'Cần chọn voucher làm phần thưởng (tạo ở tab Voucher nếu chưa có).'; return }
  if (voucherWarn.value) { msg.value = voucherWarn.value; return }
  try {
    await api.post('/api/events/draw', {
      title: eForm.title, description: eForm.description,
      register_deadline: `${eForm.deadline}T23:59:59+07:00`,
      voucher_id: eForm.voucher_id, winners_count: Number(eForm.winners_count),
    })
    okMsg.value = 'Đã tạo sự kiện quay số.'
    eForm.title = eForm.description = ''
    await refreshEvents()
  } catch (e: any) { msg.value = e?.data?.error || 'Tạo sự kiện thất bại.' }
}

// ── huỷ sự kiện (chỉ khi chưa quay số, bắt buộc lý do) ──
const cancelFor = ref<number | null>(null)
const cancelReason = ref('')
function openCancel(e: any) { cancelFor.value = e.id; cancelReason.value = ''; msg.value = ''; okMsg.value = '' }
async function doCancel(e: any) {
  msg.value = ''
  if (!cancelReason.value.trim()) { msg.value = 'Bắt buộc nhập lý do huỷ sự kiện.'; return }
  try {
    await api.post(`/api/events/${e.id}/cancel`, { reason: cancelReason.value.trim() })
    okMsg.value = `Đã huỷ sự kiện "${e.title}".`
    cancelFor.value = null
    await refreshEvents()
  } catch (err: any) { msg.value = err?.data?.error || 'Huỷ sự kiện thất bại.' }
}

// ── quay số: mở modal có vòng quay, chỉ quay khi bấm "Bắt đầu" ──
const drawModal = ref<any>(null) // event đang xem
const entrants = ref<any[]>([])  // khách đã đăng ký (gán lên vòng quay)
const wheel = ref<any>(null)     // ref tới component LuckyWheel
const spinning = ref(false)
const drawWinners = ref<any[]>([])

async function openDraw(ev: any) {
  msg.value = ''
  drawModal.value = ev
  drawWinners.value = []
  entrants.value = []
  showRedraw.value = false
  redrawReason.value = ''
  spinning.value = false
  try { entrants.value = await api.get<any[]>(`/api/events/${ev.id}/entrants`) } catch { entrants.value = [] }
  if (ev.draw_status !== 'open') {
    // đã quay / công bố -> tải kết quả sẵn có (không tự quay)
    try {
      const full = await api.get<any>(`/api/events/${ev.id}`)
      drawModal.value = full
      drawWinners.value = full.winners || []
    } catch (e: any) { msg.value = e?.data?.error || 'Không tải được kết quả.' }
  }
}

async function startSpin(ev: any) {
  if (!entrants.value.length) { msg.value = 'Chưa có khách nào đăng ký để quay.'; return }
  msg.value = ''; spinning.value = true; drawWinners.value = []
  try {
    const winners = await api.post<any[]>(`/api/events/${ev.id}/draw`)
    await wheel.value?.spin(winners[0]?.customer_id)
    drawWinners.value = winners
    drawModal.value = { ...drawModal.value, draw_status: 'drawn' }
    await refreshEvents()
  } catch (e: any) { msg.value = e?.data?.error || 'Quay số thất bại.' }
  finally { spinning.value = false }
}

async function confirmDraw(ev: any) {
  try {
    await api.post(`/api/events/${ev.id}/confirm`)
    okMsg.value = 'Đã xác nhận & công bố kết quả.'
    drawModal.value = null
    await refreshEvents()
  } catch (e: any) { msg.value = e?.data?.error || 'Xác nhận thất bại.' }
}

// quay lại (bắt buộc lý do) -> quay lại vòng quay tới người trúng mới
const showRedraw = ref(false)
const redrawReason = ref('')
async function submitRedraw(ev: any) {
  if (!redrawReason.value.trim()) return
  msg.value = ''; spinning.value = true; showRedraw.value = false; drawWinners.value = []
  try {
    const winners = await api.post<any[]>(`/api/events/${ev.id}/redraw`, { reason: redrawReason.value.trim() })
    await wheel.value?.spin(winners[0]?.customer_id)
    drawWinners.value = winners
    redrawReason.value = ''
    await refreshEvents()
  } catch (e: any) { msg.value = e?.data?.error || 'Quay lại thất bại.' }
  finally { spinning.value = false }
}

const statusLabel: Record<string, { t: string; c: string }> = {
  open: { t: 'Đang mở', c: 'bg-green-100 text-green-700' },
  drawn: { t: 'Đã quay (chờ xác nhận)', c: 'bg-amber-100 text-amber-700' },
  published: { t: 'Đã công bố', c: 'bg-brand-100 text-brand-700' },
}
const fmtDate = (s: string) => formatDate(s)
</script>

<template>
  <div>
    <h1 class="mb-4 font-serif text-2xl font-bold text-brand-900">🎉 Sự kiện (Quản lý)</h1>
    <div v-if="okMsg" class="mb-4 rounded-lg bg-green-50 px-4 py-3 text-sm text-green-700">{{ okMsg }}</div>
    <div v-if="msg" class="mb-4 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700">{{ msg }}</div>

    <!-- TẠO SỰ KIỆN -->
    <div class="card mb-6 p-4">
      <h2 class="mb-3 font-semibold">Tạo sự kiện quay số trúng thưởng</h2>
      <div class="grid gap-3 md:grid-cols-2">
        <div><label class="label">Tiêu đề</label><input v-model="eForm.title" class="input" /></div>
        <div><label class="label">Hạn đăng ký</label><input v-model="eForm.deadline" type="date" class="input" /></div>
        <div class="md:col-span-2"><label class="label">Mô tả</label><textarea v-model="eForm.description" class="input"></textarea></div>
        <div><label class="label">Số người trúng</label><input v-model.number="eForm.winners_count" type="number" class="input" /></div>
        <div>
          <label class="label">Voucher làm phần thưởng</label>
          <select v-model="eForm.voucher_id" class="input">
            <option :value="null" disabled>— Chọn voucher —</option>
            <option v-for="v in availableVouchers" :key="v.id" :value="v.id" :disabled="!voucherValid(v)">
              {{ v.name }} ({{ v.discount_percent }}%) — HSD {{ fmtDate(v.expires_at) }}{{ !voucherValid(v) ? ' · hết hạn ≤ hạn ĐK' : '' }}
            </option>
          </select>
          <p v-if="voucherWarn" class="mt-1 text-xs font-medium text-red-600">{{ voucherWarn }}</p>
          <p v-else-if="eForm.deadline" class="mt-1 text-xs text-slate-400">Chỉ chọn được voucher có hạn sử dụng <strong>sau</strong> hạn đăng ký ({{ fmtDate(eForm.deadline) }}).</p>
          <p v-if="!availableVouchers.length" class="mt-1 text-xs text-slate-400">
            Chưa có voucher — hãy tạo ở tab <NuxtLink to="/staff/vouchers" class="text-brand-600 hover:underline">Voucher</NuxtLink> trước.
          </p>
        </div>

        <button class="btn-gold md:col-span-2 disabled:cursor-not-allowed disabled:opacity-50" :disabled="!!voucherWarn" @click="createEvent">Tạo sự kiện</button>
      </div>
    </div>

    <!-- DANH SÁCH SỰ KIỆN -->
    <h2 class="mb-2 font-semibold">Sự kiện quay số</h2>
    <div class="space-y-3">
      <div v-for="e in (events || []).filter((x:any)=>x.draw_status)" :key="e.id" class="card p-4">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div>
            <strong class="text-brand-900" :class="e.cancelled_at ? 'text-slate-400 line-through' : ''">{{ e.title }}</strong>
            <span v-if="e.cancelled_at" class="badge ml-2 bg-red-100 text-red-600">Đã huỷ</span>
            <span v-else class="badge ml-2" :class="statusLabel[e.draw_status]?.c">{{ statusLabel[e.draw_status]?.t }}</span>
          </div>
          <div class="flex gap-2">
            <template v-if="!e.cancelled_at">
              <button v-if="e.draw_status === 'open'" class="btn-primary !py-1.5 text-xs" @click="openDraw(e)">🎰 Quay số</button>
              <button v-if="e.draw_status === 'open'" class="rounded-lg px-3 py-1.5 text-xs font-medium text-red-600 ring-1 ring-red-200 hover:bg-red-50" @click="openCancel(e)">Huỷ</button>
              <button v-if="e.draw_status === 'drawn'" class="btn-gold !py-1.5 text-xs" @click="openDraw(e)">Xem & xác nhận</button>
              <button v-if="e.draw_status === 'published'" class="btn-ghost !py-1.5 text-xs" @click="openDraw(e)">👁️ Xem chi tiết</button>
            </template>
          </div>
        </div>
        <div class="mt-2 flex flex-wrap gap-2 text-xs">
          <span class="badge bg-slate-100 text-slate-600">🎟️ Thưởng: {{ e.prize_name }}</span>
          <span class="badge bg-slate-100 text-slate-600">Số trúng: {{ e.winners_count }}</span>
          <span class="badge bg-slate-100 text-slate-600">Hạn: {{ fmtDate(e.register_deadline) }}</span>
          <span class="badge bg-slate-100 text-slate-600">Đã đăng ký: {{ e.eligible_count }}</span>
        </div>

        <p v-if="e.cancelled_at" class="mt-2 text-xs text-red-500">↳ Đã huỷ{{ e.cancel_reason ? ' — lý do: ' + e.cancel_reason : '' }}</p>

        <!-- form huỷ sự kiện -->
        <div v-if="cancelFor === e.id" class="mt-3 rounded-lg border border-red-200 bg-red-50 p-3">
          <label class="label">Lý do huỷ sự kiện (bắt buộc)</label>
          <input v-model="cancelReason" class="input !py-1.5 text-sm" placeholder="VD: tạo nhầm, đổi thể lệ…" @keyup.enter="doCancel(e)" />
          <p class="mt-1 text-xs text-slate-500">Chỉ huỷ được sự kiện <strong>chưa quay số</strong>. Sự kiện đã huỷ vẫn còn trong danh sách (đánh dấu "Đã huỷ") nhưng ẩn với khách hàng.</p>
          <div class="mt-2 flex justify-end gap-2">
            <button class="btn-ghost !py-1.5 text-sm" @click="cancelFor = null">Đóng</button>
            <button class="rounded-md bg-red-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-red-700" @click="doCancel(e)">Xác nhận huỷ</button>
          </div>
        </div>
        <div v-if="e.winners?.length" class="mt-3 border-t pt-2">
          <div class="mb-1 text-xs font-medium text-slate-500">Người trúng:</div>
          <div class="flex flex-wrap gap-2">
            <span v-for="w in e.winners" :key="w.id" class="badge bg-gold-500/20 text-brand-900">
              🏆 {{ w.customer_name }}<span v-if="w.fulfilled_at"> ✓đã giao</span>
            </span>
          </div>
        </div>
      </div>
      <p v-if="!(events||[]).some((x:any)=>x.draw_status)" class="card p-8 text-center text-slate-400">Chưa có sự kiện quay số.</p>
    </div>

    <!-- MODAL QUAY SỐ (vòng quay may mắn) -->
    <div v-if="drawModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-3 sm:p-4" @click.self="!spinning && (drawModal = null)">
      <div class="card max-h-[92vh] w-full max-w-4xl overflow-auto p-5 sm:p-6">
        <div class="mb-4 flex items-start justify-between gap-2">
          <div>
            <h3 class="font-serif text-lg font-bold text-brand-900">🎰 {{ drawModal.title }}</h3>
            <div class="mt-1 flex flex-wrap gap-2 text-xs text-slate-500">
              <span class="badge" :class="statusLabel[drawModal.draw_status]?.c">{{ statusLabel[drawModal.draw_status]?.t }}</span>
              <span>🎁 {{ drawModal.prize_name }}</span>
              <span>· Số trúng: {{ drawModal.winners_count }}</span>
              <span>· Hạn ĐK: {{ fmtDate(drawModal.register_deadline) }}</span>
            </div>
          </div>
          <button v-if="!spinning" class="text-slate-400 hover:text-slate-700" @click="drawModal = null">✕</button>
        </div>

        <div class="grid gap-6 lg:grid-cols-2">
          <!-- VÒNG QUAY -->
          <div>
            <LuckyWheel ref="wheel" :entries="entrants" />
          </div>

          <!-- DANH SÁCH THAM GIA + KẾT QUẢ + HÀNH ĐỘNG -->
          <div class="flex flex-col">
            <!-- danh sách tham gia -->
            <div class="mb-4">
              <div class="mb-1 text-xs font-medium text-slate-500">Danh sách tham gia ({{ entrants.length }} người)</div>
              <div v-if="entrants.length" class="flex max-h-28 flex-wrap gap-1.5 overflow-auto rounded-lg bg-slate-50 p-2">
                <span v-for="(e, i) in entrants" :key="e.customer_id" class="badge bg-white text-slate-600 ring-1 ring-slate-200">
                  {{ i + 1 }}. {{ e.customer_name }}
                </span>
              </div>
              <p v-else class="rounded-lg bg-amber-50 p-3 text-center text-sm text-amber-700">Chưa có khách nào đăng ký tham gia.</p>
            </div>

            <div v-if="spinning" class="rounded-lg bg-brand-50 px-4 py-3 text-center text-sm font-medium text-brand-700">
              🎲 Đang quay chọn người trúng…
            </div>

            <!-- kết quả (dưới danh sách tham gia) -->
            <div v-if="drawWinners.length && !spinning" class="mb-4">
              <div class="mb-2 text-sm font-semibold text-brand-800">
                {{ drawModal.draw_status === 'published' ? '🏆 Kết quả đã công bố' : '🏆 Kết quả (chờ xác nhận)' }}
              </div>
              <div class="space-y-1.5">
                <div v-for="w in drawWinners" :key="w.id" class="rounded-lg bg-gold-500/15 px-3 py-2 font-medium text-brand-900">
                  🏆 {{ w.customer_name }}<span v-if="w.fulfilled_at" class="ml-1 text-xs text-green-600">✓ đã giao</span>
                </div>
              </div>
            </div>

            <!-- ô lý do quay lại -->
            <div v-if="showRedraw" class="mb-3 rounded-lg bg-amber-50 p-3">
              <label class="label">Lý do quay số lại (bắt buộc)</label>
              <textarea v-model="redrawReason" rows="2" class="input" placeholder="VD: người trúng không đủ điều kiện…"></textarea>
              <div class="mt-2 flex justify-end gap-2">
                <button class="btn-ghost !py-1.5 text-sm" @click="showRedraw = false; redrawReason = ''">Hủy</button>
                <button class="btn-primary !py-1.5 text-sm" :disabled="!redrawReason.trim()" @click="submitRedraw(drawModal)">Xác nhận quay lại</button>
              </div>
            </div>

            <!-- nút hành động -->
            <div v-if="!spinning && !showRedraw" class="mt-auto flex flex-wrap gap-2 pt-2">
              <template v-if="drawModal.draw_status === 'open'">
                <button class="btn-gold flex-1" :disabled="!entrants.length" @click="startSpin(drawModal)">🎯 Bắt đầu quay số</button>
                <button class="btn-ghost" @click="drawModal = null">Đóng</button>
              </template>
              <template v-else-if="drawModal.draw_status === 'drawn'">
                <button class="btn-gold flex-1" @click="confirmDraw(drawModal)">✓ Xác nhận &amp; công bố</button>
                <button class="btn-ghost !border-amber-400 !text-amber-700" @click="showRedraw = true">🔄 Quay lại</button>
                <button class="btn-ghost" @click="drawModal = null">Đóng</button>
              </template>
              <template v-else>
                <button class="btn-ghost ml-auto" @click="drawModal = null">Đóng</button>
              </template>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
