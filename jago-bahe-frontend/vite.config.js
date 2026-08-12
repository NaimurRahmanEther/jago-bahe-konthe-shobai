import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/  ·  vitest reads the `test` block below.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  // Automatic JSX runtime so test files need no `import React` (matches the app).
  esbuild: { jsx: 'automatic' },
  // Dev proxy (F9): the browser calls the API same-origin at /api, and Vite
  // forwards it to the Go backend — so real integration works without the
  // backend needing CORS. Override the target with VITE_API_URL for a remote API.
  server: {
    proxy: {
      '/api': { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: './src/test/setup.js',
    css: false,
  },
})
