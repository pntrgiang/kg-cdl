// Chỉ dev.
export default defineNuxtRouteMiddleware((to) => {
  const auth = useAuthStore()
  auth.hydrate()
  if (!auth.isUser) return navigateTo(`/login?redirect=${encodeURIComponent(to.fullPath)}`)
  if (!auth.isDev) return navigateTo('/staff')
})
