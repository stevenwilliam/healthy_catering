import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// The SPA is served from /app by the Go binary, so the base path matters:
// assets requested from the root would 404 behind that prefix.
export default defineConfig({
  plugins: [react()],
  base: '/app/',
  build: { outDir: 'dist', emptyOutDir: true },
  server: {
    port: 5174,
    // Development proxies to the Go service, so the SPA runs against the real
    // API rather than a mock that drifts from it.
    proxy: { '/api': 'http://127.0.0.1:8081' },
  },
})
