<script setup lang="ts">
definePageMeta({ middleware: 'customer-auth' })
const route = useRoute()
const api = useApi()
const auth = useAuthStore()

const { data: event, error, refresh } = await useAsyncData(`event-${route.params.id}`, () =>
  api.get<any>(`/api/events/${route.params.id}`),
)
const fmtDate = (s: string) => formatDate(s)

const statusLabel: Record<string, { t: string; c: string }> = {
  open: { t: 'Đang nhận đăng ký', c: 'bg-green-100 text-green-700' },
  drawn: { t: 'Đã quay số', c: 'bg-amber-100 text-amber-700' },
  published: { t: 'Đã có kết quả', c: 'bg-brand-100 text-brand-700' },
}
// khách có nằm trong danh sách trúng không
const iWon = computed(() => {
  if (!auth.isCustomer || !event.value?.winners) return false
  return event.value.winners.some((w: any) => w.customer_id === auth.customer?.id)
})

// ── đăng ký tham gia (khách phải tự bấm) ──
const registered = ref(false)
const regMsg = ref('')
const regLoading = ref(false)
const deadlinePassed = computed(() => {
  const d = event.value?.register_deadline
  return d ? new Date(d).getTime() < Date.now() : false
})
const canRegister = computed(() =>
  auth.isCustomer && event.value?.draw_status === 'open' && !deadlinePassed.value && !registered.value,
)

async function loadRegistration() {
  if (!auth.isCustomer || !event.value) return
  try {
    const r = await api.get<any>(`/api/events/${event.value.id}/registration`)
    registered.value = !!r.registered
  } catch {}
}
async function register() {
  regMsg.value = ''
  regLoading.value = true
  try {
    await api.post(`/api/events/${event.value.id}/register`)
    registered.value = true
    regMsg.value = 'Đăng ký tham gia thành công! Chờ kết quả quay số.'
    await refresh() // cập nhật lại số người đã đăng ký

  } catch (e: any) {
    regMsg.value = e?.data?.error || 'Đăng ký thất bại.'
  } finally {
    regLoading.value = false
  }
}
onMounted(() => { auth.hydrate(); loadRegistration() })
</script>

<template>
  <section>
    <NuxtLink to="/events" class="mb-4 inline-block text-sm text-brand-600 hover:underline">← Tất cả sự kiện</NuxtLink>
    <div v-if="error" class="card p-12 text-center text-slate-400">Không tìm thấy sự kiện.</div>
    <div v-else-if="event" class="mx-auto max-w-2xl">
      <div class="card p-6">
        <span v-if="event.draw_status" class="badge" :class="statusLabel[event.draw_status]?.c">
          {{ statusLabel[event.draw_status]?.t }}
        </span>
        <h1 class="mt-2 font-serif text-2xl font-bold text-brand-900">{{ event.title }}</h1>
        <p class="mt-2 whitespace-pre-line text-slate-600">{{ event.description }}</p>

        <div class="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-4">
          <div class="rounded-lg bg-gold-500/15 p-3 text-center">
            <div class="text-xs text-slate-500">Phần thưởng</div>
            <div class="font-semibold text-brand-900">🎁 {{ event.prize_name }}</div>
          </div>
          <div class="rounded-lg bg-brand-50 p-3 text-center">
            <div class="text-xs text-slate-500">Số người trúng</div>
            <div class="font-semibold text-brand-900">{{ event.winners_count }}</div>
          </div>
          <div class="rounded-lg bg-green-50 p-3 text-center">
            <div class="text-xs text-slate-500">Đã đăng ký</div>
            <div class="font-semibold text-green-700">{{ event.eligible_count }} người</div>
          </div>
          <div class="rounded-lg bg-brand-50 p-3 text-center">
            <div class="text-xs text-slate-500">Hạn đăng ký</div>
            <div class="font-semibold text-brand-900">{{ fmtDate(event.register_deadline) }}</div>
          </div>
        </div>

        <!-- đăng ký tham gia -->
        <div class="mt-4 rounded-lg border border-dashed p-4 text-sm">
          <div class="mb-2 text-slate-600">
            Khách hàng cần <strong>bấm đăng ký</strong> trước hết ngày {{ fmtDate(event.register_deadline) }} để vào danh sách quay số.
          </div>

          <template v-if="auth.isCustomer">
            <div v-if="registered" class="rounded-lg bg-green-50 px-4 py-2 font-medium text-green-700">
              ✓ Bạn đã đăng ký tham gia sự kiện này.
            </div>
            <button
              v-else-if="canRegister"
              class="btn-gold w-full sm:w-auto"
              :disabled="regLoading"
              @click="register"
            >{{ regLoading ? 'Đang xử lý…' : '🎯 Đăng ký tham gia' }}</button>
            <div v-else-if="event.draw_status !== 'open'" class="text-slate-500">Sự kiện đã đóng đăng ký (đã quay số).</div>
            <div v-else-if="deadlinePassed" class="text-red-600">Đã hết hạn đăng ký tham gia.</div>
            <p v-if="regMsg" class="mt-2 text-sm" :class="registered ? 'text-green-600' : 'text-red-600'">{{ regMsg }}</p>
          </template>
          <template v-else>
            <div class="text-slate-600">
              Bạn cần đăng nhập bằng tài khoản khách hàng để đăng ký tham gia.
              <NuxtLink to="/customer/register" class="text-brand-600 hover:underline">Tạo tài khoản</NuxtLink>
            </div>
          </template>
        </div>

        <!-- kết quả -->
        <div v-if="event.draw_status === 'published' && event.winners?.length" class="mt-5">
          <h2 class="mb-2 font-semibold text-brand-900">🏆 Kết quả trúng thưởng</h2>
          <div v-if="iWon" class="mb-3 rounded-lg bg-green-50 px-4 py-3 font-medium text-green-700">
            🎉 Chúc mừng! Bạn đã trúng thưởng. Liên hệ nhân viên Kanji Group để nhận thưởng.
          </div>
          <div class="space-y-1">
            <div v-for="w in event.winners" :key="w.id" class="rounded-lg bg-gold-500/15 px-3 py-2 font-medium text-brand-900">
              🏆 {{ w.customer_name }}
            </div>
          </div>
        </div>
        <p v-else-if="event.draw_status !== 'published'" class="mt-5 text-sm text-slate-400">
          Kết quả sẽ được công bố sau khi quản lý quay số và xác nhận.
        </p>
      </div>
    </div>
  </section>
</template>
