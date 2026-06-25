<script setup lang="ts">
const api = useApi()
const auth = useAuthStore()
onMounted(() => auth.hydrate())
const { data: events, pending } = await useAsyncData('events', () => api.get<any[]>('/api/events'))
useSeo({
  title: 'Sự kiện khuyến mãi',
  description:
    'Sự kiện khuyến mãi và quay số trúng thưởng tại Kanji Group — Lux City. Đăng ký tham gia để nhận voucher và phần thưởng hấp dẫn.',
})
const statusLabel: Record<string, { t: string; c: string }> = {
  open: { t: 'Đang nhận đăng ký', c: 'bg-green-100 text-green-700' },
  drawn: { t: 'Đang chờ kết quả', c: 'bg-amber-100 text-amber-700' },
  published: { t: 'Đã có kết quả', c: 'bg-brand-100 text-brand-700' },
}
const fmtDate = (s: string) => formatDate(s)

// Chỉ hiển thị sự kiện ĐANG DIỄN RA (chưa công bố kết quả: open/drawn).
const ongoing = computed(() => (events.value || []).filter((e: any) => e.draw_status && e.draw_status !== 'published'))
</script>

<template>
  <section>
    <h1 class="mb-4 text-xl font-bold">🎁 Sự kiện khuyến mãi</h1>
    <ClientOnly>
      <div v-if="auth.isUser" class="mb-4 rounded-lg border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-700">
        Bạn đang đăng nhập với tư cách <strong>nhân viên Car Dealer</strong> — không thể tham gia sự kiện dành cho khách hàng.
      </div>
      <div v-else-if="!auth.isAuthed" class="mb-4 rounded-lg border border-gold-400 bg-gold-500/10 px-4 py-3 text-sm text-brand-800">
        Bạn cần <NuxtLink to="/customer/login" class="font-semibold text-brand-700 underline">đăng nhập</NuxtLink>
        (hoặc <NuxtLink to="/customer/register" class="font-semibold text-brand-700 underline">đăng ký</NuxtLink>)
        để tham gia và xem chi tiết sự kiện.
      </div>
    </ClientOnly>

    <div v-if="pending" class="py-12 text-center text-slate-400">Đang tải…</div>
    <div v-else-if="!ongoing.length" class="card p-12 text-center text-slate-400">Chưa có sự kiện nào đang diễn ra.</div>
    <div v-else class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <NuxtLink v-for="e in ongoing" :key="e.id" :to="`/events/${e.id}`" class="card block p-5 transition hover:shadow-md">
        <span class="badge" :class="statusLabel[e.draw_status]?.c">{{ statusLabel[e.draw_status]?.t }}</span>
        <h3 class="mt-3 text-lg font-semibold text-brand-900">{{ e.title }}</h3>
        <p class="mt-1 line-clamp-2 text-sm text-slate-500">{{ e.description || 'Tham gia ngay!' }}</p>
        <div class="mt-3 flex flex-wrap gap-1.5 text-xs">
          <span class="badge bg-gold-500/20 text-brand-900">🎁 {{ e.prize_name }}</span>
          <span class="badge bg-slate-100 text-slate-600">Hạn: {{ fmtDate(e.register_deadline) }}</span>
        </div>
        <div class="mt-3 text-sm font-medium text-gold-600">Xem chi tiết →</div>
      </NuxtLink>
    </div>
  </section>
</template>
