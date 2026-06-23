import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  base: '/',
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
      '@koris/ui': resolve(__dirname, '../shared/components'),
      '@koris/composables': resolve(__dirname, '../shared/composables'),
      '@koris/types': resolve(__dirname, '../shared/types'),
      '@koris/styles': resolve(__dirname, '../shared/styles'),
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
