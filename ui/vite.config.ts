import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

const apiPort = process.env.HTTP_PORT || '8093'

export default defineConfig({
  plugins: [vue()],
  base: '/microscope/',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: {
      '/microscope/api': {
        target: `http://127.0.0.1:${apiPort}`,
        changeOrigin: true,
      },
    },
  },
})
