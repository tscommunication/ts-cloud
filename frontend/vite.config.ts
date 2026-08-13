import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],

  build: {
    target: 'es2020',
    sourcemap: false,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) return

          if (/[\\/]node_modules[\\/](react|react-dom|react-router|react-router-dom)[\\/]/.test(id)) {
            return 'react'
          }
          if (/[\\/]node_modules[\\/](@mui|@emotion)[\\/]/.test(id)) return 'mui'
          if (/[\\/]node_modules[\\/](@tanstack|axios)[\\/]/.test(id)) return 'query'
          if (/[\\/]node_modules[\\/]recharts[\\/]/.test(id)) return 'charts'
          if (/[\\/]node_modules[\\/]react-hook-form[\\/]/.test(id)) return 'forms'
          if (/[\\/]node_modules[\\/]date-fns[\\/]/.test(id)) return 'dates'
        },
      },
    },
  },

  server: {
    host: '0.0.0.0',
    port: 5173,

    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
})
