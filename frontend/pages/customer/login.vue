<script setup lang="ts">
// Đăng nhập đã gộp về 1 trang duy nhất /login — chuyển hướng (giữ tham số redirect).
const route = useRoute()
navigateTo({ path: '/login', query: route.query }, { replace: true })
</script>

<template>
  <div class="py-12 text-center text-sm text-slate-400">Đang chuyển hướng đến trang đăng nhập…</div>
</template>
