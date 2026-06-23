<script setup lang="ts">
definePageMeta({ layout: 'staff', middleware: 'manager' })
const api = useApi()
const auth = useAuthStore()

const msg = ref(''); const okMsg = ref('')

// ── POPUP ──
const { data: cfg, refresh: refreshCfg } = await useAsyncData('media-cfg', () =>
  api.get<{ modal_image: string; modal_target: string }>('/api/release-info'),
)
const popup = reactive({ image: '', target: '/upcoming' })
watchEffect(() => {
  popup.image = cfg.value?.modal_image || ''
  popup.target = cfg.value?.modal_target || '/upcoming'
})
const popupPreviewErr = ref(false)
watch(() => popup.image, () => { popupPreviewErr.value = false })

async function savePopup() {
  msg.value = ''; okMsg.value = ''
  try {
    await api.put('/api/release-modal', { image: popup.image.trim(), target: popup.target })
    okMsg.value = popup.image.trim() ? 'Đã lưu popup.' : 'Đã tắt popup (xoá ảnh).'
    await refreshCfg()
  } catch (e: any) { msg.value = e?.data?.error || 'Lưu popup thất bại.' }
}
const popupUploading = ref(false)
async function uploadPopup(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0]
  if (!f) return
  msg.value = ''; okMsg.value = ''; popupUploading.value = true
  try {
    const fd = new FormData(); fd.append('file', f)
    await api.post('/api/release-popup-upload', fd)
    okMsg.value = 'Đã tải ảnh popup lên (ghi đè /release-popup.jpg).'
    await refreshCfg()
  } catch (e: any) { msg.value = e?.data?.error || 'Tải ảnh thất bại.' }
  finally { popupUploading.value = false; (e.target as HTMLInputElement).value = '' }
}

// ── BANNER ──
const { data: banners, refresh: refreshBanners } = await useAsyncData('banners-all', () =>
  api.get<any[]>('/api/banners'),
)
const bannerUploading = ref(false)
async function uploadBanner(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0]
  if (!f) return
  msg.value = ''; okMsg.value = ''; bannerUploading.value = true
  try {
    const fd = new FormData(); fd.append('file', f)
    await api.post('/api/banners', fd)
    okMsg.value = 'Đã tải banner lên.'
    await refreshBanners()
  } catch (e: any) { msg.value = e?.data?.error || 'Tải banner thất bại.' }
  finally { bannerUploading.value = false; (e.target as HTMLInputElement).value = '' }
}
async function toggleBanner(b: any) {
  msg.value = ''
  try { await api.patch(`/api/banners/${b.id}`, { active: !b.is_active }); await refreshBanners() }
  catch (e: any) { msg.value = e?.data?.error || 'Đổi trạng thái thất bại.' }
}
async function removeBanner(b: any) {
  msg.value = ''; okMsg.value = ''
  if (!confirm('Xoá banner này? (không khôi phục được)')) return
  try { await api.del(`/api/banners/${b.id}`); okMsg.value = 'Đã xoá banner.'; await refreshBanners() }
  catch (e: any) { msg.value = e?.data?.error || 'Xoá banner thất bại.' }
}
const activeCount = computed(() => (banners.value || []).filter((b: any) => b.is_active).length)
</script>

