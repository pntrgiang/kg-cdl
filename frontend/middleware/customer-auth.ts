// Yêu cầu đã đăng nhập để tham gia / xem chi tiết sự kiện.
export default defineNuxtRouteMiddleware((to) => {
  const auth = useAuthStore()
  auth.hydrate()
  if (!auth.isAuthed) {
    return navigateTo(`/customer/login?redirect=${encodeURIComponent(to.fullPath)}`)
  }
})
