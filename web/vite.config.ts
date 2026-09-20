import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import { fileURLToPath } from 'node:url'

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  
  return {
    plugins: [react()],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    server: {
      proxy: {
        '/api': {
          target: env.VITE_API_BASE || 'http://localhost:8080',
          changeOrigin: true,
        },
        '/auth': {
          target: env.VITE_API_BASE || 'http://localhost:8080',
          changeOrigin: true,
        },
      },
    },
    build: {
      rollupOptions: {
        output: {
          // The tunnel's ingress rules send any path containing "/api" to the gateway, and a rule such as
          // /api/* also matches "/assets/api-<hash>.js". The prefix keeps every file name from starting with
          // "api", so scripts are always served by the frontend.
          entryFileNames: 'assets/app-[name]-[hash].js',
          chunkFileNames: 'assets/chunk-[name]-[hash].js',
        },
      },
    },
    define: {
      // Make environment variables available in the browser
      'import.meta.env.VITE_API_BASE': JSON.stringify(env.VITE_API_BASE || ''),
    },
  }
})
