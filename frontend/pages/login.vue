<script setup lang="ts">
const auth = useAuthStore()
const route = useRoute()
const username = ref('')
const password = ref('')
const err = ref('')
const loading = ref(false)

async function submit() {
  err.value = ''
  loading.value = true
  try {
    await auth.loginStaff(username.value, password.value)
    navigateTo((route.query.redirect as string) || '/staff')
  } catch (e: any) {
    err.value = e?.data?.error || 'Đăng nhập thất bại.'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="mx-auto max-w-sm">
    <div class="card p-6">
      <div class="mb-6 text-center">
        <img src="/logo.png" class="mx-auto h-14 w-14 object-contain" alt="Kanji Group" />
        <h1 class="mt-2 font-serif text-xl font-bold text-brand-900">Đăng nhập nhân viên</h1>
        <p class="text-sm text-slate-500">Kanji Group — Car Dealer</p>
      </div>
      <form class="space-y-4" @submit.prevent="submit">
        <div>
          <label class="label">Tài khoản</label>
          <input v-model="username" class="input" autocomplete="username" />
        </div>
        <div>
          <label class="label">Mật khẩu</label>
          <input v-model="password" type="password" class="input" autocomplete="current-password" />
        </div>
        <p v-if="err" class="text-sm text-red-600">{{ err }}</p>
        <button class="btn-primary w-full" :disabled="loading">{{ loading ? 'Đang xử lý…' : 'Đăng nhập' }}</button>
      </form>
      <div class="mt-4 text-center text-sm text-slate-500">
        Là khách hàng? <NuxtLink to="/customer/login" class="text-brand-600 hover:underline">Đăng nhập tại đây</NuxtLink>
      </div>
    </div>
  </div>
</template>
