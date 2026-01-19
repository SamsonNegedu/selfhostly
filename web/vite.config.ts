import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  
  return {
    plugins: [react()],
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
      },
    },
    server: {
      proxy: {
        '/api': {
          target: env.VITE_API_BASE || 'http://localhost:8080',
          changeOrigin: true,
        },
        // Proxy GoBetterAuth requests to avoid CORS issues during development
        '/auth': {
          target: env.VITE_API_BASE || 'http://localhost:8080',
          changeOrigin: true,
        },
      },
    },
    define: {
      // Make environment variables available in the browser
      'import.meta.env.VITE_API_BASE': JSON.stringify(env.VITE_API_BASE || ''),
    },
  }
})
