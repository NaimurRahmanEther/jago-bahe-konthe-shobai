import { defineConfig } from 'vitest/config'

// Vitest 2 uses esbuild; Vite 8's production pipeline uses Oxc.
export default defineConfig({
  esbuild: { jsx: 'automatic' },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: './src/test/setup.js',
    css: false,
  },
})
