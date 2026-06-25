<script setup lang="ts">
// Head & SEO mặc định toàn site. Mỗi trang khách có thể ghi đè bằng useSeo().
const cfg = useRuntimeConfig()
const site = (cfg.public.siteUrl as string) || 'https://kg-cdl.ddns.net'
const gsc = (cfg.public.googleVerification as string) || ''
const route = useRoute()

useHead({
  // "Tên trang · Kanji Group"; nếu trang không đặt title -> dùng tiêu đề mặc định.
  titleTemplate: (t) => (t ? `${t} · Kanji Group` : 'Kanji Group — Đại lý xe Lux City'),
  // canonical reactive theo route (đặt 1 chỗ, không lặp ở từng trang)
  link: [{ rel: 'canonical', href: () => site + route.path }],
  meta: [
    { name: 'theme-color', content: '#3b2a78' },
    // Xác minh quyền sở hữu cho Google Search Console (chỉ hiện khi đã cấu hình token)
    ...(gsc ? [{ name: 'google-site-verification', content: gsc }] : []),
  ],
})

useSeoMeta({
  description:
    'Đại lý xe Kanji Group tại thành phố Lux City — bảng giá xe đang mở bán, xe sắp ra mắt, ưu đãi giảm giá và sự kiện quay số trúng thưởng dành cho khách hàng.',
  ogType: 'website',
  ogSiteName: 'Kanji Group — Car Dealer',
  ogLocale: 'vi_VN',
  ogImage: `${site}/og-cover.jpg`,
  ogImageWidth: 1200,
  ogImageHeight: 630,
  ogImageAlt: 'Kanji Group — Đại lý xe Lux City',
  twitterCard: 'summary_large_image',
})
</script>

<template>
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
</template>
