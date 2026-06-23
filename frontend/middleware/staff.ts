// Yêu cầu đăng nhập nhân viên (user). Manager/dev tùy trang kiểm tra thêm.
export default defineNuxtRouteMiddleware((to) => {
  const auth = useAuthStore()
  auth.hydrate()
  if (!auth.isUser) {
    return navigateTo(`/login?redirect=${encodeURIComponent(to.fullPath)}`)
  }
})
