import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  base: '/dashboard/',
  plugins: [vue()],
  build: {
    outDir: 'www',
    emptyOutDir: true
  }
})