<template>
  <div>
    <h1 class="mb-1 font-serif text-2xl font-bold text-brand-900">🖼️ Hình ảnh</h1>
    <p class="mb-4 text-sm text-slate-500">Quản lý popup thông báo và banner trang chủ của giao diện khách.</p>

    <div v-if="okMsg" class="mb-4 rounded-lg bg-green-50 px-4 py-3 text-sm text-green-700">{{ okMsg }}</div>
    <div v-if="msg" class="mb-4 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700">{{ msg }}</div>

    <!-- ───── POPUP ───── -->
    <h2 class="mb-2 font-semibold text-brand-900">Popup thông báo mở bán</h2>
    <div class="card mb-8 grid gap-6 p-4 lg:grid-cols-2">
      <div class="space-y-4">
        <div>
          <label class="label">Tải ảnh từ máy tính (ghi đè, luôn lưu tên cố định)</label>
          <input type="file" accept="image/*" class="block w-full text-sm" :disabled="popupUploading" @change="uploadPopup" />
          <p class="mt-1 text-xs text-slate-400">{{ popupUploading ? 'Đang tải lên…' : 'Chọn ảnh để ghi đè ảnh popup hiện tại.' }}</p>
        </div>
        <div>
          <label class="label">Hoặc dán URL ảnh</label>
          <input v-model="popup.image" type="url" placeholder="https://…/anh.jpg" class="input" />
          <p class="mt-1 text-xs text-amber-600">⚠️ Link ngoài (vd Discord CDN) có thể hết hạn; nên dùng "Tải ảnh từ máy" cho ổn định.</p>
        </div>
        <div>
          <label class="label">Khi khách bấm vào ảnh → chuyển đến</label>
          <div class="flex flex-wrap gap-4 text-sm">
            <label class="flex items-center gap-1.5"><input v-model="popup.target" type="radio" value="/upcoming" /> Xe sắp mở bán</label>
            <label class="flex items-center gap-1.5"><input v-model="popup.target" type="radio" value="/events" /> Sự kiện</label>
          </div>
        </div>
        <div class="flex flex-wrap gap-2">
          <button class="btn-primary" @click="savePopup">Lưu</button>
          <button v-if="popup.image" class="btn-ghost" @click="popup.image = ''; savePopup()">Tắt popup</button>
        </div>
      </div>
      <div>
        <label class="label">Xem trước</label>
        <div v-if="popup.image" class="overflow-hidden rounded-xl border bg-slate-100">
          <img v-if="!popupPreviewErr" :src="popup.image" class="block max-h-72 w-full object-contain" @error="popupPreviewErr = true" />
          <div v-else class="flex h-40 items-center justify-center px-4 text-center text-sm text-slate-400">Không tải được ảnh — kiểm tra URL.</div>
        </div>
        <div v-else class="flex h-40 items-center justify-center rounded-xl border border-dashed text-center text-sm text-slate-400">Chưa có ảnh → popup đang tắt.</div>
      </div>
    </div>

    <!-- ───── BANNER ───── -->
    <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
      <h2 class="font-semibold text-brand-900">Banner trang chủ <span class="text-sm font-normal text-slate-400">({{ activeCount }} đang bật)</span></h2>
      <label class="btn-primary cursor-pointer !py-1.5 text-sm">
        {{ bannerUploading ? 'Đang tải…' : '+ Tải banner lên' }}
        <input type="file" accept="image/*" class="hidden" :disabled="bannerUploading" @change="uploadBanner" />
      </label>
    </div>
    <p class="mb-3 text-xs text-slate-500">Banner đang bật sẽ tự động slide ở đầu trang "Xe đang mở bán". Tỉ lệ hiển thị 16:5 (ảnh ngang rộng đẹp nhất).</p>

    <div v-if="!banners?.length" class="card p-8 text-center text-sm text-slate-400">Chưa có banner nào. Tải lên banner đầu tiên.</div>
    <div v-else class="grid gap-4 sm:grid-cols-2">
      <div v-for="b in banners" :key="b.id" class="card overflow-hidden" :class="b.is_active ? '' : 'opacity-60'">
        <div class="aspect-[16/5] w-full bg-slate-100">
          <img :src="b.image_url" class="h-full w-full object-cover" />
        </div>
        <div class="flex items-center justify-between gap-2 p-3">
          <label class="flex cursor-pointer items-center gap-2 text-sm">
            <input type="checkbox" :checked="b.is_active" @change="toggleBanner(b)" />
            <span :class="b.is_active ? 'font-medium text-green-700' : 'text-slate-500'">{{ b.is_active ? 'Đang dùng (slide)' : 'Đang tắt' }}</span>
          </label>
          <button v-if="auth.isDev" class="text-xs font-medium text-red-600 hover:underline" @click="removeBanner(b)">Xoá</button>
          <span v-else class="text-[11px] text-slate-300">Chỉ dev được xoá</span>
        </div>
      </div>
    </div>
  </div>
</template>
