<script setup lang="ts">
definePageMeta({ layout: 'staff', middleware: 'staff' })
const auth = useAuthStore()
</script>

<template>
  <div>
    <h1 class="mb-1 font-serif text-2xl font-bold text-brand-900">🔐 Tài khoản</h1>
    <p class="mb-6 text-sm text-slate-500">
      {{ auth.displayName }} · {{ ({ dev: 'Dev', manager: 'Quản lý', staff: 'Nhân viên' } as any)[auth.role] }}
    </p>
    <ChangePasswordForm />
  </div>
</template>
