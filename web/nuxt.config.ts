export default defineNuxtConfig({
  ssr: false,
  devtools: { enabled: false },
  app: {
    head: {
      title: 'gores — Go 二进制还原分析',
      meta: [{ name: 'viewport', content: 'width=device-width, initial-scale=1' }],
    },
  },
  nitro: {
    devProxy: {
      '/api': { target: 'http://localhost:8080/api', changeOrigin: true },
    },
  },
  compatibilityDate: '2024-11-01',
})
