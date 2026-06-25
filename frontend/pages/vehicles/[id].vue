<script setup lang="ts">
const route = useRoute()
const api = useApi()
const auth = useAuthStore()
const { data: v, error } = await useAsyncData(`vehicle-${route.params.id}`, () =>
  api.get<any>(`/api/vehicles/${route.params.id}`),
)
const img = computed(() => v.value?.image_url || '')

// ── SEO: meta động + dữ liệu có cấu trúc (JSON-LD Product) cho từng xe ──
const site = (useRuntimeConfig().public.siteUrl as string) || 'https://kg-cdl.ddns.net'
const absImg = computed(() => {
  const u = v.value?.image_url || ''
  if (!u) return `${site}/logo.png`
  return u.startsWith('http') ? u : `${site}${u.startsWith('/') ? '' : '/'}${u}`
})
if (v.value) {
  const d = v.value
  const price = d.final_price ?? d.base_price ?? 0
  useSeo({
    title: d.name,
    description:
      (d.description && String(d.description).replace(/\s+/g, ' ').trim().slice(0, 160)) ||
      `${d.name}${d.brand ? ' (' + d.brand + ')' : ''} — ${formatMoney(price)} tại đại lý Kanji Group, Lux City. Xem thông số, ưu đãi và đặt lịch xem xe.`,
    image: d.image_url,
  })
  useHead({
    script: [
      {
        type: 'application/ld+json',
        innerHTML: JSON.stringify({
          '@context': 'https://schema.org',
          '@type': 'Product',
          name: d.name,
          image: absImg.value,
          description:
            (d.description && String(d.description).replace(/\s+/g, ' ').trim().slice(0, 300)) ||
            `Xe ${d.name} tại đại lý Kanji Group, Lux City.`,
          brand: { '@type': 'Brand', name: d.brand || 'Kanji Group' },
          category: d.class || undefined,
          offers: {
            '@type': 'Offer',
            priceCurrency: 'USD',
            price: Math.round(price),
            availability:
              d.status === 'sold_out' || (typeof d.quantity === 'number' && d.quantity <= 0)
                ? 'https://schema.org/OutOfStock'
                : 'https://schema.org/InStock',
            url: `${site}/vehicles/${route.params.id}`,
          },
        }),
      },
    ],
  })
} else {
  useSeo({ title: 'Không tìm thấy xe', noindex: true })
}

// xe tương tự + lịch sử khuyến mãi
const { data: similar } = await useAsyncData(`similar-${route.params.id}`, () =>
  api.get<any[]>(`/api/vehicles/${route.params.id}/similar`),
)
const { data: discounts } = await useAsyncData(`discounts-${route.params.id}`, () =>
  api.get<any[]>(`/api/vehicles/${route.params.id}/discounts`),
)
const fmtDate = (s: string) => formatDate(s)
function promoStatus(d: any): { label: string; cls: string } {
  const now = Date.now()
  const start = new Date(d.starts_at).getTime()
  const end = d.ends_at ? new Date(d.ends_at).getTime() : null
  if (!d.is_active) return { label: 'Đã kết thúc', cls: 'bg-slate-200 text-slate-500' }
  if (start > now) return { label: 'Sắp diễn ra', cls: 'bg-brand-100 text-brand-700' }
  if (end && end < now) return { label: 'Đã hết hạn', cls: 'bg-slate-200 text-slate-500' }
  return { label: 'Đang áp dụng', cls: 'bg-green-100 text-green-700' }
}

// thanh thông số hiệu năng (0-100)
const specs = computed(() => {
  const d = v.value || {}
  return [
    { label: 'Tốc độ tối đa', value: d.rate_speed || 0, icon: '⚡' },
    { label: 'Tăng tốc', value: d.rate_accel || 0, icon: '🚀' },
    { label: 'Phanh', value: d.rate_braking || 0, icon: '🛑' },
    { label: 'Độ bám đường', value: d.rate_traction || 0, icon: '🛞' },
  ]
})

// link tham khảo trên GTA Wiki (ghi nguồn) — dùng tìm kiếm để tránh link hỏng
const wikiUrl = computed(
  () => `https://gta.fandom.com/wiki/Special:Search?query=${encodeURIComponent(v.value?.name || '')}`,
)

