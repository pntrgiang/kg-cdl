<script setup lang="ts">
const auth = useAuthStore()
const tabs = [
  { label: 'Xe đang mở bán', to: '/' },
  { label: 'Xe sắp mở bán', to: '/upcoming' },
  { label: 'Sự kiện', to: '/events' },
]
async function doLogout() {
  await auth.logout()
  navigateTo('/')
}
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
        <span class="font-serif font-bold tracking-wide text-gold-400">KANJI GROUP</span>
        <button class="rounded-lg p-1 text-white/80 hover:bg-white/10 hover:text-white" aria-label="Đóng menu" @click="navOpen = false">✕</button>
      </div>
      <nav class="min-h-0 flex-1 overflow-auto p-2">
        <NuxtLink
          v-for="t in tabs" :key="t.to" :to="t.to"
          class="block rounded-lg px-3 py-2.5 text-sm font-medium text-brand-100 transition-colors hover:bg-white/10"
          active-class="bg-white/15 text-white"
        >{{ t.label }}</NuxtLink>
      </nav>
      <!-- hành động tài khoản -->
      <div class="shrink-0 border-t border-white/10 p-2">
        <template v-if="auth.isUser">
          <NuxtLink to="/staff" class="block rounded-lg px-3 py-2.5 text-sm font-semibold text-gold-400 hover:bg-white/10">🛠️ Bảng điều khiển</NuxtLink>
          <button class="block w-full rounded-lg px-3 py-2.5 text-left text-sm font-medium text-brand-100 hover:bg-white/10" @click="doLogout">Đăng xuất</button>
        </template>
        <template v-else-if="auth.isCustomer">
          <div class="px-3 py-1 text-xs text-brand-200">Xin chào, <strong class="text-gold-400">{{ auth.displayName }}</strong></div>
          <NuxtLink to="/account" class="block rounded-lg px-3 py-2.5 text-sm font-semibold text-white hover:bg-white/10">👤 Tài khoản</NuxtLink>
          <button class="block w-full rounded-lg px-3 py-2.5 text-left text-sm font-medium text-brand-100 hover:bg-white/10" @click="doLogout">Đăng xuất</button>
        </template>
        <template v-else>
          <NuxtLink to="/customer/login" class="block rounded-lg px-3 py-2.5 text-sm font-medium text-brand-100 hover:bg-white/10">Đăng nhập khách hàng</NuxtLink>
          <NuxtLink to="/login" class="block rounded-lg px-3 py-2.5 text-sm font-semibold text-gold-400 hover:bg-white/10">Đăng nhập nhân viên</NuxtLink>
        </template>
      </div>
    </aside>

    <header class="bg-gradient-to-r from-brand-900 to-brand-800 text-white shadow-lg">
      <div class="mx-auto flex max-w-6xl items-center justify-between gap-2 px-3 py-3 sm:gap-4 sm:px-4">
        <div class="flex min-w-0 items-center gap-2">
          <button
            class="rounded-lg p-1.5 text-xl text-white/90 hover:bg-white/10 md:hidden"
            aria-label="Mở menu" @click="navOpen = !navOpen"
          >☰</button>
          <NuxtLink to="/" class="flex min-w-0 items-center gap-2 sm:gap-3">
            <img src="/logo.png" alt="Kanji Group" class="h-9 w-9 shrink-0 object-contain sm:h-10 sm:w-10" />
            <div class="min-w-0 leading-tight">
              <div class="truncate font-serif text-base font-bold tracking-wide text-gold-400 sm:text-lg">KANJI GROUP</div>
              <div class="text-[10px] uppercase tracking-[0.2em] text-brand-200 sm:text-[11px]">Car Dealer</div>
            </div>
          </NuxtLink>
        </div>

        <nav class="hidden gap-1 md:flex">
          <NuxtLink
            v-for="t in tabs"
            :key="t.to"
            :to="t.to"
            class="rounded-lg px-3 py-2 text-sm font-medium text-brand-100 transition-colors duration-200 hover:bg-white/10"
            active-class="bg-white/15 text-white"
          >
            {{ t.label }}
          </NuxtLink>
        </nav>

        <div class="hidden shrink-0 items-center gap-2 md:flex">
          <template v-if="auth.isUser">
            <NuxtLink to="/staff" class="btn-gold !py-1.5 text-xs">Bảng điều khiển</NuxtLink>
            <button class="hidden text-xs text-brand-200 hover:text-white sm:inline" @click="doLogout">Đăng xuất</button>
          </template>
          <template v-else-if="auth.isCustomer">
            <span class="hidden rounded-full bg-white/10 px-3 py-1.5 text-sm text-white lg:inline-block">
              Xin chào, <strong class="text-gold-400">{{ auth.displayName }}</strong>
            </span>
            <NuxtLink
              to="/account"
              class="inline-flex items-center gap-1.5 rounded-lg border border-white/40 bg-white/10 px-2.5 py-1.5 text-xs font-semibold text-white transition hover:bg-white/20 sm:px-3 sm:text-sm"
            >👤 <span class="hidden sm:inline">Tài khoản</span></NuxtLink>
            <button
              class="inline-flex items-center gap-1.5 rounded-lg bg-gold-500 px-2.5 py-1.5 text-xs font-semibold text-brand-950 transition hover:bg-gold-400 sm:px-3 sm:text-sm"
              @click="doLogout"
            >Đăng xuất</button>
          </template>
          <template v-else>
            <NuxtLink to="/customer/login" class="btn-ghost !border-white/30 !bg-transparent !py-1.5 text-xs !text-white hover:!bg-white/10">
              Khách hàng
            </NuxtLink>
            <NuxtLink to="/login" class="btn-gold !py-1.5 text-xs">Nhân viên</NuxtLink>
          </template>
        </div>
      </div>
    </header>

    <main class="mx-auto w-full max-w-6xl flex-1 px-4 py-6">
      <slot />
    </main>

    <footer class="border-t bg-white py-6 text-center text-xs text-slate-400">
      © Kanji Group — Car Dealer
    </footer>

    <!-- modal thông báo sắp mở bán xe (hiện mỗi lần tải trang ở giao diện khách) -->
    <ReleaseCountdownModal />
  </div>
</template>
