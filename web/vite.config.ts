import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    // Wails webview uses a custom scheme (wails://) to load the app.
    // In dev mode, the Go server reverse-proxies frontend requests to Vite,
    // so Vite must accept requests from the "wails" host.
    allowedHosts: ['wails', 'localhost'],
    proxy: {
      '/api': {
        target: process.env.VITE_API_TARGET ?? 'http://127.0.0.1:8090',
        changeOrigin: true,
      },
    },
  },
})
