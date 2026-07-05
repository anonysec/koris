import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
      '@koris/ui': resolve(__dirname, '../shared/components'),
      '@koris/composables': resolve(__dirname, '../shared/composables'),
      '@koris/types': resolve(__dirname, '../shared/types'),
      '@koris/styles': resolve(__dirname, '../shared/styles'),
    },
  },
  test: {
    environment: 'happy-dom',
    include: ['tests/**/*.test.ts'],
  },
})
