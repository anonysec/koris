import { defineConfig } from 'vitest/config'
import { resolve } from 'path'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  test: {
    environment: 'happy-dom',
    globals: true,
    include: ['**/*.{test,spec}.{ts,tsx}'],
    exclude: ['node_modules/**', 'dist/**'],
  },
  resolve: {
    alias: {
      '@koris/theme':         resolve(__dirname, './index.ts'),
      '@koris/ui':            resolve(__dirname, './components'),
      '@koris/core':          resolve(__dirname, '../core/index.ts'),
      '@koris/composables':   resolve(__dirname, '../core/composables'),
      '@koris/types':         resolve(__dirname, '../core/types'),
      '@koris/types/components': resolve(__dirname, '../core/types/components'),
      '@koris/styles':        resolve(__dirname, '../core/styles'),
    },
  },
})
