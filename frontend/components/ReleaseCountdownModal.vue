<script setup lang="ts">
// Popup thông báo mở bán — hiện ảnh mỗi khi tải/mở trang mới ở giao diện khách.
// Bấm vào ảnh -> chuyển tới đích do quản lý đặt (xe sắp mở bán hoặc sự kiện).
const api = useApi()
const auth = useAuthStore()

const open = ref(false)
const imageUrl = ref('')
const target = ref('/upcoming')
const imgError = ref(false)

onMounted(async () => {
  if (auth.isUser) return // không hiện cho nhân viên/quản lý
  try {
    const r = await api.get<{ modal_image: string; modal_target: string }>('/api/release-info')
    imageUrl.value = r.modal_image || ''
    target.value = r.modal_target || '/upcoming'
  } catch { return }
  if (imageUrl.value) open.value = true // chỉ hiện khi đã đặt ảnh
})

function close() { open.value = false }
function goTarget() { close(); navigateTo(target.value) }
</script>

<template>
  <Teleport to="body">
    <Transition name="rmodal">
    <div v-if="open" class="fixed inset-0 z-[60] flex items-center justify-center p-4" @click.self="close">
      <div class="absolute inset-0 bg-black/60" @click="close" />
      <div class="rmodal-box relative w-full max-w-3xl overflow-hidden rounded-2xl bg-white shadow-2xl">
        <button
          class="absolute right-2 top-2 z-10 flex h-9 w-9 items-center justify-center rounded-full bg-black/50 text-base text-white/90 transition hover:bg-black/70 hover:text-white"
          aria-label="Đóng" @click.stop="close"
        >✕</button>

        <!-- ảnh: bấm vào để chuyển hướng -->
        <button v-if="!imgError" type="button" class="block w-full" @click="goTarget" title="Bấm để xem ngay">
          <img
            :src="imageUrl" alt="Thông báo mở bán xe"
            class="block max-h-[85vh] w-full cursor-pointer object-contain bg-slate-100"
            @error="imgError = true"
          />
        </button>
        <div v-else class="flex h-40 items-center justify-center bg-slate-100 px-4 text-center text-sm text-slate-400">
          Không tải được ảnh (có thể link đã hết hạn). Quản lý vui lòng cập nhật lại ảnh popup.
        </div>
      </div>
    </div>
    </Transition>
  </Teleport>
</template>

<style>
/* hiệu ứng nhẹ khi mở/đóng popup: nền mờ dần, hộp thu/phóng nhẹ */
.rmodal-enter-active,
.rmodal-leave-active {
  transition: opacity 0.22s ease;
}
.rmodal-enter-from,
.rmodal-leave-to {
  opacity: 0;
}
.rmodal-enter-active .rmodal-box,
.rmodal-leave-active .rmodal-box {
  transition: transform 0.22s ease;
}
.rmodal-enter-from .rmodal-box,
.rmodal-leave-to .rmodal-box {
  transform: scale(0.95);
}
@media (prefers-reduced-motion: reduce) {
  .rmodal-enter-active,
  .rmodal-leave-active,
  .rmodal-enter-active .rmodal-box,
  .rmodal-leave-active .rmodal-box {
    transition: none;
  }
  .rmodal-enter-from .rmodal-box,
  .rmodal-leave-to .rmodal-box {
    transform: none;
  }
}
</style>