// quản lý sửa nội dung giới thiệu + thông số
const editing = ref(false)
const draft = ref('')
const draftSpec = reactive({ seats: 0, trunk_kg: 0, rate_speed: 0, rate_accel: 0, rate_braking: 0, rate_traction: 0 })
const saving = ref(false)
const saveErr = ref('')
function startEdit() {
  const d = v.value || {}
  draft.value = d.description || ''
  draftSpec.seats = d.seats ?? 0
  draftSpec.trunk_kg = d.trunk_kg || 10
  draftSpec.rate_speed = d.rate_speed || 0
  draftSpec.rate_accel = d.rate_accel || 0
  draftSpec.rate_braking = d.rate_braking || 0
  draftSpec.rate_traction = d.rate_traction || 0
  editing.value = true
  saveErr.value = ''
}
async function saveDesc() {
  saving.value = true
  saveErr.value = ''
  try {
    const updated = await api.patch<any>(`/api/catalog/${v.value.catalog_id}`, {
      description: draft.value,
      seats: draftSpec.seats || null,
      trunk_kg: draftSpec.trunk_kg,
      rate_speed: draftSpec.rate_speed,
      rate_accel: draftSpec.rate_accel,
      rate_braking: draftSpec.rate_braking,
      rate_traction: draftSpec.rate_traction,
    })
    Object.assign(v.value, {
      description: updated.description,
      seats: updated.seats,
      trunk_kg: updated.trunk_kg,
      rate_speed: updated.rate_speed,
      rate_accel: updated.rate_accel,
      rate_braking: updated.rate_braking,
      rate_traction: updated.rate_traction,
    })
    editing.value = false
  } catch (e: any) {
    saveErr.value = e?.data?.error || 'Lưu thất bại.'
  } finally {
    saving.value = false
  }
}

// ── đặt lịch xem/mua xe (khách hàng) ──
const showBooking = ref(false)
const bookingDate = ref('')
const bookingNote = ref('')
const bookingMsg = ref('')
const bookingOk = ref('')
const bookingLoading = ref(false)
const todayStr = new Date().toISOString().slice(0, 10)
function openBooking() {
  if (!auth.isCustomer) {
    return navigateTo(`/customer/login?redirect=${encodeURIComponent(route.fullPath)}`)
  }
  bookingDate.value = ''; bookingNote.value = ''; bookingMsg.value = ''; bookingOk.value = ''
  showBooking.value = true
}
async function submitBooking() {
  bookingMsg.value = ''
  if (!bookingDate.value) { bookingMsg.value = 'Hãy chọn ngày muốn đến xem xe.'; return }
  bookingLoading.value = true
  try {
    await api.post('/api/bookings', { inventory_id: v.value.id, visit_date: bookingDate.value, note: bookingNote.value })
    bookingOk.value = 'Đã gửi yêu cầu đặt lịch! Bạn có thể theo dõi trạng thái ở trang “Tài khoản của tôi”.'
    showBooking.value = false
  } catch (e: any) { bookingMsg.value = e?.data?.error || 'Đặt lịch thất bại.' }
  finally { bookingLoading.value = false }
}
</script>


