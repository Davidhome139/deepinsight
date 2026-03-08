import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': '/src',
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    // Increase chunk size warning limit (default is 500kb)
    chunkSizeWarningLimit: 1000,
    rollupOptions: {
      output: {
        // Manual chunk splitting for better caching and load times
        manualChunks: {
          // Vendor chunks - split by library
          'vendor-vue': ['vue', 'vue-router', 'pinia'],
          'vendor-element': ['element-plus', '@element-plus/icons-vue'],
          'vendor-monaco': ['monaco-editor'],
          'vendor-xterm': ['xterm', 'xterm-addon-attach', 'xterm-addon-fit', 'xterm-addon-web-links'],
          'vendor-utils': ['axios', 'marked', 'highlight.js'],
        },
      },
    },
  },
})
