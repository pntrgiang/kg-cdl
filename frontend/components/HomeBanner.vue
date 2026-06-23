<script setup lang="ts">
// Banner trang chủ: slide các banner đang bật. Không có banner -> hiện hero mặc định.
const api = useApi()
const { data: banners } = await useAsyncData('home-banners', () =>
  api.get<any[]>('/api/banners/active'),
)
const list = computed(() => banners.value || [])

const idx = ref(0)
let timer: any = null
function go(i: number) { idx.value = (i + list.value.length) % list.value.length }
onMounted(() => {
  if (list.value.length > 1) {
    timer = setInterval(() => { idx.value = (idx.value + 1) % list.value.length }, 5000)
  }
})
onBeforeUnmount(() => { if (timer) clearInterval(timer) })
</script>

<template>
  <!-- có banner -> carousel -->
  <div v-if="list.length" class="mb-6 overflow-hidden rounded-2xl bg-brand-900 shadow">
    <div class="relative aspect-[16/5] w-full">
      <TransitionGroup name="bfade">
        <img
          v-for="(b, i) in list" v-show="i === idx" :key="b.id"
          :src="b.image_url" alt="Banner"
          class="absolute inset-0 h-full w-full object-cover"
        />
      </TransitionGroup>

      <!-- nút trái/phải + chấm (khi có >1) -->
      <template v-if="list.length > 1">
        <button class="absolute left-2 top-1/2 flex h-9 w-9 -translate-y-1/2 items-center justify-center rounded-full bg-black/40 text-white hover:bg-black/60" aria-label="Banner trước" @click="go(idx - 1)">
          <svg viewBox="0 0 24 24" class="h-5 w-5" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M15 18l-6-6 6-6" /></svg>
        </button>
        <button class="absolute right-2 top-1/2 flex h-9 w-9 -translate-y-1/2 items-center justify-center rounded-full bg-black/40 text-white hover:bg-black/60" aria-label="Banner sau" @click="go(idx + 1)">
          <svg viewBox="0 0 24 24" class="h-5 w-5" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M9 18l6-6-6-6" /></svg>
        </button>
        <div class="absolute bottom-2 left-1/2 flex -translate-x-1/2 gap-1.5">
          <button
            v-for="(b, i) in list" :key="b.id"
            class="h-2 rounded-full transition-all"
            :class="i === idx ? 'w-5 bg-gold-400' : 'w-2 bg-white/60 hover:bg-white'"
            @click="go(i)"
          />
        </div>
      </template>
    </div>
  </div>

  <!-- không có banner -> hero mặc định -->
  <div v-else class="mb-6 rounded-2xl bg-gradient-to-r from-brand-800 to-brand-900 p-6 text-white">
    <h1 class="font-serif text-2xl font-bold text-gold-400">Showroom Kanji Group</h1>
    <p class="mt-1 text-brand-100">Những chiếc xe đang mở bán — giá tốt nhất tại Lux City.</p>
  </div>
</template>

<style scoped>
.bfade-enter-active,
.bfade-leave-active {
  transition: opacity 0.6s ease;
}
.bfade-enter-from,
.bfade-leave-to {
  opacity: 0;
}
</style>
