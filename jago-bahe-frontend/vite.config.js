import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// Application build and dev server; tests are configured in vitest.config.js.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  // Dev proxy (F9): the browser calls the API same-origin at /api, and Vite
  // forwards it to the Go backend â€” so real integration works without the
  // backend needing CORS. Override the target with VITE_API_URL for a remote API.
  server: {
    proxy: {
      '/api': { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
})
