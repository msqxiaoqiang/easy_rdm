import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ArcoResolver } from 'unplugin-vue-components/resolvers'

export default defineConfig({
  plugins: [
    vue(),
    AutoImport({
      resolvers: [ArcoResolver()],
    }),
    Components({
      resolvers: [ArcoResolver({ sideEffect: true })],
    }),
  ],
  define: {
    'import.meta.env.VITE_PLATFORM': JSON.stringify(process.env.VITE_PLATFORM || 'gmssh'),
  },
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  base: './', // Important for GMSSH Iframe loading
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
  },
  server: {
    port: 5173,
    proxy: {
      '/dev-api': {
        target: 'http://127.0.0.1:8899',
        rewrite: (path) => path.replace(/^\/dev-api/, '/api'),
      },
    },
    allowedHosts: ['dev.gmssh.com', 'local.gmssh.com'],
    host: '127.0.0.1'
  },
})
