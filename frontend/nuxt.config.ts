// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-01-01',
  devtools: { enabled: true },

  modules: ['@nuxtjs/tailwindcss', '@pinia/nuxt'],

  // Nén sẵn asset tĩnh (JS/CSS/public) lúc build -> server phục vụ bản .br/.gz, giảm dung lượng tải.
  nitro: {
    compressPublicAssets: { gzip: true, brotli: true },
  },

  // Khu vực nhân viên dùng token ở localStorage -> render client-only,
  // tránh redirect sai khi tải lại trang (F5) vì SSR không có token.
  routeRules: {
    '/staff': { ssr: false },
    '/staff/**': { ssr: false },
    '/login': { ssr: false },
    '/customer/**': { ssr: false },
    '/account': { ssr: false },
    '/events/*': { ssr: false }, // chỉ CHI TIẾT sự kiện client-only (auth ở client); danh sách /events vẫn SSR
    // Cache ảnh xe ~30 ngày (file tĩnh theo model_code, nội dung ổn định) -> vào lại trang không tải lại.
    '/vehicles/img/**': { headers: { 'cache-control': 'public, max-age=2592000' } },
    '/vehicles/class/**': { headers: { 'cache-control': 'public, max-age=2592000' } },
  },

  runtimeConfig: {
    public: {
      // URL backend Go; override bằng NUXT_PUBLIC_API_BASE
      apiBase: process.env.NUXT_PUBLIC_API_BASE || 'http://localhost:8080',
      // URL gốc của site (SEO: canonical, og:url, sitemap). Override bằng NUXT_PUBLIC_SITE_URL
      siteUrl: process.env.NUXT_PUBLIC_SITE_URL || 'https://kg-cdl.ddns.net',
      // Mã xác minh Google Search Console (chỉ phần "content" của thẻ meta). Override bằng NUXT_PUBLIC_GOOGLE_VERIFICATION
      googleVerification: process.env.NUXT_PUBLIC_GOOGLE_VERIFICATION || '',
      // Múi giờ hiển thị (IANA) — ghim cố định theo nghiệp vụ, độc lập trình duyệt/máy chủ
      timezone: process.env.NUXT_PUBLIC_TIMEZONE || 'Asia/Ho_Chi_Minh',
    },
  },

  app: {
    head: {
      // title mặc định do titleTemplate trong app.vue đảm nhiệm (khi trang không tự đặt title)
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
      ],
      link: [
        { rel: 'icon', type: 'image/x-icon', href: '/favicon.ico' },
      ],
      htmlAttrs: { lang: 'vi' },
    },
    // Hiệu ứng chuyển trang & layout mượt.
    pageTransition: { name: 'page', mode: 'out-in' },
    layoutTransition: { name: 'layout', mode: 'out-in' },
  },
})
