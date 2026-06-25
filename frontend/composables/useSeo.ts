// useSeo: đặt thẻ SEO (title, description, Open Graph, Twitter, robots) nhất quán cho mỗi trang.
// Canonical được đặt toàn cục (reactive theo route) ở app.vue nên KHÔNG lặp lại ở đây.
export function useSeo(opts: {
  title?: string
  description?: string
  image?: string // có thể là đường dẫn tương đối ("/x.jpg") hoặc URL tuyệt đối
  noindex?: boolean
}) {
  const site = (useRuntimeConfig().public.siteUrl as string) || 'https://kg-cdl.ddns.net'
  // Chỉ đặt ogImage khi trang có ảnh riêng (vd ảnh xe). Không có -> kế thừa og-cover.jpg mặc định ở app.vue.
  const image = opts.image
    ? opts.image.startsWith('http')
      ? opts.image
      : `${site}${opts.image.startsWith('/') ? '' : '/'}${opts.image}`
    : undefined
  useSeoMeta({
    title: opts.title,
    description: opts.description,
    ogTitle: opts.title,
    ogDescription: opts.description,
    ...(image ? { ogImage: image, twitterImage: image } : {}),
    twitterTitle: opts.title,
    twitterDescription: opts.description,
    robots: opts.noindex ? 'noindex, nofollow' : 'index, follow',
  })
}
