import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
    base: process.env.KORIS_PORTAL_BASE || '/account/',
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
      // New split (canonical)
      '@koris/core':        resolve(__dirname, '../core'),
      '@koris/theme':       resolve(__dirname, '../theme'),
      // Backward-compat (old names) — safe to remove once all imports are migrated
      '@koris/composables': resolve(__dirname, '../core/composables'),
      '@koris/types':       resolve(__dirname, '../core/types'),
      '@koris/styles':      resolve(__dirname, '../core/styles'),
      '@koris/ui':          resolve(__dirname, '../theme/components'),
    }
  },
  build: {
    outDir: 'www',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks: {
          'vendor': ['vue', 'vue-router', 'pinia'],
        }
      }
    }
  }
})
