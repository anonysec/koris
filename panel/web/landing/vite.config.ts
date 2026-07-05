import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  base: '/',
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
    // Enable CSS code splitting for async chunks
    cssCodeSplit: true,
    // Ensure CSS is minified
    cssMinify: true,
    // Minify JS (esbuild is the Vite 5 default, no terser needed)
    minify: 'esbuild',
    // Split vendor (vue runtime) from application code
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['vue'],
        },
      },
    },
  },
})
