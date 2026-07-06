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
      '@koris/core':          resolve(__dirname, './index.ts'),
      '@koris/composables':   resolve(__dirname, './composables'),
      '@koris/composables/useApi':            resolve(__dirname, './composables/useApi'),
      '@koris/composables/useFormValidation': resolve(__dirname, './composables/useFormValidation'),
      '@koris/composables/useWebSocket':      resolve(__dirname, './composables/useWebSocket'),
      '@koris/composables/useFreshData':      resolve(__dirname, './composables/useFreshData'),
      '@koris/composables/useFormatDate':     resolve(__dirname, './composables/useFormatDate'),
      '@koris/types':                         resolve(__dirname, './types'),
      '@koris/types/components':              resolve(__dirname, './types/components'),
    },
  },
})