<template>
  <section>
    <NuxtLink to="/" class="mb-4 inline-block text-sm text-brand-600 hover:underline">← Về showroom</NuxtLink>
    <div v-if="error" class="card p-12 text-center text-slate-400">Không tìm thấy xe.</div>
    <div v-else-if="v">
    <div class="grid items-start gap-6 md:grid-cols-2">
      <div class="card overflow-hidden">
        <div class="relative">
          <VehicleImageViewer :src="img" :alt="v.name" />
          <span v-if="v.discount_percent > 0" class="badge absolute left-3 top-3 z-10 bg-gold-500 text-brand-950">
            -{{ Math.round(v.discount_percent) }}%
          </span>
        </div>
      </div>

      <div>
        <div class="text-sm uppercase tracking-wide text-brand-500">{{ v.brand || '—' }} · {{ v.class }}</div>
        <h1 class="font-serif text-3xl font-bold text-brand-900">{{ v.name }}</h1>

        <div class="mt-4 flex items-end gap-3">
          <span v-if="v.discount_percent > 0" class="text-lg text-slate-400 line-through">
            {{ formatMoney(v.base_price) }}
          </span>
          <span class="text-3xl font-bold text-brand-800">{{ formatMoney(v.final_price) }}</span>
        </div>

        <div class="mt-4 flex flex-wrap gap-2 text-sm">
          <span class="badge bg-slate-200 text-slate-600">Trạng thái: {{ vehicleStatusLabel(v.status) }}</span>
          <span v-if="v.on_sale_at" class="badge bg-gold-500/20 text-gold-600">
            Mở bán: {{ fmtDate(v.on_sale_at) }}
          </span>
        </div>

        <!-- đặt lịch xem/mua xe -->
        <div v-if="v.booking_open" class="mt-4">
          <ClientOnly>
            <div v-if="auth.isUser" class="rounded-lg border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-700">
              Bạn đang đăng nhập với tư cách <strong>nhân viên Car Dealer</strong> — không thể sử dụng tính năng đặt lịch dành cho khách hàng.
            </div>
            <button v-else class="btn-gold w-full sm:w-auto" @click="openBooking">📅 Đặt lịch đến xem xe</button>
            <template #fallback>
              <button class="btn-gold w-full sm:w-auto" @click="openBooking">📅 Đặt lịch đến xem xe</button>
            </template>
          </ClientOnly>
          <p v-if="bookingOk" class="mt-2 rounded-lg bg-green-50 px-3 py-2 text-sm text-green-700">{{ bookingOk }}</p>
        </div>

        <div class="card mt-6 p-4">
          <div class="mb-2 flex items-center justify-between">
            <h2 class="font-semibold text-brand-900">Giới thiệu</h2>
            <button
              v-if="auth.isManager && !editing"
              class="text-xs text-brand-600 hover:underline"
              @click="startEdit"
            >✏️ Sửa</button>
          </div>

          <template v-if="editing">
            <textarea v-model="draft" rows="5" class="input"></textarea>
            <div class="mt-3 text-xs font-medium text-slate-500">Thông số (số chỗ + cốp xe + điểm 0–100)</div>
            <div class="mt-1 grid grid-cols-2 gap-2 sm:grid-cols-6">
              <input v-model.number="draftSpec.seats" type="number" class="input" placeholder="Số chỗ" title="Số chỗ" />
              <input v-model.number="draftSpec.trunk_kg" type="number" min="1" class="input" placeholder="Cốp (kg)" title="Cốp xe (kg)" />
              <input v-model.number="draftSpec.rate_speed" type="number" class="input" placeholder="Tốc độ" title="Tốc độ 0-100" />
              <input v-model.number="draftSpec.rate_accel" type="number" class="input" placeholder="Tăng tốc" title="Tăng tốc 0-100" />
              <input v-model.number="draftSpec.rate_braking" type="number" class="input" placeholder="Phanh" title="Phanh 0-100" />
              <input v-model.number="draftSpec.rate_traction" type="number" class="input" placeholder="Độ bám" title="Độ bám 0-100" />
            </div>
            <p v-if="saveErr" class="mt-2 text-sm text-red-600">{{ saveErr }}</p>
            <div class="mt-2 flex justify-end gap-2">
              <button class="btn-ghost !py-1.5 text-sm" @click="editing = false">Hủy</button>
              <button class="btn-primary !py-1.5 text-sm" :disabled="saving" @click="saveDesc">
                {{ saving ? 'Đang lưu…' : 'Lưu' }}
              </button>
            </div>
          </template>

          <template v-else>
            <p class="whitespace-pre-line text-sm text-slate-600">
              {{ v.description || 'Mẫu xe ' + v.name + ' thuộc dòng ' + v.class + '. Liên hệ nhân viên Kanji Group để được tư vấn chi tiết.' }}
            </p>
            <a :href="wikiUrl" target="_blank" rel="noopener" class="mt-3 inline-block text-xs text-brand-500 hover:underline">
              Xem thêm trên GTA Wiki ↗
            </a>
          </template>
        </div>
      </div>
    </div>

    <!-- Thông số kỹ thuật -->
    <div class="card mt-6 p-5">
      <h2 class="mb-4 font-semibold text-brand-900">Thông số kỹ thuật</h2>

      <div class="mb-5 grid grid-cols-2 gap-3 sm:grid-cols-4">
        <div class="rounded-lg bg-brand-50 p-3 text-center">
          <div class="text-xs text-slate-500">Hãng</div>
          <div class="font-semibold text-brand-900">{{ v.brand || '—' }}</div>
        </div>
        <div class="rounded-lg bg-brand-50 p-3 text-center">
          <div class="text-xs text-slate-500">Dòng xe</div>
          <div class="font-semibold text-brand-900">{{ v.class || '—' }}</div>
        </div>
        <div class="rounded-lg bg-brand-50 p-3 text-center">
          <div class="text-xs text-slate-500">Số chỗ</div>
          <div class="font-semibold text-brand-900">{{ v.seats ?? '—' }}</div>
        </div>
        <div class="rounded-lg bg-brand-50 p-3 text-center">
          <div class="text-xs text-slate-500">Cốp xe</div>
          <div class="font-semibold text-brand-900">{{ v.trunk_kg ?? 10 }} kg</div>
        </div>
      </div>

      <div class="space-y-3">
        <div v-for="s in specs" :key="s.label">
          <div class="mb-1 flex justify-between text-sm">
            <span class="text-slate-600">{{ s.icon }} {{ s.label }}</span>
            <span class="font-medium text-brand-800">{{ s.value }}/100</span>
          </div>
          <div class="h-2.5 w-full overflow-hidden rounded-full bg-slate-100">
            <div
              class="h-full rounded-full bg-gradient-to-r from-brand-600 to-gold-500 transition-all"
              :style="{ width: s.value + '%' }"
            />
          </div>
        </div>
      </div>
      <p class="mt-3 text-xs text-slate-400">Điểm hiệu năng chuẩn hóa (0–100) từ dữ liệu xe gốc trong game.</p>
      <p class="mt-1 text-xs font-medium text-red-600">
        Lưu ý: các thông số trên chỉ mang tính tham khảo. Hiệu năng thực tế còn tuỳ thuộc vào nhiều yếu tố khác tại Lux City.
      </p>
    </div>

    <!-- Lịch sử giá & khuyến mãi -->
    <div class="card mt-6 p-5">
      <h2 class="mb-3 font-semibold text-brand-900">Lịch sử giá &amp; khuyến mãi</h2>
      <div class="mb-3 flex flex-wrap gap-2 text-sm">
        <span class="badge bg-slate-100 text-slate-600">Giá gốc: {{ formatMoney(v.base_price) }}</span>
        <span v-if="v.discount_percent > 0" class="badge bg-gold-500 text-brand-950">
          Đang giảm {{ Math.round(v.discount_percent) }}% → {{ formatMoney(v.final_price) }}
        </span>
      </div>
      <div v-if="discounts && discounts.length" class="overflow-x-auto rounded-lg border">
        <table class="w-full min-w-[480px] text-sm">
          <thead class="bg-slate-50 text-left text-xs uppercase text-slate-500">
            <tr><th class="p-2.5">Mức giảm</th><th class="p-2.5">Giá sau giảm</th><th class="p-2.5">Thời gian</th><th class="p-2.5">Trạng thái</th></tr>
          </thead>
          <tbody>
            <tr v-for="d in discounts" :key="d.id" class="border-t">
              <td class="p-2.5 font-medium text-gold-600">-{{ Math.round(d.percent) }}%</td>
              <td class="p-2.5">{{ formatMoney(v.base_price * (1 - d.percent / 100)) }}</td>
              <td class="p-2.5 text-slate-500">
                {{ fmtDate(d.starts_at) }}<span v-if="d.ends_at"> → {{ fmtDate(d.ends_at) }}</span>
              </td>
              <td class="p-2.5"><span class="badge" :class="promoStatus(d).cls">{{ promoStatus(d).label }}</span></td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-else class="text-sm text-slate-400">Chưa có chương trình khuyến mãi nào cho xe này.</p>
    </div>

    <!-- Xe tương tự -->
    <div v-if="similar && similar.length" class="mt-6">
      <h2 class="mb-3 font-semibold text-brand-900">Xe tương tự đang mở bán</h2>
      <div class="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
        <VehicleCard v-for="s in similar" :key="s.id" :item="s" />
      </div>
    </div>
    </div>

    <!-- modal đặt lịch -->
    <div v-if="showBooking" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" @click.self="showBooking = false">
      <div class="card w-full max-w-md p-5">
        <h2 class="mb-1 font-semibold text-brand-900">📅 Đặt lịch đến xem xe</h2>
        <p class="mb-4 text-sm text-slate-500">{{ v?.name }}</p>
        <div class="space-y-3">
          <div>
            <label class="label">Ngày muốn đến xem *</label>
            <input v-model="bookingDate" type="date" :min="todayStr" class="input" />
          </div>
          <div>
            <label class="label">Ghi chú <span class="font-normal text-slate-400">(tuỳ chọn)</span></label>
            <textarea v-model="bookingNote" class="input" rows="2" placeholder="VD: muốn xem buổi chiều, hỏi thêm về trả góp…"></textarea>
          </div>
          <p v-if="bookingMsg" class="text-sm text-red-600">{{ bookingMsg }}</p>
        </div>
        <div class="mt-4 flex justify-end gap-2">
          <button class="btn-ghost" @click="showBooking = false">Huỷ</button>
          <button class="btn-gold" :disabled="bookingLoading" @click="submitBooking">{{ bookingLoading ? 'Đang gửi…' : 'Gửi đặt lịch' }}</button>
        </div>
      </div>
    </div>
  </section>
</template>

