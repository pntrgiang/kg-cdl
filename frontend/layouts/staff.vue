<script setup lang="ts">
const auth = useAuthStore()

const links = computed(() => {
  const base = [
    { label: 'Tổng quan', to: '/staff', icon: '📊' },
    { label: 'Bán xe', to: '/staff/sell', icon: '💰' },
    { label: 'Đặt lịch', to: '/staff/bookings', icon: '📅' },
    { label: 'Nhập kho', to: '/staff/inventory', icon: '📦' },
    { label: 'Khách hàng', to: '/staff/customers', icon: '👥' },
    { label: 'Doanh thu', to: '/staff/revenue', icon: '📈' },
    { label: 'Nhật ký', to: '/staff/logs', icon: '📜' },
  ]
  if (auth.isManager) base.push({ label: 'Sự kiện', to: '/staff/events', icon: '🎉' })
  if (auth.isManager) base.push({ label: 'Voucher', to: '/staff/vouchers', icon: '🎟️' })
  if (auth.isManager) base.push({ label: 'Hình ảnh', to: '/staff/media', icon: '🖼️' })
  if (auth.isDev) base.push({ label: 'Nhân viên', to: '/staff/users', icon: '⚙️' })
  base.push({ label: 'Tài khoản', to: '/staff/account', icon: '🔐' })
  return base
})

async function doLogout() {
  await auth.logout()
  navigateTo('/')
}
const roleLabel = computed(() => ({ dev: 'Dev', manager: 'Quản lý', staff: 'Nhân viên' }[auth.role] || ''))

// menu điều hướng trên điện thoại/tablet nhỏ
const navOpen = ref(false)
const route = useRoute()
watch(() => route.fullPath, () => { navOpen.value = false })
</script>

<template>
  <div class="flex min-h-screen flex-col bg-slate-50">
    <!-- nền mờ -->
    <div
      class="fixed inset-0 z-40 bg-black/50 transition-opacity duration-300 md:hidden"
      :class="navOpen ? 'opacity-100' : 'pointer-events-none opacity-0'"
      @click="navOpen = false"
    />
    <!-- ngăn kéo trượt từ trái -->
    <aside
      class="fixed inset-y-0 left-0 z-50 flex w-64 max-w-[80vw] flex-col bg-gradient-to-b from-brand-900 to-brand-800 text-white shadow-2xl transition-transform duration-300 ease-out md:hidden"
      :class="navOpen ? 'translate-x-0' : '-translate-x-full'"
    >
      <div class="flex shrink-0 items-center justify-between border-b border-white/10 px-4 py-3.5">
        <span class="flex items-center gap-2">
          <span class="font-serif font-bold tracking-wide text-gold-400">KANJI GROUP</span>
          <span class="badge bg-white/15 text-white">{{ roleLabel }}</span>
        </span>
        <button class="rounded-lg p-1 text-white/80 hover:bg-white/10 hover:text-white" aria-label="Đóng menu" @click="navOpen = false">✕</button>
      </div>
      <nav class="min-h-0 flex-1 overflow-auto p-2">
        <NuxtLink
          v-for="l in links" :key="l.to" :to="l.to"
          class="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-brand-100 transition-colors hover:bg-white/10"
          active-class="bg-white/15 text-white"
        ><span>{{ l.icon }}</span>{{ l.label }}</NuxtLink>
      </nav>
      <!-- hành động tài khoản -->
      <div class="shrink-0 border-t border-white/10 p-2">
        <div class="px-3 py-1 text-xs text-brand-200">{{ auth.displayName }}</div>
        <NuxtLink to="/" class="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-semibold text-gold-400 hover:bg-white/10">🏠 Trang khách</NuxtLink>
        <button class="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left text-sm font-medium text-brand-100 hover:bg-white/10" @click="doLogout">🚪 Đăng xuất</button>
      </div>
    </aside>

    <header class="bg-gradient-to-r from-brand-900 to-brand-800 text-white">
      <div class="flex items-center justify-between gap-2 px-3 py-3 sm:px-4">
        <div class="flex min-w-0 items-center gap-2">
          <button
            class="rounded-lg p-1.5 text-xl text-white/90 hover:bg-white/10 md:hidden"
            aria-label="Mở menu" @click="navOpen = !navOpen"
          >☰</button>
          <NuxtLink to="/staff" class="flex min-w-0 items-center gap-2">
            <img src="/logo.png" alt="Kanji Group" class="h-8 w-8 shrink-0 object-contain sm:h-9 sm:w-9" />
            <span class="truncate font-serif font-bold tracking-wide text-gold-400">KANJI GROUP</span>
            <span class="badge hidden bg-white/15 text-white sm:inline-flex">{{ roleLabel }}</span>
          </NuxtLink>
        </div>
        <div class="hidden shrink-0 items-center gap-2 text-sm md:flex">
          <NuxtLink
            to="/"
            class="inline-flex items-center gap-1.5 rounded-lg border border-gold-400/60 bg-white/10 px-2.5 py-1.5 text-xs font-semibold text-gold-400 transition hover:bg-gold-400 hover:text-brand-950"
          >
            🏠 <span class="hidden sm:inline">Trang khách</span>
          </NuxtLink>
          <span class="hidden rounded-full bg-white/10 px-3 py-1.5 text-white lg:inline-block">
            <strong>{{ auth.displayName }}</strong>
          </span>
          <button class="btn-gold !py-1.5 text-xs" @click="doLogout">Đăng xuất</button>
        </div>
      </div>
    </header>

    <div class="mx-auto flex w-full max-w-7xl flex-1 gap-6 px-4 py-6">
      <aside class="hidden w-56 shrink-0 md:block">
        <nav class="card sticky top-6 p-2">
          <NuxtLink
            v-for="l in links"
            :key="l.to"
            :to="l.to"
            class="flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-slate-600 transition-colors duration-200 hover:bg-brand-50"
            active-class="bg-brand-100 text-brand-900"
          >
            <span>{{ l.icon }}</span>{{ l.label }}
          </NuxtLink>
        </nav>
      </aside>
      <main class="min-w-0 flex-1">
        <slot />
      </main>
    </div>

    <footer class="border-t bg-white py-6 text-center text-xs text-slate-400">
      © <a href="https://discord.com/channels/1513475540042911764" target="_blank" rel="noopener noreferrer" class="transition hover:text-brand-600 hover:underline">Kanji Group — Car Dealer</a>
    </footer>
  </div>
</template>
