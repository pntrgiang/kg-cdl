// Chỉ quản lý (manager) hoặc dev.
export default defineNuxtRouteMiddleware((to) => {
  const auth = useAuthStore()
  auth.hydrate()
  if (!auth.isUser) return navigateTo(`/login?redirect=${encodeURIComponent(to.fullPath)}`)
  if (!auth.isManager) return navigateTo('/staff')
})
