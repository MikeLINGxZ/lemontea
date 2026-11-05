import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { fileURLToPath, URL } from 'node:url'
// @ts-ignore
import wails from '@wailsio/runtime/plugins/vite'

export default defineConfig({
  plugins: [react(), tailwindcss(), wails('./bindings')],
  server: {
    host: '127.0.0.1',
  },
  preview: {
    host: '127.0.0.1',
  },
  resolve: {
    dedupe: ['react', 'react-dom'],
    alias: {
      // @ts-ignore
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      // @ts-ignore
      '@bindings': fileURLToPath(new URL('./bindings', import.meta.url)),
    },
  },
  // @ts-ignore
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/__tests__/setup.ts'],
  },
})
