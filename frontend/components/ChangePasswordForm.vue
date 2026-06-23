<script setup lang="ts">
const auth = useAuthStore()
const oldPassword = ref('')
const newPassword = ref('')
const confirm = ref('')
const err = ref('')
const ok = ref('')
const loading = ref(false)

async function submit() {
  err.value = ''
  ok.value = ''
  if (newPassword.value.length < 6) {
    err.value = 'Mật khẩu mới cần tối thiểu 6 ký tự.'
    return
  }
  if (newPassword.value !== confirm.value) {
    err.value = 'Xác nhận mật khẩu không khớp.'
    return
  }
  loading.value = true
  try {
    await auth.changePassword(oldPassword.value, newPassword.value)
    ok.value = 'Đổi mật khẩu thành công.'
    oldPassword.value = newPassword.value = confirm.value = ''
  } catch (e: any) {
    err.value = e?.data?.error || 'Đổi mật khẩu thất bại.'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="card max-w-md p-6">
    <h2 class="mb-4 font-semibold text-brand-900">Đổi mật khẩu</h2>
    <form class="space-y-3" @submit.prevent="submit">
      <div>
        <label class="label">Mật khẩu hiện tại</label>
        <input v-model="oldPassword" type="password" class="input" autocomplete="current-password" />
      </div>
      <div>
        <label class="label">Mật khẩu mới (≥6 ký tự)</label>
        <input v-model="newPassword" type="password" class="input" autocomplete="new-password" />
      </div>
      <div>
        <label class="label">Xác nhận mật khẩu mới</label>
        <input v-model="confirm" type="password" class="input" autocomplete="new-password" />
      </div>
      <p v-if="err" class="text-sm text-red-600">{{ err }}</p>
      <p v-if="ok" class="text-sm text-green-600">{{ ok }}</p>
      <button class="btn-primary w-full" :disabled="loading">{{ loading ? 'Đang xử lý…' : 'Cập nhật mật khẩu' }}</button>
    </form>
  </div>
</template>
