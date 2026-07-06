import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@':                  resolve(__dirname, 'src'),
      '@koris/core':        resolve(__dirname, '../core/index.ts'),
      '@koris/theme':       resolve(__dirname, '../theme/index.ts'),
      '@koris/composables': resolve(__dirname, '../core/composables'),
      '@koris/types':       resolve(__dirname, '../core/types'),
      '@koris/ui':          resolve(__dirname, '../theme/components'),
      '@koris/styles':      resolve(__dirname, '../core/styles'),
    },
  },
  test: {
    environment: 'happy-dom',
    include: ['tests/**/*.test.ts'],
  },
})
