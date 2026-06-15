import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  base: '/portal/',
  plugins: [vue()],
  build: {
    outDir: 'www',
    emptyOutDir: true
  }
})
