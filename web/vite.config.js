import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  build: {
    // element-plus is intentionally imported in full; its chunk is ~944 kB minified,
    // so raise the warning threshold above that.
    chunkSizeWarningLimit: 1000,
    rollupOptions: {
      output: {
        manualChunks: {
          vue: ['vue', 'vue-router'],
          'element-plus': ['element-plus'],
          vendor: ['axios', 'dayjs']
        }
      }
    }
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    }
  }
})
