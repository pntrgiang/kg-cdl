<script setup lang="ts">
const auth = useAuthStore()
const username = ref('')
const password = ref('')
const err = ref('')
const loading = ref(false)
const showForgot = ref(false)

async function submit() {
  err.value = ''
  loading.value = true
  try {
    await auth.loginCustomer(username.value, password.value)
    navigateTo((useRoute().query.redirect as string) || '/')
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
        <h1 class="mt-2 font-serif text-xl font-bold text-brand-900">Đăng nhập khách hàng</h1>
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

      <button type="button" class="mt-3 w-full text-center text-xs text-slate-400 hover:text-brand-600 hover:underline" @click="showForgot = !showForgot">
        Quên mật khẩu?
      </button>
      <p v-if="showForgot" class="mt-2 rounded-lg bg-amber-50 px-3 py-2 text-xs text-amber-700">
        Vì lý do bảo mật, vui lòng <strong>liên hệ quản lý Car Dealer</strong> để được đặt lại mật khẩu. Sau khi được đặt lại, bạn đăng nhập bằng mật khẩu mới rồi tự đổi lại trong trang “Tài khoản của tôi”.
      </p>

      <div class="mt-4 text-center text-sm text-slate-500">
        Chưa có tài khoản? <NuxtLink to="/customer/register" class="text-brand-600 hover:underline">Đăng ký</NuxtLink>
      </div>
    </div>
  </div>
</template>
