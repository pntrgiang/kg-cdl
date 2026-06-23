// Khôi phục phiên đăng nhập từ localStorage trước khi app render (client only).
export default defineNuxtPlugin(() => {
  const auth = useAuthStore()
  auth.hydrate()
})
