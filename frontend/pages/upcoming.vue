<script setup lang="ts">
const api = useApi()
const auth = useAuthStore()

const { data: items, pending } = await useAsyncData('vehicles-upcoming', () =>
  api.get<any[]>('/api/vehicles?status=upcoming'),
)

useSeo({
  title: 'Xe sắp mở bán',
  description:
    'Những mẫu xe sắp ra mắt tại Kanji Group — Lux City. Xem trước thông số và đếm ngược thời gian mở bán để không bỏ lỡ.',
})

// ── countdown mở bán xe mới ──
const { data: release, refresh: refreshRelease } = await useAsyncData('release-info', () =>
  api.get<{ release_at: string; default_at: string; overridden: boolean; modal_image: string }>('/api/release-info'),
)
const releaseTs = computed(() => (release.value ? new Date(release.value.release_at).getTime() : 0))

const nowTs = ref(Date.now())
let timer: any = null
onMounted(() => {
  nowTs.value = Date.now()
  timer = setInterval(async () => {
    nowTs.value = Date.now()
    // hết giờ -> tải lại mốc mới (override hết hạn sẽ tự về mặc định thứ 7 21:00)
    if (releaseTs.value && nowTs.value >= releaseTs.value) {
      await refreshRelease()
    }
  }, 1000)
})
onBeforeUnmount(() => { if (timer) clearInterval(timer) })

const remaining = computed(() => Math.max(0, releaseTs.value - nowTs.value))
const parts = computed(() => {
  const t = remaining.value
  return {
    d: Math.floor(t / 86400000),
    h: Math.floor((t % 86400000) / 3600000),
    m: Math.floor((t % 3600000) / 60000),
    s: Math.floor((t % 60000) / 1000),
  }
})
const pad = (n: number) => String(n).padStart(2, '0')

// nhãn ngày giờ chính xác (GMT+7)
const TZ = 'Asia/Ho_Chi_Minh'
function weekdayLabel(iso: string) {
  return new Intl.DateTimeFormat('vi-VN', { timeZone: TZ, weekday: 'long' }).format(new Date(iso))
}
const exactLabel = computed(() => {
  if (!release.value) return ''
  const iso = release.value.release_at
  return `${weekdayLabel(iso)}, ${formatDateTime(iso)} (GMT+7)`
})

// ── quản lý chỉnh sửa mốc countdown (chỉ tuần này) ──
const editing = ref(false)
const customDT = ref('')
const editMsg = ref(''); const editOk = ref('')
function gmt7InputValue(iso: string) {
  const f = new Intl.DateTimeFormat('en-CA', {
    timeZone: TZ, year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit', hourCycle: 'h23',
  })
  const p: Record<string, string> = {}
  for (const x of f.formatToParts(new Date(iso))) p[x.type] = x.value
  return `${p.year}-${p.month}-${p.day}T${p.hour}:${p.minute}`
}
function openEdit() {
  editMsg.value = ''; editOk.value = ''
  customDT.value = release.value ? gmt7InputValue(release.value.release_at) : ''
  editing.value = true
}
async function saveCustom() {
  editMsg.value = ''
  if (!customDT.value) { editMsg.value = 'Hãy chọn thời điểm mở bán.'; return }
  try {
    await api.put('/api/release-override', { release_at: `${customDT.value}:00+07:00` })
    editing.value = false; editOk.value = 'Đã cập nhật mốc mở bán cho tuần này.'
    await refreshRelease()
  } catch (e: any) { editMsg.value = e?.data?.error || 'Cập nhật thất bại.' }
}
async function resetDefault() {
  editMsg.value = ''
  try {
    await api.put('/api/release-override', { reset: true })
    editing.value = false; editOk.value = 'Đã trở về mặc định (thứ Bảy 21:00).'
    await refreshRelease()
  } catch (e: any) { editMsg.value = e?.data?.error || 'Đặt lại thất bại.' }
}
</script>

<template>
  <section>
    <h1 class="mb-4 text-xl font-bold">🔜 Xe sắp mở bán</h1>

    <!-- bảng countdown -->
    <div v-if="release" class="mb-6 overflow-hidden rounded-2xl bg-gradient-to-br from-brand-900 to-brand-800 p-5 text-white shadow-lg sm:p-6">
      <div v-if="release.overridden" class="mb-3 text-center">
        <span class="badge bg-gold-500/20 text-gold-300">Lịch đặc biệt tuần này</span>
      </div>

      <ClientOnly>
        <div class="mx-auto grid max-w-md grid-cols-4 gap-2 sm:gap-3">
          <div v-for="b in [{v: parts.d, l: 'Ngày'}, {v: parts.h, l: 'Giờ'}, {v: parts.m, l: 'Phút'}, {v: parts.s, l: 'Giây'}]" :key="b.l"
            class="rounded-xl bg-white/10 py-3 text-center">
            <div class="font-serif text-2xl font-bold tabular-nums text-gold-400 sm:text-3xl">{{ pad(b.v) }}</div>
            <div class="text-[10px] uppercase tracking-wider text-brand-200 sm:text-xs">{{ b.l }}</div>
          </div>
        </div>
        <template #fallback>
          <div class="mx-auto h-[88px] max-w-md animate-pulse rounded-xl bg-white/10" />
        </template>
      </ClientOnly>

      <p class="mt-3 text-center text-sm text-brand-100">
        🗓️ Mở bán: <strong class="text-white">{{ exactLabel }}</strong>
      </p>

      <!-- quản lý: chỉnh sửa mốc -->
      <div v-if="auth.isManager" class="mt-3 border-t border-white/10 pt-3 text-center">
        <p v-if="editOk" class="mb-2 text-xs text-green-300">{{ editOk }}</p>
        <button v-if="!editing" class="text-xs font-medium text-gold-300 hover:underline" @click="openEdit">✏️ Chỉnh mốc mở bán cho tuần này</button>
        <div v-else class="mx-auto max-w-md rounded-lg bg-white/10 p-3 text-left">
          <label class="mb-1 block text-xs text-brand-100">Thời điểm mở bán (GMT+7) — chỉ áp dụng tuần này:</label>
          <input v-model="customDT" type="datetime-local" class="w-full rounded-md border-0 px-2 py-1.5 text-sm text-slate-800" />
          <p v-if="editMsg" class="mt-1 text-xs text-red-300">{{ editMsg }}</p>
          <div class="mt-2 flex flex-wrap gap-2">
            <button class="rounded-md bg-gold-500 px-3 py-1 text-xs font-semibold text-brand-950 hover:bg-gold-400" @click="saveCustom">Lưu</button>
            <button class="rounded-md bg-white/10 px-3 py-1 text-xs text-white hover:bg-white/20" @click="resetDefault">Về mặc định (T7 21:00)</button>
            <button class="rounded-md px-3 py-1 text-xs text-brand-200 hover:text-white" @click="editing = false">Đóng</button>
          </div>
        </div>
      </div>
    </div>

    <div v-if="pending" class="py-12 text-center text-slate-400">Đang tải…</div>
    <div v-else-if="!items?.length" class="card p-12 text-center text-slate-400">
      Chưa có xe nào sắp mở bán.
    </div>
    <div v-else class="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
      <VehicleCard v-for="v in items" :key="v.id" :item="v" />
    </div>
  </section>
</template>
