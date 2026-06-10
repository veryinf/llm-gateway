import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { TanStackRouterVite } from '@tanstack/router-plugin/vite'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [
    TanStackRouterVite({ routesDirectory: 'src/routes', generatedRouteTree: 'src/routeTree.gen.ts' }),
    react(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      '@': '/src',
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api/admin': { target: 'http://localhost:3001', changeOrigin: true },
      '/api/stats': { target: 'http://localhost:3001', changeOrigin: true },
      '/api/audit': { target: 'http://localhost:3001', changeOrigin: true },
      '/api/dashboard': { target: 'http://localhost:3001', changeOrigin: true },
      '/api/login': { target: 'http://localhost:3001', changeOrigin: true },
      '/v1': { target: 'http://localhost:3001', changeOrigin: true },
    },
  },
})
