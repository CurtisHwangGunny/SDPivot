import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: { '@': resolve(__dirname, 'src') },
  },
  server: {
    port: 3099,
    host: '0.0.0.0',
    proxy: {
      '/api': {
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
    },
  },
  // Multi-page entry: user app + ops admin app
  build: {
    chunkSizeWarningLimit: 1200,
    rollupOptions: {
      input: {
        main: resolve(__dirname, 'index.html'),
        ops: resolve(__dirname, 'ops.html'),
      },
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) return undefined
          if (id.includes('tdesign-vue-next')) return 'tdesign-vendor'
          if (id.includes('vue-router')) return 'router-vendor'
          if (id.includes('pinia')) return 'pinia-vendor'
          if (id.includes('axios')) return 'axios-vendor'
          if (id.includes('/vue/')) return 'vue-core'
          return 'app-vendor'
        },
      },
    },
  },
})
